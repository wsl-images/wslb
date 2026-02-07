param(
    [switch]$Json,
    [string]$Out = "e2e-report.json",
    [switch]$CleanupStateDisk,
    [switch]$Strict
)

$ErrorActionPreference = "Stop"

function Invoke-ProcessCapture {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [string[]]$Arguments = @(),
        [string]$WorkingDirectory = (Get-Location).Path,
        [string]$InputText = "",
        [int]$TimeoutMs = 900000
    )
    $stdoutFile = [System.IO.Path]::GetTempFileName()
    $stderrFile = [System.IO.Path]::GetTempFileName()
    $stdinFile = $null
    try {
        if (-not [string]::IsNullOrEmpty($InputText)) {
            $stdinFile = [System.IO.Path]::GetTempFileName()
            Set-Content -Path $stdinFile -Value $InputText -NoNewline -Encoding UTF8
        }
        $startArgs = @{
            FilePath               = $FilePath
            ArgumentList           = $Arguments
            PassThru               = $true
            Wait                   = $true
            NoNewWindow            = $true
            WorkingDirectory       = $WorkingDirectory
            RedirectStandardOutput = $stdoutFile
            RedirectStandardError  = $stderrFile
        }
        if ($stdinFile) {
            $startArgs.RedirectStandardInput = $stdinFile
        }
        $p = Start-Process @startArgs
        $stdout = Get-Content -Raw -Path $stdoutFile -ErrorAction SilentlyContinue
        $stderr = Get-Content -Raw -Path $stderrFile -ErrorAction SilentlyContinue
        return [pscustomobject]@{
            ExitCode = $p.ExitCode
            Stdout   = if ($null -ne $stdout) { $stdout } else { "" }
            Stderr   = if ($null -ne $stderr) { $stderr } else { "" }
        }
    } finally {
        Remove-Item -Force -ErrorAction SilentlyContinue $stdoutFile, $stderrFile
        if ($stdinFile) {
            Remove-Item -Force -ErrorAction SilentlyContinue $stdinFile
        }
    }
}

function Add-StepResult {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Action,
        [switch]$Critical = $true
    )
    $start = (Get-Date).ToUniversalTime()
    $step = [ordered]@{
        name      = $Name
        status    = "passed"
        critical  = [bool]$Critical
        startedAt = $start.ToString("o")
        endedAt   = $null
        message   = ""
    }
    try {
        & $Action
    } catch {
        $step.status = "failed"
        $step.message = $_.Exception.Message
    }
    $step.endedAt = (Get-Date).ToUniversalTime().ToString("o")
    $script:Report.steps += [pscustomobject]$step
}

