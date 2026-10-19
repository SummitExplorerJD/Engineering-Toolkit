[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$CandidateDirectory,

    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [ValidateSet('windows', 'linux', 'darwin')]
    [string]$OS,

    [Parameter(Mandatory = $true)]
    [ValidateSet('amd64', 'arm64')]
    [string]$Arch
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$candidate = (Resolve-Path -LiteralPath $CandidateDirectory).Path
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$testRoot = Join-Path $repositoryRoot ('.engkit-test-temp\installer-' + [guid]::NewGuid().ToString('N'))
$installDirectory = Join-Path $testRoot 'bin'
$tamperedAssets = Join-Path $testRoot 'tampered'
$wrongVersionAssets = Join-Path $testRoot 'wrong-version'
$originalHome = $env:HOME
$originalFailure = $env:ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP
$originalPathFile = $env:ENGKIT_INSTALL_TEST_USER_PATH_FILE
$originalProfile = $env:ENGKIT_INSTALL_TEST_PROFILE
$nativePreferenceVariable = Get-Variable -Name PSNativeCommandUseErrorActionPreference -ErrorAction SilentlyContinue
$originalNativePreference = if ($null -ne $nativePreferenceVariable) { $nativePreferenceVariable.Value } else { $null }
if ($null -ne $nativePreferenceVariable) {
    $PSNativeCommandUseErrorActionPreference = $false
}

function Invoke-Installer {
    param(
        [string]$Assets,
        [switch]$Upgrade,
        [switch]$AddToPath,
        [switch]$ExpectFailure
    )

    if ($OS -eq 'windows') {
        $parameters = @{
            Version        = $Version
            InstallDir     = $installDirectory
            AssetDirectory = $Assets
            Upgrade        = [bool]$Upgrade
            AddToPath      = [bool]$AddToPath
        }
        try {
            & (Join-Path $candidate 'install.ps1') @parameters
            if ($ExpectFailure) { throw 'Installer unexpectedly succeeded' }
        } catch {
            if (-not $ExpectFailure) { throw }
        }
        return
    }

    $arguments = @((Join-Path $candidate 'install.sh'), '--version', $Version, '--install-dir', $installDirectory, '--asset-directory', $Assets)
    if ($Upgrade) { $arguments += '--upgrade' }
    if ($AddToPath) { $arguments += '--add-to-path' }
    & sh @arguments
    $failed = $LASTEXITCODE -ne 0
    if ($ExpectFailure -and -not $failed) { throw 'Installer unexpectedly succeeded' }
    if (-not $ExpectFailure -and $failed) { throw "Installer failed with exit code $LASTEXITCODE" }
}

New-Item -ItemType Directory -Path $testRoot, $tamperedAssets, $wrongVersionAssets -Force | Out-Null
try {
    Copy-Item -LiteralPath (Join-Path $candidate 'SHA256SUMS') -Destination $tamperedAssets
    $extension = if ($OS -eq 'windows') { '.exe' } else { '' }
    $expectedBinaryName = "engkit-$Version-$OS-$Arch$extension"
    $platformBinary = @(Get-ChildItem -LiteralPath $candidate -File | Where-Object {
        $_.Name -ceq $expectedBinaryName
    })
    if (@($platformBinary).Count -ne 1) {
        throw "Expected exactly one $OS binary in $candidate"
    }
    Copy-Item -LiteralPath $platformBinary[0].FullName -Destination $tamperedAssets
    Add-Content -LiteralPath (Join-Path $tamperedAssets $platformBinary[0].Name) -Value 'tampered'
    Invoke-Installer -Assets $tamperedAssets -ExpectFailure
    if (Test-Path -LiteralPath (Join-Path $installDirectory ($(if ($OS -eq 'windows') { 'engkit.exe' } else { 'engkit' })))) {
        throw 'Checksum failure left an installed executable'
    }

    $wrongVersion = 'v9.9.9'
    $wrongVersionBinary = Join-Path $wrongVersionAssets $expectedBinaryName
    $module = 'github.com/SummitExplorerJD/Engineering-Toolkit'
    $linkerFlags = "-s -w -buildid= -X $module/internal/cli.Version=$wrongVersion"
    Push-Location $repositoryRoot
    try {
        & go build -trimpath -buildvcs=false -ldflags $linkerFlags -o $wrongVersionBinary ./cmd/engkit
        if ($LASTEXITCODE -ne 0) { throw 'Could not build wrong-version installer fixture' }
    } finally {
        Pop-Location
    }
    $wrongVersionHash = (Get-FileHash -LiteralPath $wrongVersionBinary -Algorithm SHA256).Hash.ToLowerInvariant()
    "$wrongVersionHash  $expectedBinaryName" | Set-Content -LiteralPath (Join-Path $wrongVersionAssets 'SHA256SUMS') -Encoding ascii
    Invoke-Installer -Assets $wrongVersionAssets -ExpectFailure
    if (Test-Path -LiteralPath (Join-Path $installDirectory ($(if ($OS -eq 'windows') { 'engkit.exe' } else { 'engkit' })))) {
        throw 'Version failure left an installed executable'
    }

    Invoke-Installer -Assets $candidate
    $installed = Join-Path $installDirectory ($(if ($OS -eq 'windows') { 'engkit.exe' } else { 'engkit' }))
    $installedHash = (Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash

    Invoke-Installer -Assets $candidate -ExpectFailure
    if ((Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash -ne $installedHash) {
        throw 'Rejected replacement changed the installed executable'
    }

    $env:ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP = '1'
    Invoke-Installer -Assets $candidate -Upgrade -ExpectFailure
    $env:ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP = $null
    if ((Get-FileHash -LiteralPath $installed -Algorithm SHA256).Hash -ne $installedHash) {
        throw 'Failed upgrade did not restore the installed executable'
    }

    $env:HOME = Join-Path $testRoot 'home'
    New-Item -ItemType Directory -Path $env:HOME -Force | Out-Null
    if ($OS -eq 'windows') {
        $env:ENGKIT_INSTALL_TEST_USER_PATH_FILE = Join-Path $testRoot 'user-path.txt'
    } else {
        $env:ENGKIT_INSTALL_TEST_PROFILE = Join-Path $env:HOME '.profile'
    }
    Invoke-Installer -Assets $candidate -Upgrade -AddToPath
    Invoke-Installer -Assets $candidate -Upgrade -AddToPath

    if ($OS -eq 'windows') {
        $pathText = Get-Content -Raw -LiteralPath $env:ENGKIT_INSTALL_TEST_USER_PATH_FILE
        if (@($pathText -split ';' | Where-Object { $_ -ieq $installDirectory }).Count -ne 1) {
            throw 'Windows PATH update was not idempotent'
        }
    } else {
        $profileText = Get-Content -Raw -LiteralPath $env:ENGKIT_INSTALL_TEST_PROFILE
        if ([regex]::Matches($profileText, [regex]::Escape('# Added by Engineering Toolkit installer')).Count -ne 1) {
            throw 'POSIX PATH update was not idempotent'
        }
    }

    $reportedVersion = (& $installed version --json | ConvertFrom-Json).version
    if ($reportedVersion -cne $Version) {
        throw "Installed executable reported $reportedVersion, want $Version"
    }
    Write-Host "Installer tests passed for $OS/$Arch"
} finally {
    $env:HOME = $originalHome
    $env:ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP = $originalFailure
    $env:ENGKIT_INSTALL_TEST_USER_PATH_FILE = $originalPathFile
    $env:ENGKIT_INSTALL_TEST_PROFILE = $originalProfile
    if ($null -ne $nativePreferenceVariable) {
        $PSNativeCommandUseErrorActionPreference = $originalNativePreference
    }
    if (Test-Path -LiteralPath $testRoot) {
        Remove-Item -LiteralPath $testRoot -Recurse -Force
    }
}

