[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$EvidenceDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = (Resolve-Path -LiteralPath $EvidenceDirectory).Path
$portableFiles = @(Get-ChildItem -LiteralPath $root -Filter portable.json -File -Recurse | Sort-Object FullName)
if ($portableFiles.Count -ne 6) {
    throw "Expected six portable conformance reports, found $($portableFiles.Count)"
}

$expected = [System.IO.File]::ReadAllBytes($portableFiles[0].FullName)
$expectedHash = (Get-FileHash -LiteralPath $portableFiles[0].FullName -Algorithm SHA256).Hash.ToLowerInvariant()
foreach ($file in $portableFiles) {
    $actual = [System.IO.File]::ReadAllBytes($file.FullName)
    if (-not [System.Linq.Enumerable]::SequenceEqual($expected, $actual)) {
        $actualHash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        throw "Portable evidence differs: $($file.FullName) has $actualHash, want $expectedHash"
    }
}

$runtimeFiles = @(Get-ChildItem -LiteralPath $root -Filter runtime.json -File -Recurse | Sort-Object FullName)
if ($runtimeFiles.Count -ne 6) {
    throw "Expected six runtime conformance reports, found $($runtimeFiles.Count)"
}
$platforms = @{}
foreach ($file in $runtimeFiles) {
    $runtime = Get-Content -Raw -LiteralPath $file.FullName | ConvertFrom-Json
    $key = "$($runtime.os)/$($runtime.arch)"
    if ($platforms.ContainsKey($key)) {
        throw "Duplicate runtime conformance report for $key"
    }
    $platforms[$key] = $true
}
$expectedPlatforms = @('windows/amd64', 'windows/arm64', 'linux/amd64', 'linux/arm64', 'darwin/amd64', 'darwin/arm64')
foreach ($platform in $expectedPlatforms) {
    if (-not $platforms.ContainsKey($platform)) {
        throw "Runtime conformance report is missing for $platform"
    }
}

Write-Host "Provider conformance evidence matches across all six platforms ($expectedHash)"
