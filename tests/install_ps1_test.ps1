# Behavior tests for install.ps1. Hermetic: release archives are built locally
# from a stub program and served by an Invoke-WebRequest stand-in, so no
# network is touched. Needs Windows and Go on PATH.
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$InstallScript = Join-Path $Root "install.ps1"
$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "install-ps1-test-$([System.Guid]::NewGuid().ToString('N'))"
$Fixtures = Join-Path $TmpDir "fixtures"
New-Item -ItemType Directory -Path $Fixtures -Force | Out-Null

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "x86_64" }
    "ARM64" { "arm64" }
    default { throw "unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}
$Archive = "flashduty-cli_Windows_${Arch}.zip"

# Stub CLI: prints its build version; `sleep` keeps it running so the
# installed .exe is locked while a reinstall happens.
$StubSource = Join-Path $TmpDir "stub.go"
Set-Content -Path $StubSource -Encoding ascii -Value @'
package main

import (
	"fmt"
	"os"
	"time"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "sleep" {
		time.Sleep(10 * time.Minute)
	}
	fmt.Println("stub " + version)
}
'@

# Each release is <fixtures>/<version>/{archive, checksums.txt}, the layout
# install.ps1 downloads from <MIRROR_URL>/releases/download/<version>/.
function New-Release($Version) {
    $dir = Join-Path $Fixtures $Version
    $build = Join-Path $dir "build"
    New-Item -ItemType Directory -Path $build -Force | Out-Null
    & go build -ldflags "-X main.version=$Version" -o (Join-Path $build "flashduty-cli.exe") $StubSource
    if ($LASTEXITCODE -ne 0) { throw "go build failed for $Version" }
    $zip = Join-Path $dir $Archive
    Compress-Archive -Path (Join-Path $build "flashduty-cli.exe") -DestinationPath $zip -Force
    $sum = (Get-FileHash -Path $zip -Algorithm SHA256).Hash.ToLower()
    Set-Content -Path (Join-Path $dir "checksums.txt") -Encoding ascii -Value "$sum  $Archive"
}

# Stand-in for the cmdlet: functions take precedence over cmdlets, so the
# installer run below in this session downloads from the local fixtures.
function Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    if ($Uri -notmatch '/releases/download/([^/]+)/([^/]+)$') { throw "unexpected URL: $Uri" }
    $src = Join-Path (Join-Path $Fixtures $Matches[1]) $Matches[2]
    if (-not (Test-Path $src)) { throw "no fixture for URL: $Uri" }
    Copy-Item -Path $src -Destination $OutFile -Force
}

function Invoke-Installer($InstallDir, $Version, $InstalledName) {
    $env:MIRROR_URL = "https://mirror.example/flashduty-cli"
    $env:FLASHDUTY_VERSION = $Version
    $env:FLASHDUTY_INSTALL_DIR = $InstallDir
    $env:INSTALLED_NAME = $InstalledName
    & $InstallScript | Out-Null
}

function Assert-Runs($Exe, $Version) {
    $out = (& $Exe | Out-String).Trim()
    if ($out -ne "stub $Version") { throw "$Exe printed '$out', want 'stub $Version'" }
}

function Assert-ExactName($Dir, $Name) {
    $names = @(Get-ChildItem -Path $Dir -File | Where-Object { $_.Name -notlike "*.old" } | ForEach-Object { $_.Name })
    if ($names.Count -ne 1 -or $names[0] -cne $Name) {
        throw "files in ${Dir}: [$($names -join ', ')], want exactly [$Name]"
    }
}

function Get-OldCopies($Dir, $Name) {
    @(Get-ChildItem -Path $Dir -File -Filter "$Name*.old")
}

function Test-Case($Name, [scriptblock]$Body) {
    try {
        & $Body
        Write-Host "PASS: $Name"
    } catch {
        Write-Host "FAIL: $Name -- $_"
        $script:Failed++
    }
}

$Failed = 0
$Sleepers = @()
$SavedUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
try {
    New-Release "v1.0.0"
    New-Release "v2.0.0"
    New-Release "v3.0.0"

    Test-Case "default name installs flashduty.exe" {
        $dir = Join-Path $TmpDir "default"
        Invoke-Installer $dir "v1.0.0" ""
        Assert-ExactName $dir "flashduty.exe"
        Assert-Runs (Join-Path $dir "flashduty.exe") "v1.0.0"
    }

    Test-Case "INSTALLED_NAME=fduty installs fduty.exe" {
        $dir = Join-Path $TmpDir "fduty"
        Invoke-Installer $dir "v1.0.0" "fduty"
        Assert-ExactName $dir "fduty.exe"
        Assert-Runs (Join-Path $dir "fduty.exe") "v1.0.0"
    }

    Test-Case "INSTALLED_NAME=FDUTY.EXE installs FDUTY.exe" {
        $dir = Join-Path $TmpDir "fduty-upper"
        Invoke-Installer $dir "v1.0.0" "FDUTY.EXE"
        Assert-ExactName $dir "FDUTY.exe"
        Assert-Runs (Join-Path $dir "FDUTY.exe") "v1.0.0"
    }

    $runDir = Join-Path $TmpDir "running"
    $runExe = Join-Path $runDir "flashduty.exe"

    Test-Case "reinstall replaces the installed exe while it is running" {
        Invoke-Installer $runDir "v1.0.0" ""
        $script:Sleepers += Start-Process -FilePath $runExe -ArgumentList "sleep" -PassThru -WindowStyle Hidden
        Start-Sleep -Seconds 1
        Invoke-Installer $runDir "v2.0.0" ""
        Assert-Runs $runExe "v2.0.0"
        if ($script:Sleepers[-1].HasExited) { throw "running v1.0.0 process exited during reinstall" }
        if ((Get-OldCopies $runDir "flashduty.exe").Count -ne 1) { throw "want one moved-aside copy of the running exe" }
    }

    Test-Case "reinstall while the moved-aside copy is still running" {
        # The v1.0.0 process from the previous case still runs from the
        # moved-aside file, which therefore cannot be deleted or replaced.
        $script:Sleepers += Start-Process -FilePath $runExe -ArgumentList "sleep" -PassThru -WindowStyle Hidden
        Start-Sleep -Seconds 1
        Invoke-Installer $runDir "v3.0.0" ""
        Assert-Runs $runExe "v3.0.0"
        Assert-ExactName $runDir "flashduty.exe"
    }

    Test-Case "a stale moved-aside copy from an earlier install is cleaned up" {
        $dir = Join-Path $TmpDir "stale"
        Invoke-Installer $dir "v1.0.0" ""
        Invoke-Installer $dir "v2.0.0" ""
        Invoke-Installer $dir "v3.0.0" ""
        Assert-Runs (Join-Path $dir "flashduty.exe") "v3.0.0"
        $old = Get-OldCopies $dir "flashduty.exe"
        if ($old.Count -ne 1) { throw "want one moved-aside copy after three installs, got $($old.Count): $($old.Name -join ', ')" }
    }
} finally {
    $Sleepers | Where-Object { -not $_.HasExited } | Stop-Process -Force -ErrorAction SilentlyContinue
    [Environment]::SetEnvironmentVariable("Path", $SavedUserPath, "User")
    Start-Sleep -Seconds 1
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

if ($Failed -gt 0) {
    Write-Host "$Failed case(s) failed"
    exit 1
}
Write-Host "all install.ps1 cases passed"
