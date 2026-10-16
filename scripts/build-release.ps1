[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v[0-9]+\.[0-9]+\.[0-9]+$')]
    [string]$Version,

    [string]$OutputRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
if ([string]::IsNullOrWhiteSpace($OutputRoot)) {
    $OutputRoot = Join-Path $repositoryRoot 'dist'
}
$outputDirectory = Join-Path $OutputRoot $Version

if (Test-Path -LiteralPath $outputDirectory) {
    $existing = @(Get-ChildItem -LiteralPath $outputDirectory -Force)
    if ($existing.Count -ne 0) {
        throw "Release output already exists and is not empty: $outputDirectory"
    }
} else {
    New-Item -ItemType Directory -Path $outputDirectory -Force | Out-Null
}

$module = 'github.com/SummitExplorerJD/Engineering-Toolkit'
$targets = @(
    @{ OS = 'windows'; Arch = 'amd64'; Extension = '.exe' },
    @{ OS = 'windows'; Arch = 'arm64'; Extension = '.exe' },
    @{ OS = 'linux'; Arch = 'amd64'; Extension = '' },
    @{ OS = 'linux'; Arch = 'arm64'; Extension = '' },
    @{ OS = 'darwin'; Arch = 'amd64'; Extension = '' },
    @{ OS = 'darwin'; Arch = 'arm64'; Extension = '' }
)

$savedGOOS = $env:GOOS
$savedGOARCH = $env:GOARCH
$savedCGOEnabled = $env:CGO_ENABLED
$artifacts = @()

Push-Location $repositoryRoot
try {
    foreach ($target in $targets) {
        $env:GOOS = $target.OS
        $env:GOARCH = $target.Arch
        $env:CGO_ENABLED = '0'

        $filename = "engkit-$Version-$($target.OS)-$($target.Arch)$($target.Extension)"
        $destination = Join-Path $outputDirectory $filename
        $linkerFlags = "-s -w -buildid= -X $module/internal/cli.Version=$Version"

        Write-Host "Building $filename"
        & go build -trimpath -buildvcs=false -ldflags $linkerFlags -o $destination ./cmd/engkit
        if ($LASTEXITCODE -ne 0) {
            throw "Go build failed for $($target.OS)/$($target.Arch)"
        }

        $file = Get-Item -LiteralPath $destination
        $digest = (Get-FileHash -Algorithm SHA256 -LiteralPath $destination).Hash.ToLowerInvariant()
        $artifacts += [ordered]@{
            filename = $file.Name
            os       = $target.OS
            arch     = $target.Arch
            size     = $file.Length
            sha256   = $digest
        }
    }
} finally {
    Pop-Location
    $env:GOOS = $savedGOOS
    $env:GOARCH = $savedGOARCH
    $env:CGO_ENABLED = $savedCGOEnabled
}

Copy-Item -LiteralPath (Join-Path $repositoryRoot 'LICENSE') -Destination (Join-Path $outputDirectory 'LICENSE')

$goVersion = (& go version).Trim()
$manifest = [ordered]@{
    schema_version = 'engkit.release-manifest/v1alpha1'
    version        = $Version
    go_version     = $goVersion
    cgo_enabled    = $false
    artifacts      = $artifacts
}
$manifestPath = Join-Path $outputDirectory 'release-manifest.json'
$manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestPath -Encoding utf8NoBOM

$checksumFiles = Get-ChildItem -LiteralPath $outputDirectory -File |
    Where-Object { $_.Name -ne 'SHA256SUMS' } |
    Sort-Object Name
$checksumLines = foreach ($file in $checksumFiles) {
    $digest = (Get-FileHash -Algorithm SHA256 -LiteralPath $file.FullName).Hash.ToLowerInvariant()
    "$digest  $($file.Name)"
}
$checksumLines | Set-Content -LiteralPath (Join-Path $outputDirectory 'SHA256SUMS') -Encoding ascii

if ($IsWindows -and $env:PROCESSOR_ARCHITECTURE -eq 'AMD64') {
    $smokeBinary = Join-Path $outputDirectory "engkit-$Version-windows-amd64.exe"
    $versionResult = (& $smokeBinary version --json | ConvertFrom-Json).version
    if ($versionResult -ne $Version) {
        throw "Release smoke test reported $versionResult, want $Version"
    }
}

Write-Host "Release candidate created at $outputDirectory"
