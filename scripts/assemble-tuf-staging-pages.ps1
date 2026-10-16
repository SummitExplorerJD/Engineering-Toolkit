[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v[0-9]+\.[0-9]+\.[0-9]+$')]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$MetadataRoot,

    [Parameter(Mandatory = $true)]
    [string]$BuildRoot,

    [Parameter(Mandatory = $true)]
    [string]$OutputRoot
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$metadataDirectory = (Resolve-Path -LiteralPath $MetadataRoot).Path
$buildDirectory = (Resolve-Path -LiteralPath $BuildRoot).Path

if (Test-Path -LiteralPath $OutputRoot) {
    $existing = @(Get-ChildItem -LiteralPath $OutputRoot -Force)
    if ($existing.Count -ne 0) {
        throw "Pages output already exists and is not empty: $OutputRoot"
    }
} else {
    New-Item -ItemType Directory -Path $OutputRoot -Force | Out-Null
}
$outputDirectory = (Resolve-Path -LiteralPath $OutputRoot).Path

$privateMaterial = @(Get-ChildItem -LiteralPath $metadataDirectory -Recurse -File |
    Where-Object { $_.Name -like '*.private.pem' -or (Get-Content -LiteralPath $_.FullName -Raw) -match '"private"\s*:' })
if ($privateMaterial.Count -ne 0) {
    throw 'Private key material was found in the public staging metadata input.'
}

function Read-JsonFile([string]$Path) {
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json -Depth 100
}

function Assert-NotExpired($Metadata, [string]$Role) {
    $expires = [DateTimeOffset]::Parse($Metadata.signed.expires)
    if ($expires -le [DateTimeOffset]::UtcNow) {
        throw "$Role metadata expired at $expires. Regenerate staging metadata before deployment."
    }
}

function Assert-RoleThreshold($Root, $Metadata, [string]$Role) {
    $rolePolicy = $Root.signed.roles.$Role
    if ($null -eq $rolePolicy) {
        throw "Root metadata does not define role $Role."
    }
    $authorized = @($rolePolicy.keyids)
    $validKeyIds = @($Metadata.signatures |
        Where-Object { $_.keyid -in $authorized } |
        Select-Object -ExpandProperty keyid -Unique)
    if ($validKeyIds.Count -lt [int]$rolePolicy.threshold) {
        throw "$Role metadata has $($validKeyIds.Count) authorized signatures; threshold is $($rolePolicy.threshold)."
    }
}

$root = Read-JsonFile (Join-Path $metadataDirectory 'root.json')
$targets = Read-JsonFile (Join-Path $metadataDirectory '1.targets.json')
$snapshot = Read-JsonFile (Join-Path $metadataDirectory '1.snapshot.json')
$timestamp = Read-JsonFile (Join-Path $metadataDirectory 'timestamp.json')

if ($root.signed._type -ne 'root' -or -not $root.signed.consistent_snapshot) {
    throw 'Staging root must be root metadata with consistent snapshots enabled.'
}
if ([int]$root.signed.roles.root.threshold -ne 2 -or [int]$root.signed.roles.targets.threshold -ne 2) {
    throw 'Staging root and targets roles must use 2-of-3 thresholds.'
}

Assert-NotExpired $root 'root'
Assert-NotExpired $targets 'targets'
Assert-NotExpired $snapshot 'snapshot'
Assert-NotExpired $timestamp 'timestamp'
Assert-RoleThreshold $root $root 'root'
Assert-RoleThreshold $root $targets 'targets'
Assert-RoleThreshold $root $snapshot 'snapshot'
Assert-RoleThreshold $root $timestamp 'timestamp'

$targetProperties = @($targets.signed.targets.PSObject.Properties)
if ($targetProperties.Count -ne 6) {
    throw "Expected six updater targets, found $($targetProperties.Count)."
}

$publicMetadata = Join-Path $outputDirectory 'metadata'
$publicTargets = Join-Path $outputDirectory 'targets'
New-Item -ItemType Directory -Path $publicMetadata, $publicTargets -Force | Out-Null
Get-ChildItem -LiteralPath $metadataDirectory -Filter '*.json' -File |
    Copy-Item -Destination $publicMetadata

$publishedTargets = @()
foreach ($property in $targetProperties) {
    $targetPath = $property.Name
    $target = $property.Value
    $custom = $target.custom
    if ($custom.schema -ne 'engkit.release-target/v1alpha1' -or
        $custom.version -ne $Version -or
        $custom.channel -ne 'stable' -or
        $custom.format -ne 'executable') {
        throw "Unexpected signed custom metadata for $targetPath."
    }

    $expectedLeaf = if ($custom.os -eq 'windows') { 'engkit.exe' } else { 'engkit' }
    $expectedPath = "releases/stable/$Version/$($custom.os)-$($custom.arch)/$expectedLeaf"
    if ($targetPath -ne $expectedPath) {
        throw "Signed target path $targetPath does not match $expectedPath."
    }

    $extension = if ($custom.os -eq 'windows') { '.exe' } else { '' }
    $sourceName = "engkit-$Version-$($custom.os)-$($custom.arch)$extension"
    $sourcePath = Join-Path $buildDirectory $sourceName
    if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
        throw "Release candidate is missing $sourceName."
    }

    $source = Get-Item -LiteralPath $sourcePath
    $digest = (Get-FileHash -LiteralPath $sourcePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($source.Length -ne [int64]$target.length -or $digest -ne $target.hashes.sha256) {
        throw "Release candidate does not match signed length and SHA-256 for $targetPath."
    }

    $relativeDirectory = (($targetPath -replace '/', [IO.Path]::DirectorySeparatorChar) | Split-Path -Parent)
    $targetDirectory = Join-Path $publicTargets $relativeDirectory
    New-Item -ItemType Directory -Path $targetDirectory -Force | Out-Null
    $destination = Join-Path $targetDirectory "$digest.$expectedLeaf"
    Copy-Item -LiteralPath $sourcePath -Destination $destination
    $publishedTargets += [ordered]@{ path = $targetPath; length = $source.Length; sha256 = $digest }
}

Set-Content -LiteralPath (Join-Path $outputDirectory '.nojekyll') -Value '' -Encoding ascii
$rootDigest = (Get-FileHash -LiteralPath (Join-Path $publicMetadata 'root.json') -Algorithm SHA256).Hash.ToLowerInvariant()
$index = @"
<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Engineering Toolkit TUF staging</title></head>
<body>
<h1>Engineering Toolkit TUF staging</h1>
<p><strong>Not for production use.</strong> This repository uses provisional software keys.</p>
<p>Release: $Version</p>
<p>Bootstrap root SHA-256: <code>$rootDigest</code></p>
<p><a href="metadata/root.json">Bootstrap root metadata</a></p>
</body>
</html>
"@
Set-Content -LiteralPath (Join-Path $outputDirectory 'index.html') -Value $index -Encoding utf8NoBOM

$deploymentManifest = [ordered]@{
    environment = 'staging'
    production_approved = $false
    version = $Version
    root_sha256 = $rootDigest
    targets = $publishedTargets
}
$deploymentManifest | ConvertTo-Json -Depth 5 |
    Set-Content -LiteralPath (Join-Path $outputDirectory 'deployment-manifest.json') -Encoding utf8NoBOM

Write-Host "TUF staging Pages site assembled at $outputDirectory"
Write-Host "Bootstrap root SHA-256: $rootDigest"
