# install.ps1 - azboard installer for Windows
# Usage:
#   iwr https://raw.githubusercontent.com/Popplywop/azboard/main/install.ps1 | iex
# Or with a custom install directory:
#   $env:INSTALL_DIR="C:\Tools"; iwr https://raw.githubusercontent.com/Popplywop/azboard/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo    = "Popplywop/azboard"
$Binary  = "azboard.exe"
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\azboard" }

# Detect architecture
$Arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else {
    Write-Error "Unsupported architecture. azboard requires a 64-bit system."
    exit 1
}

# Fetch latest release version
Write-Host "Fetching latest release..."
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
$Version = $Release.tag_name
$VersionNum = $Version.TrimStart("v")

$Archive  = "azboard_${VersionNum}_windows_${Arch}.zip"
$BaseUrl  = "https://github.com/$Repo/releases/download/$Version"
$Url      = "$BaseUrl/$Archive"

Write-Host "Installing azboard $Version (windows/$Arch)..."

# Download to temp directory
$Tmp = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path "$($_.FullName)" }
try {
    $ArchivePath = Join-Path $Tmp $Archive
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing

    # Verify with cosign if available
    $CosignCmd = Get-Command cosign -ErrorAction SilentlyContinue
    if ($CosignCmd) {
        Write-Host "Verifying signature with cosign..."
        $SigPath  = "$ArchivePath.sig"
        $PemPath  = "$ArchivePath.pem"
        Invoke-WebRequest -Uri "$Url.sig" -OutFile $SigPath -UseBasicParsing
        Invoke-WebRequest -Uri "$Url.pem" -OutFile $PemPath -UseBasicParsing
        & cosign verify-blob `
            --certificate         $PemPath `
            --signature           $SigPath `
            --certificate-identity-regexp "https://github.com/$Repo" `
            --certificate-oidc-issuer "https://token.actions.githubusercontent.com" `
            $ArchivePath
        Write-Host "Signature verified."
    } else {
        Write-Host "cosign not found - skipping signature verification."
        Write-Host "Install cosign to verify: https://docs.sigstore.dev/cosign/system_config/installation/"
    }

    # Extract
    Expand-Archive -Path $ArchivePath -DestinationPath $Tmp -Force

    # Create install dir and copy binary
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }
    Copy-Item -Path (Join-Path $Tmp $Binary) -Destination (Join-Path $InstallDir $Binary) -Force

} finally {
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}

# Add to PATH for current user if not already present
$UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to your PATH (restart your terminal to take effect)."
}

Write-Host ""
Write-Host "azboard $Version installed to $InstallDir\$Binary"
Write-Host "Run: azboard --version"