function Add-SkippedStep {
    param(
        [string]$Name,
        [string]$Reason,
        [switch]$Critical = $true
    )
    $now = (Get-Date).ToUniversalTime().ToString("o")
    $script:Report.steps += [pscustomobject]@{
        name      = $Name
        status    = "skipped"
        critical  = [bool]$Critical
        startedAt = $now
        endedAt   = $now
        message   = $Reason
    }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$tmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("wslb-e2e-" + [Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmpRoot | Out-Null

$stamp = Get-Date -Format "yyyyMMddHHmmss"
$workspacePath = Join-Path $tmpRoot ".devcontainer\devcontainer.json"
$outputDir = Join-Path (Split-Path -Parent $workspacePath) ".wslb-out"
$stateVhd = Join-Path $tmpRoot "state\home.vhdx"
$imageID = "e2e-wsl"
$distroName = "wslb-e2e-$stamp"
$candidateName = "$distroName`__candidate"
$sentinel = [Guid]::NewGuid().ToString("N")
$wslbExe = Join-Path $tmpRoot "wslb.exe"

$script:Report = [ordered]@{
    ok        = $true
    startedAt = (Get-Date).ToUniversalTime().ToString("o")
    endedAt   = $null
    workspace = $workspacePath
    steps     = @()
}

try {
    New-Item -ItemType Directory -Path (Join-Path $tmpRoot ".devcontainer"), (Join-Path $tmpRoot "state"), (Join-Path $tmpRoot "distros") | Out-Null

    Add-StepResult -Name "build-wslb" {
        $r = Invoke-ProcessCapture -FilePath "go" -Arguments @("build", "-o", $wslbExe, ".") -WorkingDirectory $repoRoot
        if ($r.ExitCode -ne 0) {
            throw "go build failed: $($r.Stderr)`n$($r.Stdout)"
        }
    }

    $manifest = @"
{
  "name": "wslb-e2e",
  "image": "ubuntu:24.04",
  "features": {
    "ghcr.io/devcontainers/features/common-utils:2": {
      "installZsh": "false",
      "upgradePackages": "false"
    }
  },
  "remoteUser": "dev",
  "wslb": {
    "version": 1,
    "id": "$imageID",
    "displayName": "E2E Ubuntu",
    "description": "E2E managed image",
    "distroName": "$distroName",
    "managed": true,
    "state": {
      "mode": "windows-dir",
      "path": "./state/home",
      "mountPoint": "/home"
    },
    "wslconf": {
      "boot": { "systemd": true },
      "automount": { "enabled": true, "mountFsTab": true, "options": "metadata" }
    },
    "distribution": {
      "shortcut": {
        "enabled": true,
        "icon": { "simpleIcon": "ubuntu", "color": "E95420", "style": "flat" }
      }
    }
  }
}
"@
    Set-Content -Path $workspacePath -Value $manifest -Encoding UTF8

    $script:DoctorResult = $null
    Add-StepResult -Name "doctor" {
        $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "doctor") -WorkingDirectory $tmpRoot
        if ($r.ExitCode -ne 0) {
            throw "doctor failed: $($r.Stdout)`n$($r.Stderr)"
        }
        $script:DoctorResult = $r.Stdout | ConvertFrom-Json
    }

    Add-StepResult -Name "workspace-validate" {
        $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "workspace", "validate", "--workspace-file", $workspacePath) -WorkingDirectory $tmpRoot
        if ($r.ExitCode -ne 0) {
            throw "workspace validate failed: $($r.Stdout)`n$($r.Stderr)"
        }
    }

    $doctorResult = $script:DoctorResult
    $canRunWSL = $true
    if ($null -ne $doctorResult) {
        $wslCheck = $doctorResult.checks | Where-Object { $_.id -eq "wsl.version" } | Select-Object -First 1
        if ($null -eq $wslCheck -or $wslCheck.status -eq "fail") { $canRunWSL = $false }

        $engineChecks = @($doctorResult.checks | Where-Object { $_.id -like "engine.*" })
        $hasReadyEngine = @($engineChecks | Where-Object { $_.status -eq "ok" }).Count -gt 0
        if (-not $hasReadyEngine) { $canRunWSL = $false }
    }

    if (-not $canRunWSL) {
        $reason = "Missing required WSL prerequisites from doctor output"
        if ($Strict) { throw $reason }
        Add-SkippedStep -Name "wsl-state-create" -Reason $reason
        Add-SkippedStep -Name "wsl-install" -Reason $reason
        Add-SkippedStep -Name "wsl-upgrade" -Reason $reason
        Add-SkippedStep -Name "wsl-rollback" -Reason $reason
    } else {
        Add-StepResult -Name "wsl-state-create" {
            $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "wsl", "--workspace-file", $workspacePath, "--non-interactive", "--fallback-windows-dir", "state", "create", $imageID) -WorkingDirectory $tmpRoot
            if ($r.ExitCode -ne 0) {
                throw "wsl state create failed: $($r.Stdout)`n$($r.Stderr)"
            }
        }

        Add-StepResult -Name "wsl-install" {
            $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "wsl", "--workspace-file", $workspacePath, "--non-interactive", "--fallback-windows-dir", "install", $imageID) -WorkingDirectory $tmpRoot
            if ($r.ExitCode -ne 0) {
                throw "wsl install failed: $($r.Stdout)`n$($r.Stderr)"
            }
        }

        Add-StepResult -Name "wsl-sentinel-write" {
            $mk = Invoke-ProcessCapture -FilePath "wsl" -Arguments @("-d", $distroName, "-u", "root", "--", "mkdir", "-p", "/home/dev")
            if ($mk.ExitCode -ne 0) {
                throw "sentinel mkdir failed: $($mk.Stdout)`n$($mk.Stderr)"
            }
            $wr = Invoke-ProcessCapture -FilePath "wsl" -Arguments @("-d", $distroName, "-u", "root", "--", "tee", "/home/dev/keep.txt") -InputText ($sentinel + "`n")
            if ($wr.ExitCode -ne 0) {
                throw "sentinel write failed: $($wr.Stdout)`n$($wr.Stderr)"
            }
            $rd = Invoke-ProcessCapture -FilePath "wsl" -Arguments @("-d", $distroName, "-u", "root", "--", "cat", "/home/dev/keep.txt")
            if ($rd.ExitCode -ne 0) {
                throw "sentinel read-after-write failed: $($rd.Stdout)`n$($rd.Stderr)"
            }
            if (($rd.Stdout).Trim() -ne $sentinel) {
                throw "sentinel verification failed after write. expected=$sentinel actual=$(($rd.Stdout).Trim())"
            }
        }

        Add-StepResult -Name "workspace-mutate-feature" {
            $mutated = Get-Content -Raw -Path $workspacePath
            $mutated = $mutated -replace '"description": "E2E managed image"', '"description": "E2E managed image upgraded"'
            Set-Content -Path $workspacePath -Value $mutated -Encoding UTF8
        }

        Add-StepResult -Name "wsl-upgrade" {
            $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "wsl", "--workspace-file", $workspacePath, "--non-interactive", "--fallback-windows-dir", "upgrade", $imageID) -WorkingDirectory $tmpRoot
            if ($r.ExitCode -ne 0) {
                throw "wsl upgrade failed: $($r.Stdout)`n$($r.Stderr)"
            }
        }

        Add-StepResult -Name "wsl-sentinel-verify-after-upgrade" {
            $r = Invoke-ProcessCapture -FilePath "wsl" -Arguments @("-d", $distroName, "-u", "root", "--", "cat", "/home/dev/keep.txt")
            if ($r.ExitCode -ne 0) {
                throw "sentinel read failed after upgrade: $($r.Stdout)`n$($r.Stderr)"
            }
            $actual = ($r.Stdout).Trim()
            if ($actual -ne $sentinel) {
                throw "sentinel mismatch after upgrade. expected=$sentinel actual=$actual"
            }
        }

        $script:RollbackVersionId = $null
        Add-StepResult -Name "wsl-find-previous-version" {
            $versionsPath = Join-Path $outputDir "state\$imageID\versions.json"
            if (-not (Test-Path $versionsPath)) {
                throw "versions metadata not found: $versionsPath"
            }
            $versions = Get-Content -Raw -Path $versionsPath | ConvertFrom-Json
            if ($versions.versions.Count -lt 1) {
                throw "no version records found"
            }
            $script:RollbackVersionId = $versions.versions[-1].versionId
            if ([string]::IsNullOrWhiteSpace($script:RollbackVersionId)) {
                throw "versionId missing in versions metadata"
            }
        }

        Add-StepResult -Name "wsl-rollback" {
            $r = Invoke-ProcessCapture -FilePath $wslbExe -Arguments @("--json", "wsl", "--workspace-file", $workspacePath, "--non-interactive", "--fallback-windows-dir", "rollback", $imageID, "--to", $script:RollbackVersionId) -WorkingDirectory $tmpRoot
            if ($r.ExitCode -ne 0) {
                throw "wsl rollback failed: $($r.Stdout)`n$($r.Stderr)"
            }
        }

        Add-StepResult -Name "wsl-sentinel-verify-after-rollback" {
            $r = Invoke-ProcessCapture -FilePath "wsl" -Arguments @("-d", $distroName, "-u", "root", "--", "cat", "/home/dev/keep.txt")
            if ($r.ExitCode -ne 0) {
                throw "sentinel read failed after rollback: $($r.Stdout)`n$($r.Stderr)"
            }
            $actual = ($r.Stdout).Trim()
            if ($actual -ne $sentinel) {
                throw "sentinel mismatch after rollback. expected=$sentinel actual=$actual"
            }
        }
    }

}
finally {
    $Report.endedAt = (Get-Date).ToUniversalTime().ToString("o")

    try { Invoke-ProcessCapture -FilePath "wsl" -Arguments @("--unregister", $candidateName) | Out-Null } catch {}
    try { Invoke-ProcessCapture -FilePath "wsl" -Arguments @("--unregister", $distroName) | Out-Null } catch {}
    try { Invoke-ProcessCapture -FilePath "wsl" -Arguments @("--unmount", $stateVhd) | Out-Null } catch {}

    if ($CleanupStateDisk) {
        try { Remove-Item -Recurse -Force -ErrorAction SilentlyContinue (Join-Path $tmpRoot "state") } catch {}
    }

    $criticalFailures = @($Report.steps | Where-Object { $_.critical -and $_.status -eq "failed" }).Count
    $Report.ok = ($criticalFailures -eq 0)

    if (-not [System.IO.Path]::IsPathRooted($Out)) {
        $Out = Join-Path (Get-Location) $Out
    }
    $outParent = Split-Path -Parent $Out
    if ($outParent -and -not (Test-Path $outParent)) {
        New-Item -ItemType Directory -Path $outParent -Force | Out-Null
    }
    $Report | ConvertTo-Json -Depth 8 | Set-Content -Path $Out -Encoding UTF8

    if ($Json) {
        $Report | ConvertTo-Json -Depth 8
    } else {
        Write-Host "E2E report: $Out"
        Write-Host "ok: $($Report.ok)"
    }

    if (-not $Report.ok) {
        exit 1
    }
}
