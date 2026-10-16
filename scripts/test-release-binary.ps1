[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BinaryPath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-rc\.[1-9][0-9]*)?$')]
    [string]$Version
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$resolvedBinary = (Resolve-Path -LiteralPath $BinaryPath).Path

if (-not $IsWindows) {
    & chmod +x $resolvedBinary
    if ($LASTEXITCODE -ne 0) {
        throw "Could not make the release binary executable: $resolvedBinary"
    }
}

$textVersion = (& $resolvedBinary version).Trim()
if ($LASTEXITCODE -ne 0) {
    throw "Release binary version command failed with exit code $LASTEXITCODE"
}
$expectedTextVersion = "engkit $Version"
if ($textVersion -ne $expectedTextVersion) {
    throw "Release binary reported '$textVersion', want '$expectedTextVersion'"
}

$jsonVersion = (& $resolvedBinary version --json | ConvertFrom-Json).version
if ($LASTEXITCODE -ne 0) {
    throw "Release binary JSON version command failed with exit code $LASTEXITCODE"
}
if ($jsonVersion -ne $Version) {
    throw "Release binary JSON reported '$jsonVersion', want '$Version'"
}

& $resolvedBinary --help | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Release binary help command failed with exit code $LASTEXITCODE"
}

Write-Host "Native release smoke test passed for $resolvedBinary"
