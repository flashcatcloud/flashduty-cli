# Flashduty CLI installer for Windows
# Usage: irm https://raw.githubusercontent.com/flashcatcloud/flashduty-cli/main/install.ps1 | iex
#
# Environment variables:
#   FLASHDUTY_VERSION     - specific version to install (e.g. "v0.1.2")
#   FLASHDUTY_INSTALL_DIR - install directory (default: $HOME\.flashduty\bin)
#   MIRROR_URL            - fetch release assets from this https mirror prefix
#                           instead of github.com. The mirror must replicate
#                           GitHub's release layout
#                           (<MIRROR_URL>/releases/download/<tag>/<asset>) and
#                           expose a plain-text <MIRROR_URL>/releases/latest file
#                           containing the latest tag.

$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$Repo = "flashcatcloud/flashduty-cli"
$Binary = "flashduty-cli.exe"
$InstalledName = "flashduty.exe"

# When set, all release downloads are fetched from this prefix instead of github.com.
$MirrorUrl = $env:MIRROR_URL
if ($MirrorUrl) {
    $MirrorUrl = $MirrorUrl.TrimEnd('/')
    if ($MirrorUrl -notlike "https://*") {
        Write-Error "[flashduty] MIRROR_URL must use https:// scheme, got: $MirrorUrl"
        exit 1
    }
}

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
    if ($MirrorUrl) {
        try {
            $raw = Invoke-RestMethod -Uri "$MirrorUrl/releases/latest" -UseBasicParsing
            $version = ([string]$raw).Trim()
        } catch {
            Fail "could not fetch $MirrorUrl/releases/latest. Set FLASHDUTY_VERSION to install a specific version."
        }
    } else {
        try {
            $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
            $version = $release.tag_name
        } catch {
            Fail "could not determine latest version. Set FLASHDUTY_VERSION to install a specific version."
        }
    }
    # The resolved value comes from a network response and is interpolated into
    # the download URL — reject anything that isn't a plain release tag.
    if ($version -notmatch '^v[0-9][A-Za-z0-9.+-]*$') {
        Fail "resolved version is not a valid release tag: '$version'"
    }
    return $version
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
$Base = if ($MirrorUrl) {
    "$MirrorUrl/releases/download/$Version"
} else {
    "https://github.com/$Repo/releases/download/$Version"
}
$Url = "$Base/$Archive"

Write-Info "Installing Flashduty CLI $Version (Windows/$Arch)"
Write-Info "Downloading $Url"

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "flashduty-install-$([System.Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    $ArchivePath = Join-Path $TmpDir $Archive
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath -UseBasicParsing

    # Verify against the published checksums.txt when present. Releases cut
    # before the mirror existed don't ship one, so a missing file only warns.
    $ChecksumPath = Join-Path $TmpDir "checksums.txt"
    try {
        Invoke-WebRequest -Uri "$Base/checksums.txt" -OutFile $ChecksumPath -UseBasicParsing
    } catch {
        $ChecksumPath = $null
    }
    if ($ChecksumPath -and (Test-Path $ChecksumPath)) {
        $expected = $null
        foreach ($line in Get-Content $ChecksumPath) {
            $parts = $line -split '\s+', 2
            if ($parts.Count -eq 2 -and $parts[1].Trim() -eq $Archive) {
                $expected = $parts[0].Trim().ToLower()
                break
            }
        }
        if (-not $expected) {
            Fail "archive $Archive not listed in checksums.txt (wrong release or renamed asset)"
        }
        $actual = (Get-FileHash -Path $ArchivePath -Algorithm SHA256).Hash.ToLower()
        if ($actual -ne $expected) {
            Fail "checksum mismatch for ${Archive}: expected $expected, got $actual"
        }
        Write-Info "Checksum OK"
    } else {
        Write-Info "WARNING: checksums.txt not available -- skipping integrity check"
    }

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
