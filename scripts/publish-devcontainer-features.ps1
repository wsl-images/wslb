param(
  [string]$Org = "wsl-images",
  [string]$Repo = "devcontainer-features",
  [switch]$Private,
  [switch]$SkipPush
)

$ErrorActionPreference = "Stop"

function Resolve-GhExe {
  $candidates = @()
  if ($env:WSLB_GH_EXE) { $candidates += $env:WSLB_GH_EXE }
  try {
    $cmd = Get-Command gh -ErrorAction Stop
    if ($cmd -and $cmd.Path) { $candidates += $cmd.Path }
  } catch {}
  $candidates += "C:\Program Files\GitHub CLI\gh.exe"
  $candidates += "C:\Users\$env:USERNAME\AppData\Local\Programs\GitHub CLI\gh.exe"

  foreach ($c in $candidates) {
    if ($c -and (Test-Path $c)) { return $c }
  }
  throw "GitHub CLI not found. Install gh or set WSLB_GH_EXE."
}

function Exec([string]$File, [string[]]$Arguments, [string]$WorkDir = "") {
  if ($WorkDir) {
    Push-Location $WorkDir
  }
  try {
    & $File @Arguments
    if ($LASTEXITCODE -ne 0) {
      throw "Command failed: $File $($Arguments -join ' ')"
    }
  } finally {
    if ($WorkDir) {
      Pop-Location
    }
  }
}

$repoRef = "$Org/$Repo"
$gh = Resolve-GhExe
$sourceDir = Join-Path (Split-Path -Parent $PSScriptRoot) "devcontainer-features"

if (!(Test-Path $sourceDir)) {
  throw "Feature source directory not found: $sourceDir"
}

Write-Host "[publish] gh = $gh"
Write-Host "[publish] repo = $repoRef"
Exec $gh @("auth", "status")

$repoExists = $true
try {
  & $gh api "repos/$repoRef" | Out-Null
  if ($LASTEXITCODE -ne 0) { throw "repo not found" }
} catch {
  $repoExists = $false
}

if (-not $repoExists) {
  $visibility = if ($Private) { "--private" } else { "--public" }
  Write-Host "[publish] creating repository $repoRef ($visibility)"
  Exec $gh @("repo", "create", $repoRef, $visibility, "--description", "Official wsl-images Dev Container Features", "-y")
}

$tmp = Join-Path $env:TEMP ("wslb-devcontainer-features-" + [Guid]::NewGuid().ToString("n"))
Write-Host "[publish] cloning into $tmp"
Exec $gh @("repo", "clone", $repoRef, $tmp)

try {
  Get-ChildItem -Path $tmp -Force | Where-Object { $_.Name -ne ".git" } | Remove-Item -Recurse -Force
  Copy-Item -Path (Join-Path $sourceDir "*") -Destination $tmp -Recurse -Force

  Exec "git" @("add", "-A") $tmp
  $status = & git -C $tmp status --porcelain
  if (-not $status) {
    Write-Host "[publish] no changes to publish."
    return
  }

  $msg = "chore: publish devcontainer features"
  Exec "git" @("commit", "-m", $msg) $tmp

  if (-not $SkipPush) {
    Exec "git" @("push", "origin", "HEAD:main") $tmp
    Write-Host "[publish] pushed to https://github.com/$repoRef"
  } else {
    Write-Host "[publish] commit created locally at $tmp (push skipped)"
  }
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
