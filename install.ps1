# Usage: irm https://raw.githubusercontent.com/AliQ80/jutsu/main/install.ps1 | iex
# Or: .\install.ps1 -Version v1.2.3 -InstallDir 'C:\Tools\jutsu'
# Requires Windows PowerShell 5.1 or PowerShell 7 on Windows.
[CmdletBinding()]
param([string]$Version, [string]$InstallDir)

function Get-JutsuPlatform { return [Environment]::OSVersion.Platform }

function Get-JutsuArchitecture {
    if ((Get-JutsuPlatform) -ne [PlatformID]::Win32NT) {
        throw 'This installer supports Windows only. Use install.sh on Linux/macOS.'
    }
    # PROCESSOR_ARCHITEW6432 reports the native architecture under emulation.
    $native = $env:PROCESSOR_ARCHITEW6432
    if (-not $native) { $native = $env:PROCESSOR_ARCHITECTURE }
    switch ($native) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { throw "Unsupported Windows architecture: $native" }
    }
}

function Add-JutsuPathEntry([string]$Existing, [string]$Directory) {
    $normalized = $Directory.TrimEnd('\', '/')
    foreach ($entry in ($Existing -split ';')) {
        $expanded = [Environment]::ExpandEnvironmentVariables($entry.Trim().Trim('"')).TrimEnd('\', '/')
        if ($expanded -ieq $normalized) { return $Existing }
    }
    if ([string]::IsNullOrEmpty($Existing)) { return $Directory }
    return $Existing.TrimEnd(';') + ';' + $Directory
}

function Get-JutsuUserPath {
    return [Environment]::GetEnvironmentVariable('Path', 'User')
}

function Set-JutsuUserPath([string]$Value) {
    [Environment]::SetEnvironmentVariable('Path', $Value, 'User')
}

function Install-Jutsu {
    [CmdletBinding()]
    param([string]$Version, [string]$InstallDir)
    $ErrorActionPreference = 'Stop'
    $arch = Get-JutsuArchitecture
    if (-not $InstallDir) {
        if (-not $env:LOCALAPPDATA) { throw 'LOCALAPPDATA is unavailable. Specify -InstallDir.' }
        $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\jutsu'
    }
    if ($InstallDir.Contains(';')) { throw 'InstallDir cannot contain a semicolon (PATH separator).' }
    $InstallDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($InstallDir)
    $repo = 'https://github.com/AliQ80/jutsu'
    if (-not $Version) {
        $Version = (Invoke-RestMethod 'https://api.github.com/repos/AliQ80/jutsu/releases/latest' -ErrorAction Stop).tag_name
    }
    if ($Version -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
        throw "Invalid or missing release tag: '$Version'. Use an exact tag such as v1.2.3."
    }
    $archive = 'jutsu_{0}_windows_{1}.zip' -f ($Version -replace '^v', ''), $arch
    $base = "$repo/releases/download/$Version"
    $work = Join-Path ([IO.Path]::GetTempPath()) ('jutsu-install-' + [Guid]::NewGuid())
    $staged = $null
    try {
        $null = New-Item -ItemType Directory -Path $work
        Write-Host "Downloading $archive ($Version)..."
        $zip = Join-Path $work $archive
        $manifest = Join-Path $work 'checksums.txt'
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$archive" -OutFile $zip -ErrorAction Stop
        Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile $manifest -ErrorAction Stop
        $entries = @(foreach ($line in (Get-Content -LiteralPath $manifest)) {
            if ($line -match '^([a-fA-F0-9]{64})\s+\*?(.+)$' -and $Matches[2] -ceq $archive) { $Matches[1] }
        })
        if ($entries.Count -ne 1) { throw "Expected exactly one checksum entry for $archive." }
        if ((Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash -ine $entries[0]) {
            throw "Checksum mismatch for $archive. Installation cancelled."
        }
        $extracted = Join-Path $work 'extracted'
        Expand-Archive -LiteralPath $zip -DestinationPath $extracted -ErrorAction Stop
        $binary = Join-Path $extracted 'jutsu.exe'
        if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) { throw 'Release archive is missing jutsu.exe.' }
        $null = New-Item -ItemType Directory -Path $InstallDir -Force
        $destination = Join-Path $InstallDir 'jutsu.exe'
        $staged = Join-Path $InstallDir ('.jutsu-' + [Guid]::NewGuid() + '.exe')
        Copy-Item -LiteralPath $binary -Destination $staged
        try {
            if (Test-Path -LiteralPath $destination) {
                [IO.File]::Replace($staged, $destination, [System.Management.Automation.Language.NullString]::Value)
            } else {
                [IO.File]::Move($staged, $destination)
            }
        } catch {
            throw "Cannot replace '$destination'. Close Jutsu and retry; check directory permissions. Existing executable was preserved. $($_.Exception.Message)"
        }
        Write-Host "Installed Jutsu $Version to $destination"
        try {
            $userPath = Get-JutsuUserPath
            $updated = Add-JutsuPathEntry $userPath $InstallDir
            if ($updated -cne $userPath) { Set-JutsuUserPath $updated }
            $env:Path = Add-JutsuPathEntry $env:Path $InstallDir
        } catch {
            throw "Jutsu is installed at '$destination', but PATH setup failed. Add '$InstallDir' to your user PATH manually and reopen your terminal. $($_.Exception.Message)"
        }
        if (-not (Get-Command jj -ErrorAction SilentlyContinue)) {
            Write-Warning 'jj was not found on PATH. Install it: https://jj-vcs.github.io/jj/latest/install-and-setup/'
        }
        Write-Host 'Ready: run jutsu. If another installation is found first, check Get-Command jutsu -All.'
    } finally {
        if ($staged -and (Test-Path -LiteralPath $staged)) { Remove-Item -LiteralPath $staged -Force }
        if (Test-Path -LiteralPath $work) { Remove-Item -LiteralPath $work -Recurse -Force }
    }
}

Install-Jutsu -Version $Version -InstallDir $InstallDir
