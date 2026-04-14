# Flashduty CLI installer for Windows
# Usage: irm https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.ps1 | iex
#
# Environment variables:
#   FLASHDUTY_VERSION     - specific version to install (e.g. "v0.1.2")
#   FLASHDUTY_INSTALL_DIR - install directory (default: $HOME\.flashduty\bin)

$ErrorActionPreference = "Stop"

$Repo = "flashcatcloud/flashduty-cli"
$Binary = "flashduty-cli.exe"
$InstalledName = "flashduty.exe"

function Write-Info($msg) {
    Write-Host "[flashduty] $msg"
}

function Fail($msg) {
    Write-Error "[flashduty] $msg"
    exit 1
}

# --- detect architecture ---

function Get-Arch {
    switch ($env:PROCESSOR_ARCHITECTURE) {
        "AMD64" { return "x86_64" }
        "ARM64" { return "arm64" }
        default  { Fail "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

# --- resolve version ---

function Get-Version {
    if ($env:FLASHDUTY_VERSION) {
        return $env:FLASHDUTY_VERSION
    }
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $release.tag_name
    } catch {
        Fail "could not determine latest version. Set FLASHDUTY_VERSION to install a specific version."
    }
}

# --- main ---

$Arch = Get-Arch
$Version = Get-Version

$InstallDir = if ($env:FLASHDUTY_INSTALL_DIR) {
    $env:FLASHDUTY_INSTALL_DIR
} else {
    Join-Path $HOME ".flashduty\bin"
}

$Archive = "flashduty-cli_Windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"

Write-Info "Installing Flashduty CLI $Version (Windows/$Arch)"
Write-Info "Downloading $Url"

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "flashduty-install-$([System.Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    $ArchivePath = Join-Path $TmpDir $Archive
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing

    Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

    $BinaryPath = Join-Path $TmpDir $Binary
    if (-not (Test-Path $BinaryPath)) {
        Fail "binary '$Binary' not found in archive"
    }

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $DestPath = Join-Path $InstallDir $InstalledName
    Move-Item -Path $BinaryPath -Destination $DestPath -Force

    Write-Info "Installed to $DestPath"

    # Add to user PATH if not already there
    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Info "Added $InstallDir to user PATH (restart your terminal for it to take effect)"
    }

    Write-Info "Run 'flashduty version' to verify"
} finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
