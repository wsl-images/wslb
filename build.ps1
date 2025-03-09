param (
    [string]$Version = "0.1.0"
)

$BinaryName = "wslb"
$BinDir = "bin"
$LdFlags = "-s -w -X github.com/wsl-images/wslb/internal/version.Version=$Version"


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

Write-Host "Creating Windows installer..."
$CleanVersion = $Version.Replace(' ', '')
dotnet build installer/wslb-installer/wslb-installer.wixproj -p:Version=$CleanVersion -c Release -o $BinDir

$MsiPath = "$BinDir\wslb-installer.msi"
if (Test-Path $MsiPath) {
    $Hash = (Get-FileHash -Path $MsiPath -Algorithm SHA256).Hash
    Write-Host "MSI Installer SHA256: $Hash"
    $Hash | Out-File "$BinDir\installer-hash.txt"
}

Write-Host "Build completed successfully!"