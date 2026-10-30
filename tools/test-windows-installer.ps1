# Self-contained Windows checks; no network, Pester dependency, or user PATH writes.
param([switch]$Portable)

$ErrorActionPreference = 'Stop'
$installer = Join-Path $PSScriptRoot '..\install.ps1'
$tokens = $null
$parseErrors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile($installer, [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
# Load only function definitions, leaving the real entry point unexecuted.
foreach ($definition in $ast.EndBlock.Statements | Where-Object { $_ -is [Management.Automation.Language.FunctionDefinitionAst] }) {
    . ([scriptblock]::Create($definition.Extent.Text))
}
$nativeArchitecture = ${function:Get-JutsuArchitecture}
$root = Join-Path ([IO.Path]::GetTempPath()) ('jutsu-tests-' + [Guid]::NewGuid())
$originalPath = $env:Path
$originalArchitecture = $env:PROCESSOR_ARCHITECTURE
$originalNative = $env:PROCESSOR_ARCHITEW6432
$originalLocalAppData = $env:LOCALAPPDATA
$originalTemp = $env:TEMP
$originalTmp = $env:TMP
$originalTmpDir = $env:TMPDIR

# Portable mode exercises fixture logic on Linux; it cannot verify Windows file locking.
if ($Portable) { function Get-JutsuPlatform { return [PlatformID]::Win32NT } }

function Assert($Condition, [string]$Message) {
    if (-not $Condition) { throw "ASSERTION FAILED: $Message" }
}
function Get-JutsuUserPath { return $script:userPath }
function Set-JutsuUserPath([string]$Value) {
    if ($script:failPath) { throw 'fixture PATH denied' }
    $script:userPath = $Value
}
function Invoke-RestMethod { return @{ tag_name = 'v1.2.3' } }
function Invoke-WebRequest {
    param($Uri, $OutFile, [switch]$UseBasicParsing, $ErrorAction)
    if ($script:mode -eq 'download') { throw 'fixture download failed' }
    if ($Uri -like '*/checksums.txt') {
        $hash = (Get-FileHash -LiteralPath $script:fixtureZip -Algorithm SHA256).Hash
        if ($script:mode -eq 'mismatch') { $hash = '0' * 64 }
        $archiveName = $script:requestedArchive
        if ($script:mode -eq 'missing-checksum') { $archiveName += '.other' }
        $lines = @("$hash  $archiveName")
        if ($script:mode -eq 'duplicate') { $lines += $lines[0] }
        Set-Content -LiteralPath $OutFile -Value $lines -Encoding ASCII
    } else {
        $script:requestedArchive = ($Uri -split '/')[-1]
        $script:requestedUri = $Uri
        Copy-Item -LiteralPath $script:fixtureZip -Destination $OutFile
    }
}
function Reset-Fixture {
    $script:mode = ''
    $script:failPath = $false
    $script:userPath = 'C:\Existing;%USERPROFILE%\bin'
    $env:Path = $originalPath
    $env:PROCESSOR_ARCHITEW6432 = ''
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $script:destination = Join-Path $root ([Guid]::NewGuid().ToString() + ' tools')
}
function Assert-Clean {
    Assert (@(Get-ChildItem -LiteralPath $env:TEMP -Filter 'jutsu-install-*').Count -eq 0) 'download temporary files cleaned'
    if (Test-Path -LiteralPath $script:destination) {
        Assert (@(Get-ChildItem -LiteralPath $script:destination -Filter '.jutsu-*').Count -eq 0) 'staging files cleaned'
    }
}
function Expect-Failure([scriptblock]$Action, [string]$Pattern) {
    $message = ''
    try { & $Action } catch { $message = $_.Exception.Message }
    Assert (-not [string]::IsNullOrEmpty($message) -and $message -like $Pattern) "expected '$Pattern', got '$message'"
    Assert-Clean
}

try {
    $null = New-Item -ItemType Directory -Path $root
    $env:TEMP = Join-Path $root 'temp'
    $env:TMP = $env:TEMP
    $env:TMPDIR = $env:TEMP
    $null = New-Item -ItemType Directory -Path $env:TEMP
    $fixtureDir = Join-Path $root 'fixture'
    $null = New-Item -ItemType Directory -Path $fixtureDir
    Set-Content -LiteralPath (Join-Path $fixtureDir 'jutsu.exe') -Value 'new binary' -Encoding ASCII
    $script:fixtureZip = Join-Path $root 'fixture.zip'
    Compress-Archive -Path (Join-Path $fixtureDir 'jutsu.exe') -DestinationPath $script:fixtureZip

    Reset-Fixture
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    Assert ((& $nativeArchitecture) -eq 'amd64') 'amd64 detected'
    $env:PROCESSOR_ARCHITEW6432 = 'ARM64'
    Assert ((& $nativeArchitecture) -eq 'arm64') 'native ARM64 detected under emulation'
    $env:PROCESSOR_ARCHITEW6432 = ''
    $env:PROCESSOR_ARCHITECTURE = 'ARM64'
    Assert ((& $nativeArchitecture) -eq 'arm64') 'native ARM64 shell detected'
    $env:PROCESSOR_ARCHITECTURE = 'x86'
    Expect-Failure { Install-Jutsu -InstallDir $script:destination } '*Unsupported Windows architecture*'
    Assert (-not (Test-Path -LiteralPath $script:destination)) 'unsupported target did not install'
    function Get-JutsuPlatform { return [PlatformID]::Unix }
    Expect-Failure { Install-Jutsu -InstallDir $script:destination } '*Windows only*'
    if ($Portable) {
        function Get-JutsuPlatform { return [PlatformID]::Win32NT }
    } else {
        function Get-JutsuPlatform { return [Environment]::OSVersion.Platform }
    }

    Reset-Fixture
    Install-Jutsu -InstallDir $script:destination
    Assert ($script:requestedArchive -eq 'jutsu_1.2.3_windows_amd64.zip') 'latest resolved correctly'
    $target = Join-Path $script:destination 'jutsu.exe'
    Assert ((Get-Content -LiteralPath $target -Raw).Trim() -eq 'new binary') 'initial binary installed'
    Assert ($script:userPath.StartsWith('C:\Existing;%USERPROFILE%\bin;')) 'user PATH entries preserved'
    $savedUserPath = $script:userPath
    $savedProcessPath = $env:Path
    Set-Content -LiteralPath $target -Value 'old binary'
    Install-Jutsu -Version v2.0.0 -InstallDir $script:destination
    Assert ($script:requestedUri -like '*/v2.0.0/jutsu_2.0.0_windows_amd64.zip') 'pinned tag used'
    Assert ((Get-Content -LiteralPath $target -Raw).Trim() -eq 'new binary') 'upgrade replaced old binary'
    Assert ($script:userPath -ceq $savedUserPath) 'user PATH not duplicated'
    Assert ($env:Path -ceq $savedProcessPath) 'process PATH not duplicated'
    Assert-Clean
    Assert ((Add-JutsuPathEntry 'C:\OTHER;c:\tools\' 'C:\Tools') -ceq 'C:\OTHER;c:\tools\') 'case and slash normalization'

    Reset-Fixture
    $env:LOCALAPPDATA = Join-Path $root 'local'
    Install-Jutsu -Version v1.2.3
    Assert (Test-Path -LiteralPath (Join-Path $env:LOCALAPPDATA 'Programs\jutsu\jutsu.exe')) 'default destination'
    Assert-Clean

    foreach ($failureMode in @('download', 'mismatch', 'missing-checksum', 'duplicate', 'missing-exe')) {
        Reset-Fixture
        $script:mode = $failureMode
        $null = New-Item -ItemType Directory -Path $script:destination
        $target = Join-Path $script:destination 'jutsu.exe'
        Set-Content -LiteralPath $target -Value 'keep binary'
        $beforePath = $env:Path
        if ($failureMode -eq 'missing-exe') {
            $script:fixtureZip = Join-Path $root 'missing.zip'
            Set-Content -LiteralPath (Join-Path $root 'other.txt') -Value 'other'
            Compress-Archive -Path (Join-Path $root 'other.txt') -DestinationPath $script:fixtureZip
        }
        Expect-Failure { Install-Jutsu -Version v1.2.3 -InstallDir $script:destination } '*'
        Assert ((Get-Content -LiteralPath $target -Raw).Trim() -eq 'keep binary') "$failureMode preserved binary"
        Assert ($script:userPath -ceq 'C:\Existing;%USERPROFILE%\bin') "$failureMode preserved user PATH"
        Assert ($env:Path -ceq $beforePath) "$failureMode preserved process PATH"
    }
    $script:fixtureZip = Join-Path $root 'fixture.zip'

    if (-not $Portable) {
        Reset-Fixture
        $null = New-Item -ItemType Directory -Path $script:destination
        $target = Join-Path $script:destination 'jutsu.exe'
        Set-Content -LiteralPath $target -Value 'locked binary'
        $lock = [IO.File]::Open($target, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::None)
        try { Expect-Failure { Install-Jutsu -Version v1.2.3 -InstallDir $script:destination } '*Close Jutsu and retry*' }
        finally { $lock.Dispose() }
        Assert ((Get-Content -LiteralPath $target -Raw).Trim() -eq 'locked binary') 'locked executable preserved'
    }

    Reset-Fixture
    $script:failPath = $true
    Expect-Failure { Install-Jutsu -Version v1.2.3 -InstallDir $script:destination } '*installed*PATH setup failed*'
    Assert (Test-Path -LiteralPath (Join-Path $script:destination 'jutsu.exe')) 'PATH failure retained installation'

    Reset-Fixture
    # Deterministically hide jj without changing command discovery for built-in cmdlets.
    function Get-Command { param($Name, $ErrorAction) return $null }
    $warnings = @()
    Install-Jutsu -Version v1.2.3 -InstallDir $script:destination -WarningVariable warnings
    Assert (($warnings | Out-String) -like '*jj was not found*') 'missing jj is a warning'
    Assert-Clean
    Remove-Item Function:\Get-Command

    Reset-Fixture
    Expect-Failure { Install-Jutsu -Version '../invalid' -InstallDir $script:destination } '*Invalid or missing release tag*'
    Expect-Failure { Install-Jutsu -Version v1.2.3 -InstallDir 'C:\bad;path' } '*semicolon*'
    Write-Host 'Windows installer fixture checks passed.'
} finally {
    $env:Path = $originalPath
    $env:PROCESSOR_ARCHITECTURE = $originalArchitecture
    $env:PROCESSOR_ARCHITEW6432 = $originalNative
    $env:LOCALAPPDATA = $originalLocalAppData
    $env:TEMP = $originalTemp
    $env:TMP = $originalTmp
    $env:TMPDIR = $originalTmpDir
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}
