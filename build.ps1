param (
    [string]$Version = "x.x.x"
)

$BinaryName = "wslb"
$BinDir = "bin"
$LdFlags = "-X github.com/wsl-images/wslb/internal/version.Version=$Version"


if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
    Write-Host "Created bin directory"
}

Write-Host "Building WSLB Linux binary..."
$env:CGO_ENABLED = 0
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags $LdFlags -o "$BinDir\$BinaryName" .

Remove-Item env:CGO_ENABLED
Remove-Item env:GOOS
Remove-Item env:GOARCH

Write-Host "Building WSLB Windows binary..."
go build -ldflags $LdFlags -o "$BinDir\$BinaryName.exe" .

Write-Host "Build completed successfully!"