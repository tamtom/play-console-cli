param([string]$Installer, [string]$Work, [string]$Mode)
$ErrorActionPreference = 'Stop'
$global:GplayFixtureMode = $Mode
$global:GplayCandidate = [Text.Encoding]::ASCII.GetBytes('candidate executable')
$global:GplayAsset = 'gplay-windows-amd64.exe'
$installDir = Join-Path $Work 'installed'
New-Item -ItemType Directory -Path $installDir -Force | Out-Null
$destination = Join-Path $installDir 'gplay.exe'
[IO.File]::WriteAllText($destination, 'previous executable')
$env:GPLAY_INSTALL_DIR = $installDir
$env:GPLAY_VERSION = 'v1.0.0'
$env:PROCESSOR_ARCHITECTURE = 'AMD64'
$env:PROCESSOR_ARCHITEW6432 = ''

# Only the external download boundary is substituted; checksum parsing, hashing
# and destination writes run through the shipped installer.
function Invoke-WebRequest {
    param($Uri, $OutFile, [switch]$UseBasicParsing)
    if ($Uri.EndsWith('/checksums.txt')) {
        if ($global:GplayFixtureMode -eq 'unavailable') { throw 'fixture HTTP 404' }
        $sha = [Security.Cryptography.SHA256]::Create()
        try { $hash = ([BitConverter]::ToString($sha.ComputeHash($global:GplayCandidate))).Replace('-', '').ToLower() }
        finally { $sha.Dispose() }
        $name = $global:GplayAsset
        switch ($global:GplayFixtureMode) {
            'missing' { $name = 'another-asset.exe' }
            'substring' { $name = 'prefix-' + $name }
            'malformed' { $hash = 'invalid' }
            'mismatch' { $hash = '0' * 64 }
        }
        $manifest = "$hash  $name`n"
        if ($global:GplayFixtureMode -eq 'duplicate') { $manifest += $manifest }
        [IO.File]::WriteAllText($OutFile, $manifest)
    } else {
        [IO.File]::WriteAllBytes($OutFile, $global:GplayCandidate)
    }
}

# The install directory is temporary. Restore any Windows user-PATH bookkeeping
# performed by the shipped installer even when an assertion fails.
$originalPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$failed = $false
$failure = ""
try {
    try { & $Installer } catch { $failed = $true; $failure = $_.Exception.Message }
    $installed = [IO.File]::ReadAllText($destination)
    if ($Mode -eq 'valid') {
        if ($failed -or $installed -ne 'candidate executable') { throw "valid verified installer failed: $failure" }
    } elseif (-not $failed -or $installed -ne 'previous executable') {
        throw "unsafe $Mode checksum accepted or previous executable changed"
    }
} finally {
    if ([Environment]::OSVersion.Platform -eq [PlatformID]::Win32NT) {
        [Environment]::SetEnvironmentVariable('Path', $originalPath, 'User')
    }
}
