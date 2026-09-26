#Requires -Version 5.1
<#
.SYNOPSIS
    Install the gplay (Google Play Console CLI) standalone binary on Windows.

.DESCRIPTION
    Downloads the latest gplay release binary for your architecture from GitHub,
    verifies its SHA-256 checksum, installs it to a user-writable directory, and
    adds that directory to your user PATH.

    One-liner:
      irm https://raw.githubusercontent.com/tamtom/play-console-cli/main/install.ps1 | iex

    Environment overrides:
      $env:GPLAY_VERSION      Install a specific tag (e.g. v0.6.0) instead of latest.
      $env:GPLAY_INSTALL_DIR  Install location (default: %LOCALAPPDATA%\gplay\bin).
#>

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Repo    = 'tamtom/play-console-cli'
$BinName = 'gplay'

# Prefer TLS 1.2 on older Windows PowerShell hosts.
try { [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 } catch {}

# --- Resolve architecture -------------------------------------------------
$rawArch = $env:PROCESSOR_ARCHITEW6432
if (-not $rawArch) { $rawArch = $env:PROCESSOR_ARCHITECTURE }
switch ($rawArch) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    'x86'   { throw 'Unsupported architecture: 32-bit x86. gplay ships 64-bit builds only.' }
    default { $arch = 'amd64' }
}

# --- Resolve install location --------------------------------------------
$installDir = $env:GPLAY_INSTALL_DIR
if (-not $installDir) { $installDir = Join-Path $env:LOCALAPPDATA 'gplay\bin' }

# --- Resolve download URLs ------------------------------------------------
if ($env:GPLAY_VERSION) {
    $baseUrl = "https://github.com/$Repo/releases/download/$($env:GPLAY_VERSION)"
} else {
    $baseUrl = "https://github.com/$Repo/releases/latest/download"
}
$checksumsUrl = "$baseUrl/checksums.txt"

$tmpDir = Join-Path ([IO.Path]::GetTempPath()) ("gplay-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
try {
    # Download the native-arch asset; fall back to amd64 if unpublished (x64
    # runs on Windows-on-ARM via emulation).
    $asset   = "$BinName-windows-$arch.exe"
    $binPath = Join-Path $tmpDir $asset
    Write-Host "Downloading $asset..."
    try {
        Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $binPath -UseBasicParsing
    } catch {
        if ($arch -ne 'amd64') {
            Write-Host "No native $arch build published; falling back to amd64 (runs via emulation)."
            $arch    = 'amd64'
            $asset   = "$BinName-windows-$arch.exe"
            $binPath = Join-Path $tmpDir $asset
            Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $binPath -UseBasicParsing
        } else {
            throw
        }
    }

    # --- Verify checksum --------------------------------------------------
    $checksumsPath = Join-Path $tmpDir 'checksums.txt'
    try {
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing
    } catch {
        throw "Cannot download checksums.txt; refusing to install an unverified binary."
    }
    $entries = @(Get-Content -Path $checksumsPath | Where-Object {
        $fields = $_.Trim() -split '\s+'
        $fields.Count -eq 2 -and $fields[1].TrimStart('*') -ceq $asset
    })
    if ($entries.Count -ne 1) {
        throw "Missing or duplicate SHA-256 checksum for $asset."
    }
    $expected = ($entries[0].Trim() -split '\s+')[0].ToLower()
    if ($expected -notmatch '^[0-9a-f]{64}$') {
        throw "Invalid SHA-256 checksum for $asset."
    }
    $actual = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) {
        throw "Checksum verification failed for $asset."
    }
    Write-Host "Checksum verified."

    # --- Install ----------------------------------------------------------
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    $dest = Join-Path $installDir "$BinName.exe"
    Copy-Item -Path $binPath -Destination $dest -Force
    Write-Host "Installed $BinName to $dest"

    # --- Ensure install dir is on the user PATH ---------------------------
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $onPath = $false
    if ($userPath) {
        $onPath = @($userPath -split ';' | Where-Object { $_.TrimEnd('\') -ieq $installDir.TrimEnd('\') }).Count -gt 0
    }
    if (-not $onPath) {
        $newPath = if ($userPath) { "$userPath;$installDir" } else { $installDir }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        $env:Path = "$env:Path;$installDir"
        Write-Host ""
        Write-Host "Added $installDir to your user PATH."
        Write-Host "Open a new terminal for it to take effect."
    }

    Write-Host ""
    Write-Host "Run: $BinName --help"
} finally {
    Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
