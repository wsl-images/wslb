param (
    [string]$Version = "0.1.0"
)

$BinaryName = "wslb"
$BinDir = "bin"
$LdFlags = "-X github.com/wsl-images/wslb/internal/version.Version=$Version"

# Ensure bin directory exists
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
    Write-Host "Created bin directory"
}

# Build Linux binary
Write-Host "Building WSLB Linux binary..."
$env:CGO_ENABLED = 0
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$linuxBinaryPath = Join-Path $BinDir $BinaryName
go build -ldflags $LdFlags -o $linuxBinaryPath .

# Clean up environment variables
Remove-Item env:CGO_ENABLED
Remove-Item env:GOOS
Remove-Item env:GOARCH

# Build Windows binary
Write-Host "Building WSLB Windows binary..."
$windowsBinaryPath = Join-Path $BinDir "$BinaryName.exe"
go build -ldflags $LdFlags -o $windowsBinaryPath .

# Sign the Windows binary with Cosign (keyless signing via GitHub OIDC)
Write-Host "Signing Windows binary with Cosign..."
# This will produce two files: wslb.exe.sig (signature) and wslb.exe.pem (certificate)
cosign sign-blob $windowsBinaryPath `
    --output-signature "$windowsBinaryPath.sig" `
    --output-certificate "$windowsBinaryPath.pem" `
    --yes

# Verify the signature to ensure it's valid
Write-Host "Verifying Cosign signature..."
cosign verify-blob $windowsBinaryPath `
    --signature "$windowsBinaryPath.sig" `
    --certificate "$windowsBinaryPath.pem" `
    --certificate-oidc-issuer "https://token.actions.githubusercontent.com"

# Calculate SHA256 hash of the Windows binary
Write-Host "Calculating wslb.exe SHA256..."
if (Test-Path $windowsBinaryPath) {
    $hash = (Get-FileHash -Path $windowsBinaryPath -Algorithm SHA256).Hash
    $hashFilePath = Join-Path $BinDir "installer-hash.txt"
    $hash | Out-File $hashFilePath
    Write-Host "Calculated SHA256: $hash"
} else {
    Write-Error "Error: $windowsBinaryPath not found!"
    exit 1
}

Write-Host "Build completed successfully!"
