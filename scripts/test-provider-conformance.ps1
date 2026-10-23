[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BinaryPath,

    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory,

    [Parameter(Mandatory = $true)]
    [ValidateSet('windows', 'linux', 'darwin')]
    [string]$OS,

    [Parameter(Mandatory = $true)]
    [ValidateSet('amd64', 'arm64')]
    [string]$Arch
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$resolvedBinary = (Resolve-Path -LiteralPath $BinaryPath).Path
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$testParent = Join-Path $repositoryRoot '.engkit-test-temp'
$testRoot = Join-Path $testParent ('provider-conformance-' + [guid]::NewGuid().ToString('N'))
$workspace = Join-Path $testRoot 'workspace'
$evidenceRoot = [System.IO.Path]::GetFullPath($OutputDirectory)
$utf8 = [System.Text.UTF8Encoding]::new($false)
$nativePreferenceVariable = Get-Variable -Name PSNativeCommandUseErrorActionPreference -ErrorAction SilentlyContinue
$originalNativePreference = if ($null -ne $nativePreferenceVariable) { $nativePreferenceVariable.Value } else { $null }
if ($null -ne $nativePreferenceVariable) {
    $PSNativeCommandUseErrorActionPreference = $false
}

function Invoke-EngkitJson {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    $lines = @(& $resolvedBinary @Arguments)
    if ($LASTEXITCODE -ne 0) {
        throw "engkit $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
    $text = $lines -join "`n"
    try {
        return $text | ConvertFrom-Json
    } catch {
        throw "engkit $($Arguments -join ' ') did not return valid JSON: $text"
    }
}

function Write-LFText {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Value
    )

    $parent = Split-Path -Parent $Path
    if ($parent) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    $normalized = $Value.Replace("`r`n", "`n").Replace("`r", "`n")
    [System.IO.File]::WriteAllText($Path, $normalized, $utf8)
}

function Get-ByteDigest {
    param([Parameter(Mandatory = $true)][byte[]]$Bytes)

    $hash = [System.Security.Cryptography.SHA256]::HashData($Bytes)
    return 'sha256:' + [Convert]::ToHexString($hash).ToLowerInvariant()
}

function Get-FileDigest {
    param([Parameter(Mandatory = $true)][string]$Path)

    return Get-ByteDigest -Bytes ([System.IO.File]::ReadAllBytes($Path))
}

function Get-Snapshot {
    param([Parameter(Mandatory = $true)][string[]]$Paths)

    $snapshot = [ordered]@{}
    foreach ($path in $Paths) {
        $snapshot[$path] = Get-FileDigest -Path $path
    }
    return $snapshot
}

function Assert-Snapshot {
    param([Parameter(Mandatory = $true)]$Snapshot)

    foreach ($entry in $Snapshot.GetEnumerator()) {
        if (-not (Test-Path -LiteralPath $entry.Key -PathType Leaf)) {
            throw "Preserved file is missing: $($entry.Key)"
        }
        $actual = Get-FileDigest -Path $entry.Key
        if ($actual -cne $entry.Value) {
            throw "Preserved file changed: $($entry.Key)"
        }
    }
}

function Assert-Condition {
    param(
        [Parameter(Mandatory = $true)][bool]$Condition,
        [Parameter(Mandatory = $true)][string]$Message
    )

    if (-not $Condition) {
        throw $Message
    }
}

New-Item -ItemType Directory -Path $testRoot, $evidenceRoot -Force | Out-Null
try {
    $catalog = Invoke-EngkitJson -Arguments @('catalog', 'inspect')
    $createPlan = Invoke-EngkitJson -Arguments @(
        'project', 'plan-create',
        '--destination', $workspace,
        '--profile', 'go-service@0.1.0',
        '--set', 'project_name=Provider Conformance Pilot'
    )
    Assert-Condition ($createPlan.executable -eq $true) 'Pilot project plan is not executable'
    $createResult = Invoke-EngkitJson -Arguments @(
        'project', 'create',
        '--destination', $workspace,
        '--profile', 'go-service@0.1.0',
        '--set', 'project_name=Provider Conformance Pilot',
        '--approve', $createPlan.plan_id
    )
    Assert-Condition ($createResult.status -ceq 'complete') 'Pilot project creation did not complete'

    $validation = Invoke-EngkitJson -Arguments @('skills', 'validate', '--root', $workspace, '--id', 'project-guide')
    Assert-Condition ($validation.valid -eq $true) 'Installed project-guide pilot is invalid'

    $specKit = Join-Path $workspace '.agents/skills/speckit-plan/SKILL.md'
    $unrelated = Join-Path $workspace 'provider-conformance-sentinel.txt'
    Write-LFText -Path $specKit -Value "official Spec Kit sentinel`n"
    Write-LFText -Path $unrelated -Value "unrelated workspace sentinel`n"

    $providerRoots = [ordered]@{
        codex    = '.agents/skills'
        claude   = '.claude/skills'
        opencode = '.opencode/skills'
    }
    $providerNative = @{}
    foreach ($provider in $providerRoots.Keys) {
        $path = Join-Path $workspace ($providerRoots[$provider] + '/provider-native/SKILL.md')
        Write-LFText -Path $path -Value "provider-native $provider sentinel`n"
        $providerNative[$provider] = $path
    }

    $canonicalManifest = Join-Path $workspace '.agents/skills/project-guide/skill.yaml'
    $canonicalInstructions = Join-Path $workspace '.agents/skills/project-guide/instructions.md'
    $preservedPaths = @($canonicalManifest, $canonicalInstructions, $specKit, $unrelated) + @($providerNative.Values)
    $preserved = Get-Snapshot -Paths $preservedPaths

    $goldenPath = Join-Path $repositoryRoot 'internal/providersync/testdata/project-guide.SKILL.md'
    $goldenText = [System.IO.File]::ReadAllText($goldenPath).Replace("`r`n", "`n").Replace("`r", "`n").TrimEnd("`n") + "`n"
    $goldenBytes = $utf8.GetBytes($goldenText)
    $goldenDigest = Get-ByteDigest -Bytes $goldenBytes

    $portableProviders = @()
    $runtimeProviders = @()
    foreach ($provider in $providerRoots.Keys) {
        $planArguments = @('skills', 'plan-sync', '--root', $workspace, '--provider', $provider, '--id', 'project-guide')
        $first = Invoke-EngkitJson -Arguments $planArguments
        $second = Invoke-EngkitJson -Arguments $planArguments
        Assert-Condition ($first.plan_id -ceq $second.plan_id) "$provider plan ID changed without workspace changes"
        Assert-Condition ($first.executable -eq $true) "$provider plan is not executable"
        Assert-Condition (@($first.outputs).Count -eq 1) "$provider plan did not produce exactly one output"
        Assert-Condition ($first.outputs[0].action -ceq 'create') "$provider first output action is not create"
        Assert-Condition ($first.outputs[0].after_sha256 -ceq $goldenDigest) "$provider output digest does not match the reviewed golden"
        Assert-Condition ($first.mappings[0].quality -ceq 'exact') "$provider mapping is not exact"

        $sync = Invoke-EngkitJson -Arguments @(
            'skills', 'sync', '--root', $workspace, '--provider', $provider,
            '--id', 'project-guide', '--approve', $first.plan_id
        )
        Assert-Condition ($sync.status -ceq 'complete') "$provider synchronization did not complete"
        $outputPath = Join-Path $workspace $first.outputs[0].path
        Assert-Condition ((Get-FileDigest -Path $outputPath) -ceq $goldenDigest) "$provider live output digest is incorrect"
        Assert-Snapshot -Snapshot $preserved

        $unchanged = Invoke-EngkitJson -Arguments $planArguments
        $unchangedAgain = Invoke-EngkitJson -Arguments $planArguments
        Assert-Condition ($unchanged.plan_id -ceq $unchangedAgain.plan_id) "$provider unchanged plan is not deterministic"
        Assert-Condition ($unchanged.outputs[0].action -ceq 'unchanged') "$provider repeated plan is not unchanged"

        $ownershipPath = Join-Path $workspace ".engkit/providers/$provider/project-guide.json"
        $ownership = Get-Content -Raw -LiteralPath $ownershipPath | ConvertFrom-Json
        Assert-Condition ($ownership.synchronization_plan_id -ceq $first.plan_id) "$provider ownership does not bind the publishing plan"
        Assert-Condition ($ownership.outputs[0].sha256 -ceq $goldenDigest) "$provider ownership output digest is incorrect"

        $unsyncArguments = @('skills', 'plan-unsync', '--root', $workspace, '--provider', $provider, '--id', 'project-guide')
        $unsyncPlan = Invoke-EngkitJson -Arguments $unsyncArguments
        $unsyncAgain = Invoke-EngkitJson -Arguments $unsyncArguments
        Assert-Condition ($unsyncPlan.plan_id -ceq $unsyncAgain.plan_id) "$provider unsync plan is not deterministic"
        $unsync = Invoke-EngkitJson -Arguments @(
            'skills', 'unsync', '--root', $workspace, '--provider', $provider,
            '--id', 'project-guide', '--approve', $unsyncPlan.plan_id
        )
        Assert-Condition ($unsync.status -ceq 'complete') "$provider unsynchronization did not complete"
        Assert-Condition (-not (Test-Path -LiteralPath $outputPath)) "$provider output remains after unsync"
        Assert-Condition (-not (Test-Path -LiteralPath $ownershipPath)) "$provider ownership remains after unsync"
        Assert-Snapshot -Snapshot $preserved

        $findingCodes = @($first.mappings[0].findings | ForEach-Object { $_.code } | Sort-Object)
        $portableProviders += [ordered]@{
            provider         = $provider
            contract_version = $first.adapter.contract_version
            evidence_date    = $first.adapter.evidence_date
            target_root      = $first.adapter.target_root
            source_version   = $first.sources[0].version
            package_sha256   = $first.sources[0].package_sha256
            manifest_sha256  = $first.sources[0].manifest_sha256
            mapping_quality  = $first.mappings[0].quality
            finding_codes    = $findingCodes
            output_path      = $first.outputs[0].path
            output_sha256    = $first.outputs[0].after_sha256
        }
        $runtimeProviders += [ordered]@{
            provider          = $provider
            sync_plan_id      = $first.plan_id
            unchanged_plan_id = $unchanged.plan_id
            unsync_plan_id    = $unsyncPlan.plan_id
        }
    }

    $template = @($catalog.templates | Where-Object { $_.id -ceq 'project-guide' -and $_.version -ceq '0.1.0' })
    $profile = @($catalog.profiles | Where-Object { $_.id -ceq 'go-service' -and $_.version -ceq '0.1.0' })
    Assert-Condition ($template.Count -eq 1) 'Packaged project-guide@0.1.0 template is unavailable'
    Assert-Condition ($profile.Count -eq 1) 'Packaged go-service@0.1.0 profile is unavailable'

    $portable = [ordered]@{
        format          = 'engineering-toolkit-provider-conformance-internal-v1'
        catalog_id      = $catalog.id
        catalog_version = $catalog.version
        catalog_sha256  = $catalog.sha256
        template_sha256 = $template[0].sha256
        profile_sha256  = $profile[0].sha256
        pilot_id        = 'project-guide'
        pilot_version   = '0.1.0'
        golden_sha256   = $goldenDigest
        providers       = $portableProviders
    }
    $reportedVersion = (Invoke-EngkitJson -Arguments @('version', '--json')).version
    $runtime = [ordered]@{
        format         = 'engineering-toolkit-provider-conformance-runtime-internal-v1'
        os             = $OS
        arch           = $Arch
        engkit_version = $reportedVersion
        providers      = $runtimeProviders
    }
    Write-LFText -Path (Join-Path $evidenceRoot 'portable.json') -Value (($portable | ConvertTo-Json -Depth 10) + "`n")
    Write-LFText -Path (Join-Path $evidenceRoot 'runtime.json') -Value (($runtime | ConvertTo-Json -Depth 10) + "`n")
    Write-Host "Provider conformance passed for $OS/$Arch"
} finally {
    if ($null -ne $nativePreferenceVariable) {
        $PSNativeCommandUseErrorActionPreference = $originalNativePreference
    }
    if (Test-Path -LiteralPath $testRoot) {
        Remove-Item -LiteralPath $testRoot -Recurse -Force
    }
    if ((Test-Path -LiteralPath $testParent) -and @((Get-ChildItem -LiteralPath $testParent -Force)).Count -eq 0) {
        Remove-Item -LiteralPath $testParent -Force
    }
}
