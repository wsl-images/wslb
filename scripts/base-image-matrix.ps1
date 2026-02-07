param(
    [string]$Out = "base-image-matrix-report.json"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$prev = Get-Location
Set-Location $repoRoot
try {
    $env:WSLB_BASE_IMAGE_MATRIX = "1"
    $tmpLog = [System.IO.Path]::GetTempFileName()
    try {
        & go test -json ./internal/targets/wsl -run TestBaseImageMatrixWSLPrereqsAndOOBE 2>&1 | Tee-Object -FilePath $tmpLog | Out-Host
        $exitCode = $LASTEXITCODE
    } finally {
        Remove-Item Env:WSLB_BASE_IMAGE_MATRIX -ErrorAction SilentlyContinue
    }

    $lines = Get-Content -Path $tmpLog -ErrorAction SilentlyContinue
    $results = @{}
    foreach ($line in $lines) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $obj = $null
        try {
            $obj = $line | ConvertFrom-Json -ErrorAction Stop
        } catch {
            continue
        }
        if (-not $obj.Test) { continue }
        if (-not $obj.Test.StartsWith("TestBaseImageMatrixWSLPrereqsAndOOBE/")) { continue }
        $image = $obj.Test.Substring("TestBaseImageMatrixWSLPrereqsAndOOBE/".Length)
        if (-not $results.ContainsKey($image)) {
            $results[$image] = [ordered]@{
                image = $image
                status = "running"
                elapsed = 0
                output = @()
            }
        }
        if ($obj.Action -eq "output" -and $obj.Output) {
            $results[$image].output += $obj.Output.TrimEnd()
        }
        if ($obj.Action -eq "pass" -or $obj.Action -eq "fail" -or $obj.Action -eq "skip") {
            $results[$image].status = $obj.Action
            if ($null -ne $obj.Elapsed) {
                $results[$image].elapsed = [double]$obj.Elapsed
            }
        }
    }

    $orderedResults = $results.Values | Sort-Object image
    $report = [ordered]@{
        ok = ($exitCode -eq 0)
        exitCode = $exitCode
        generatedAt = (Get-Date).ToUniversalTime().ToString("o")
        total = @($orderedResults).Count
        passed = @($orderedResults | Where-Object { $_.status -eq "pass" }).Count
        failed = @($orderedResults | Where-Object { $_.status -eq "fail" }).Count
        skipped = @($orderedResults | Where-Object { $_.status -eq "skip" }).Count
        results = @($orderedResults)
    }

    if (-not [System.IO.Path]::IsPathRooted($Out)) {
        $Out = Join-Path $repoRoot $Out
    }
    $outDir = Split-Path -Parent $Out
    if ($outDir -and -not (Test-Path $outDir)) {
        New-Item -ItemType Directory -Path $outDir -Force | Out-Null
    }
    $report | ConvertTo-Json -Depth 8 | Set-Content -Path $Out -Encoding UTF8
    Write-Host "Base image matrix report: $Out"

    if ($exitCode -ne 0) {
        exit $exitCode
    }
} finally {
    Set-Location $prev
    if ($tmpLog -and (Test-Path $tmpLog)) {
        Remove-Item -Force $tmpLog -ErrorAction SilentlyContinue
    }
}
