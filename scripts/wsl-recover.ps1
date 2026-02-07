param(
    [switch]$Deep,
    [switch]$AutoElevate,
    [int]$TimeoutSec = 20,
    [string]$LogPath = $(Join-Path (Get-Location) ("wsl-recover-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".log"))
)

$ErrorActionPreference = "Continue"

function Write-Log {
    param([string]$Message)
    $line = "$(Get-Date -Format o)  $Message"
    Write-Host $line
    Add-Content -Path $LogPath -Value $line
}

function Invoke-WithTimeout {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [string[]]$Arguments = @(),
        [int]$TimeoutSeconds = 20
    )

    $stdoutFile = [System.IO.Path]::GetTempFileName()
    $stderrFile = [System.IO.Path]::GetTempFileName()
    try {
        $proc = Start-Process -FilePath $FilePath -ArgumentList $Arguments -PassThru -NoNewWindow -RedirectStandardOutput $stdoutFile -RedirectStandardError $stderrFile
        $timedOut = $false
        if (-not $proc.WaitForExit($TimeoutSeconds * 1000)) {
            $timedOut = $true
            try { Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue } catch {}
        }
        $stdout = Get-Content -Raw -Path $stdoutFile -ErrorAction SilentlyContinue
        $stderr = Get-Content -Raw -Path $stderrFile -ErrorAction SilentlyContinue
        [pscustomobject]@{
            ExitCode = if ($timedOut) { -999 } else { $proc.ExitCode }
            TimedOut = $timedOut
            Stdout   = if ($null -ne $stdout) { $stdout.Trim() } else { "" }
            Stderr   = if ($null -ne $stderr) { $stderr.Trim() } else { "" }
            Command  = "$FilePath $($Arguments -join ' ')"
        }
    } finally {
        Remove-Item -Force -ErrorAction SilentlyContinue $stdoutFile, $stderrFile
    }
}

function Test-IsAdmin {
    try {
        $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
        $principal = New-Object Security.Principal.WindowsPrincipal($identity)
        return $principal.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)
    } catch {
        return $false
    }
}

function Run-WSL {
    param([string[]]$Arguments)
    Invoke-WithTimeout -FilePath "wsl.exe" -Arguments $Arguments -TimeoutSeconds $TimeoutSec
}

Write-Log "WSL recovery started"
Write-Log "Log file: $LogPath"
Write-Log "Deep mode: $Deep"
Write-Log "Auto-elevate: $AutoElevate"

if ($Deep -and -not (Test-IsAdmin)) {
    if ($AutoElevate) {
        Write-Log "Deep mode requested without admin rights. Relaunching elevated..."
        $args = @(
            "-NoProfile",
            "-ExecutionPolicy", "Bypass",
            "-File", ('"{0}"' -f $PSCommandPath),
            "-Deep",
            "-TimeoutSec", "$TimeoutSec",
            "-LogPath", ('"{0}"' -f $LogPath)
        )
        Start-Process -FilePath "pwsh.exe" -Verb RunAs -ArgumentList ($args -join " ")
        exit 0
    }
    Write-Log "Deep mode requires an elevated shell."
    Write-Log "Rerun as admin: pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/wsl-recover.ps1 -Deep"
    Write-Log "Or auto-elevate: pwsh -NoProfile -ExecutionPolicy Bypass -File scripts/wsl-recover.ps1 -Deep -AutoElevate"
    exit 2
}

$pre = Get-Process -Name wsl,wslhost,wslservice,wslrelay,vmmem,vmmemWSL -ErrorAction SilentlyContinue
Write-Log ("Pre-recovery process count: " + @($pre).Count)

$shutdown = Run-WSL -Arguments @("--shutdown")
if ($shutdown.TimedOut) {
    Write-Log "wsl --shutdown timed out"
} else {
    Write-Log "wsl --shutdown exit=$($shutdown.ExitCode)"
}
if ($shutdown.Stderr) { Write-Log ("stderr: " + $shutdown.Stderr) }

