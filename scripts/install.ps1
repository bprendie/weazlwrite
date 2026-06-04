param(
    [switch]$SkipLaunch
)

$ErrorActionPreference = "Stop"

$AppName = "weazlwrite"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$InstallRoot = if ($env:WEAZLWRITE_HOME) { $env:WEAZLWRITE_HOME } else { Join-Path $env:APPDATA $AppName }
$BinDir = Join-Path $InstallRoot "bin"
$VaultDir = Join-Path $InstallRoot "vault"
$ConfigPath = Join-Path $InstallRoot "config.json"
$GoCache = if ($env:GOCACHE) { $env:GOCACHE } else { Join-Path $RepoRoot ".gocache" }
$GoModCache = if ($env:GOMODCACHE) { $env:GOMODCACHE } else { Join-Path $RepoRoot ".gomodcache" }
$MsysRoot = if ($env:WEAZLWRITE_MSYS_ROOT) { $env:WEAZLWRITE_MSYS_ROOT } else { "C:\msys64" }
$MsysGccBin = Join-Path $MsysRoot "ucrt64\bin"
$MsysPacman = Join-Path $MsysRoot "usr\bin\pacman.exe"
$RequiredGo = "1.25.10"

function Add-ToCurrentPath($Path) {
    if (-not ($env:Path.Split(';') -contains $Path)) {
        $env:Path = "$Path;$env:Path"
    }
}

function Add-ToUserPath($Path) {
    $current = [Environment]::GetEnvironmentVariable("Path", "User")
    if (-not $current) {
        $current = ""
    }
    $items = $current.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries)
    if ($items -contains $Path) {
        return
    }
    $next = if ($current.Trim()) { "$current;$Path" } else { $Path }
    [Environment]::SetEnvironmentVariable("Path", $next, "User")
    Write-Host "Added $Path to the user PATH."
}

function Get-GoVersion {
    $raw = (& go version 2>$null)
    if ($LASTEXITCODE -ne 0 -or -not $raw) {
        return $null
    }
    if ($raw -match 'go([0-9]+\.[0-9]+(?:\.[0-9]+)?)') {
        return [version]$Matches[1]
    }
    return $null
}

function Ensure-Winget {
    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "winget was not found. Install Go $RequiredGo+ and MSYS2 manually, then rerun this script."
    }
}

function Ensure-Go {
    $go = Get-Command go -ErrorAction SilentlyContinue
    $version = if ($go) { Get-GoVersion } else { $null }
    if ($version -and $version -ge [version]$RequiredGo) {
        return
    }

    Ensure-Winget
    Write-Host "Installing Go $RequiredGo+ with winget..."
    winget install --id GoLang.Go --source winget --accept-package-agreements --accept-source-agreements
    Add-ToCurrentPath "C:\Program Files\Go\bin"
    Add-ToCurrentPath (Join-Path $env:USERPROFILE "go\bin")

    $version = Get-GoVersion
    if (-not $version -or $version -lt [version]$RequiredGo) {
        throw "Go $RequiredGo or newer is required. Restart PowerShell if Go was just installed, then rerun this script."
    }
}

function Ensure-Gcc {
    if (Get-Command gcc -ErrorAction SilentlyContinue) {
        return
    }
    Add-ToCurrentPath $MsysGccBin
    if (Get-Command gcc -ErrorAction SilentlyContinue) {
        return
    }

    Ensure-Winget
    if (-not (Test-Path $MsysPacman)) {
        Write-Host "Installing MSYS2 with winget for the SQLite CGO toolchain..."
        winget install --id MSYS2.MSYS2 --source winget --accept-package-agreements --accept-source-agreements
    }
    if (-not (Test-Path $MsysPacman)) {
        throw "MSYS2 pacman was not found at $MsysPacman. Install MSYS2, then rerun this script."
    }

    Write-Host "Installing GCC with MSYS2 pacman..."
    & $MsysPacman -Sy --needed --noconfirm mingw-w64-ucrt-x86_64-gcc
    Add-ToCurrentPath $MsysGccBin
    Add-ToUserPath $MsysGccBin

    if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
        throw "gcc was not found after MSYS2 install. Open a new PowerShell window and rerun this script."
    }
}

if (-not $env:APPDATA) {
    throw "APPDATA is not set. Windows installs need APPDATA for WeazlWrite's config, vaults, and bin directory."
}

Ensure-Go
Ensure-Gcc

New-Item -ItemType Directory -Force -Path $BinDir, $VaultDir, $GoCache, $GoModCache | Out-Null
Add-ToCurrentPath $BinDir
Add-ToUserPath $BinDir

$env:CGO_ENABLED = "1"
$env:GOCACHE = $GoCache
$env:GOMODCACHE = $GoModCache

Write-Host "Building $AppName..."
Push-Location $RepoRoot
try {
    go build -buildvcs=false -o (Join-Path $BinDir "$AppName.exe") ./cmd/weazlwrite
    go build -buildvcs=false -o (Join-Path $BinDir "$AppName-setup.exe") ./cmd/weazlwrite-setup
}
finally {
    Pop-Location
}

Write-Host "Installed $AppName to $BinDir"
Write-Host "Config and vaults live under $InstallRoot"

Write-Host ""
Write-Host "Configuring model provider..."
& (Join-Path $BinDir "$AppName-setup.exe")

if ($SkipLaunch -or $env:WEAZLWRITE_SKIP_LAUNCH -eq "1") {
    Write-Host "Skipping first launch."
}
else {
    Write-Host ""
    Write-Host "Launching $AppName..."
    & (Join-Path $BinDir "$AppName.exe")
}