$unmount = Run-WSL -Arguments @("--unmount")
if ($unmount.TimedOut) {
    Write-Log "wsl --unmount timed out"
} else {
    Write-Log "wsl --unmount exit=$($unmount.ExitCode)"
}
if ($unmount.Stderr) { Write-Log ("stderr: " + $unmount.Stderr) }

$targets = @("wsl", "wslhost", "wslservice", "wslrelay", "vmmem", "vmmemWSL")
foreach ($name in $targets) {
    $procs = Get-Process -Name $name -ErrorAction SilentlyContinue
    if (-not $procs) { continue }
    foreach ($p in $procs) {
        try {
            Stop-Process -Id $p.Id -Force -ErrorAction Stop
            Write-Log "Stopped process $($p.ProcessName) pid=$($p.Id)"
        } catch {
            Write-Log "Failed to stop process $($p.ProcessName) pid=$($p.Id): $($_.Exception.Message)"
        }
    }
}

if ($Deep) {
    Write-Log "Deep mode: forcing wslservice/vmmemWSL stop and restarting vmcompute"
    $kills = @(
        @("taskkill.exe", @("/F", "/IM", "wslservice.exe", "/T")),
        @("taskkill.exe", @("/F", "/IM", "vmmemWSL.exe", "/T"))
    )
    foreach ($k in $kills) {
        $r = Invoke-WithTimeout -FilePath $k[0] -Arguments $k[1] -TimeoutSeconds $TimeoutSec
        Write-Log "deep kill [$($r.Command)] exit=$($r.ExitCode) timeout=$($r.TimedOut)"
        if ($r.Stdout) { Write-Log ("stdout: " + $r.Stdout) }
        if ($r.Stderr) { Write-Log ("stderr: " + $r.Stderr) }
    }

    try {
        Restart-Service -Name "vmcompute" -Force -ErrorAction Stop
        Write-Log "Restarted vmcompute service"
    } catch {
        Write-Log "Failed to restart vmcompute service: $($_.Exception.Message)"
        Write-Log "If not running as admin, rerun in elevated PowerShell with -Deep."
    }
    $shutdown2 = Run-WSL -Arguments @("--shutdown")
    Write-Log "post-deep wsl --shutdown timedOut=$($shutdown2.TimedOut) exit=$($shutdown2.ExitCode)"
}

Start-Sleep -Seconds 2

$status = Run-WSL -Arguments @("--status")
$list = Run-WSL -Arguments @("-l", "-q")
$version = Run-WSL -Arguments @("--version")

Write-Log "Health check: wsl --status timedOut=$($status.TimedOut) exit=$($status.ExitCode)"
if ($status.Stdout) { Write-Log ("status stdout: " + $status.Stdout) }
if ($status.Stderr) { Write-Log ("status stderr: " + $status.Stderr) }

Write-Log "Health check: wsl -l -q timedOut=$($list.TimedOut) exit=$($list.ExitCode)"
if ($list.Stdout) { Write-Log ("list stdout: " + $list.Stdout) }
if ($list.Stderr) { Write-Log ("list stderr: " + $list.Stderr) }

Write-Log "Health check: wsl --version timedOut=$($version.TimedOut) exit=$($version.ExitCode)"
if ($version.Stdout) { Write-Log ("version stdout: " + $version.Stdout) }
if ($version.Stderr) { Write-Log ("version stderr: " + $version.Stderr) }

$post = Get-Process -Name wsl,wslhost,wslservice,wslrelay,vmmem,vmmemWSL -ErrorAction SilentlyContinue
Write-Log ("Post-recovery process count: " + @($post).Count)

$healthy = (-not $status.TimedOut) -and (-not $list.TimedOut)
if ($healthy) {
    Write-Log "WSL recovery completed successfully"
    exit 0
}

Write-Log "WSL recovery completed but health checks still timed out"
Write-Log "Next step: run elevated PowerShell: pwsh -File scripts/wsl-recover.ps1 -Deep"
exit 1
