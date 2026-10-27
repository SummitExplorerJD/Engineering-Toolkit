[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BinaryPath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v0\.3\.0(?:-rc\.[1-9][0-9]*)?$')]
    [string]$Version,

    [string]$OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$resolvedBinary = (Resolve-Path -LiteralPath $BinaryPath).Path
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("engkit-v030-smoke-" + [guid]::NewGuid().ToString('N'))
}
$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $resolvedOutput) {
    if (@(Get-ChildItem -LiteralPath $resolvedOutput -Force).Count -ne 0) {
        throw "Smoke output directory is not empty: $resolvedOutput"
    }
} else {
    New-Item -ItemType Directory -Path $resolvedOutput | Out-Null
}

function Invoke-EngkitJSON {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments,

        [int[]]$AllowedExitCodes = @(0)
    )

    $stdoutPath = Join-Path $resolvedOutput ("stdout-" + [guid]::NewGuid().ToString('N') + '.json')
    $stderrPath = Join-Path $resolvedOutput ("stderr-" + [guid]::NewGuid().ToString('N') + '.txt')
    try {
        & $resolvedBinary @Arguments 1> $stdoutPath 2> $stderrPath
        $exitCode = $LASTEXITCODE
        if ($AllowedExitCodes -notcontains $exitCode) {
            $stderrText = Get-Content -Raw -LiteralPath $stderrPath
            throw "engkit $($Arguments -join ' ') exited with $exitCode. stderr: $stderrText"
        }
        $stdoutText = Get-Content -Raw -LiteralPath $stdoutPath
        if ([string]::IsNullOrWhiteSpace($stdoutText)) {
            throw "engkit $($Arguments -join ' ') produced no JSON output"
        }
        return $stdoutText | ConvertFrom-Json -Depth 100
    } finally {
        Remove-Item -LiteralPath $stdoutPath, $stderrPath -Force -ErrorAction SilentlyContinue
    }
}

function Assert-Equal {
    param($Actual, $Expected, [string]$Description)
    if ($Actual -ne $Expected) {
        throw "$Description was '$Actual', want '$Expected'"
    }
}

function Get-TreeDigest {
    param([Parameter(Mandatory = $true)][string]$Root)

    $lines = foreach ($file in Get-ChildItem -LiteralPath $Root -File -Recurse | Sort-Object FullName) {
        $relative = [System.IO.Path]::GetRelativePath($Root, $file.FullName).Replace('\', '/')
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $file.FullName).Hash.ToLowerInvariant()
        "$relative $hash"
    }
    $bytes = [System.Text.Encoding]::UTF8.GetBytes(($lines -join "`n"))
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        return ([System.BitConverter]::ToString($sha.ComputeHash($bytes))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha.Dispose()
    }
}

function Write-UTF8NoBOM {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][AllowEmptyString()][string]$Contents)
    [System.IO.File]::WriteAllText($Path, $Contents, [System.Text.UTF8Encoding]::new($false))
}

function Test-MCPServer {
    param([Parameter(Mandatory = $true)][string]$Root)

    $inputPath = Join-Path $resolvedOutput 'mcp-input.jsonl'
    $stdoutPath = Join-Path $resolvedOutput 'mcp-stdout.jsonl'
    $stderrPath = Join-Path $resolvedOutput 'mcp-stderr.txt'
    $initializeRequest = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"engkit-release-smoke","version":"1.0.0"}}}'
    $initializedNotification = '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}'
    $toolsRequest = '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
    Write-UTF8NoBOM -Path $inputPath -Contents ((@($initializeRequest, $initializedNotification, $toolsRequest) -join "`n") + "`n")

    $startInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $startInfo.FileName = $resolvedBinary
    foreach ($argument in @('mcp', 'serve', '--transport', 'stdio', '--root', $Root)) {
        $startInfo.ArgumentList.Add($argument)
    }
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    if (-not $process.Start()) {
        throw 'Could not start the MCP stdio server'
    }

    try {
        $process.StandardInput.WriteLine($initializeRequest)
        $process.StandardInput.Flush()
        $initializeTask = $process.StandardOutput.ReadLineAsync()
        if (-not $initializeTask.Wait(15000)) {
            throw 'MCP stdio server did not answer initialize within 15 seconds'
        }
        $initializeLine = $initializeTask.Result

        $process.StandardInput.WriteLine($initializedNotification)
        $process.StandardInput.WriteLine($toolsRequest)
        $process.StandardInput.Flush()
        $toolsTask = $process.StandardOutput.ReadLineAsync()
        if (-not $toolsTask.Wait(15000)) {
            throw 'MCP stdio server did not answer tools/list within 15 seconds'
        }
        $toolsLine = $toolsTask.Result
        Write-UTF8NoBOM -Path $stdoutPath -Contents ((@($initializeLine, $toolsLine) -join "`n") + "`n")

        $initialize = $initializeLine | ConvertFrom-Json -Depth 100
        $toolResponse = $toolsLine | ConvertFrom-Json -Depth 100
        $process.StandardInput.Close()
        if (-not $process.WaitForExit(15000)) {
            throw 'MCP stdio server did not stop after input reached EOF'
        }
        $stderrText = $process.StandardError.ReadToEnd()
        Write-UTF8NoBOM -Path $stderrPath -Contents $stderrText
        if ($process.ExitCode -ne 0) {
            throw "MCP stdio server exited with $($process.ExitCode). stderr: $stderrText"
        }
    } finally {
        if (-not $process.HasExited) {
            $process.Kill($true)
            $process.WaitForExit()
        }
        $process.Dispose()
    }

    Assert-Equal $initialize.result.protocolVersion '2025-11-25' 'MCP negotiated protocol version'
    $capabilities = $initialize.result.capabilities
    $capabilityNames = @($capabilities.PSObject.Properties.Name)
    if ($null -eq $capabilities.tools -or $capabilityNames -contains 'prompts' -or $capabilityNames -contains 'resources') {
        throw 'MCP server advertised unexpected capabilities'
    }
    $actualTools = @($toolResponse.result.tools | ForEach-Object { $_.name } | Sort-Object)
    $expectedTools = @('catalog_inspect', 'context_search', 'engkit_doctor', 'plan_inspect', 'profiles_recommend', 'skills_list', 'skills_validate', 'workspace_inspect')
    if (($actualTools -join ',') -ne ($expectedTools -join ',')) {
        throw "MCP tools were '$($actualTools -join ',')', want '$($expectedTools -join ',')'"
    }
}

try {
    $versionResult = Invoke-EngkitJSON -Arguments @('version', '--json')
    Assert-Equal $versionResult.version $Version 'Release version'

    $catalog = Invoke-EngkitJSON -Arguments @('catalog', 'inspect')
    Assert-Equal $catalog.schema_version 'engkit.catalog/v1alpha1' 'Catalog schema'
    Assert-Equal $catalog.version '0.3.0' 'Catalog version'
    $profileIDs = @($catalog.profiles | ForEach-Object { "$($_.id)@$($_.version)" } | Sort-Object)
    $expectedProfiles = @('go-cli@0.1.0', 'go-library@0.1.0', 'go-service@0.1.0')
    if (($profileIDs -join ',') -ne ($expectedProfiles -join ',')) {
        throw "Catalog profiles were '$($profileIDs -join ',')', want '$($expectedProfiles -join ',')'"
    }

    $emptyTarget = Join-Path $resolvedOutput 'empty-workspace'
    New-Item -ItemType Directory -Path $emptyTarget | Out-Null
    $workspacePlan = Invoke-EngkitJSON -Arguments @('init', 'plan', '--target', $emptyTarget)
    Assert-Equal $workspacePlan.schema_version 'engkit.plan/v1alpha1' 'Workspace initialization plan schema'

    $newTarget = Join-Path $resolvedOutput 'new-go-cli'
    $projectPlan = Invoke-EngkitJSON -Arguments @('init', 'plan', '--target', $newTarget, '--profile', 'go-cli@0.1.0', '--set', 'project_name=SyntheticCLI')
    Assert-Equal $projectPlan.schema_version 'engkit.composition-plan/v1alpha1' 'Project creation plan schema'

    $ambiguousTarget = Join-Path $resolvedOutput 'ambiguous-workspace'
    New-Item -ItemType Directory -Path $ambiguousTarget | Out-Null
    Write-UTF8NoBOM -Path (Join-Path $ambiguousTarget 'existing.txt') -Contents "synthetic`n"
    & $resolvedBinary init plan --target $ambiguousTarget --profile go-cli@0.1.0 1> $null 2> $null
    if ($LASTEXITCODE -eq 0) {
        throw 'Unified init accepted a non-empty uninitialized target with a profile'
    }

    $workspace = Join-Path $resolvedOutput 'initialized-go'
    $engkitDirectory = Join-Path $workspace '.engkit'
    New-Item -ItemType Directory -Path $engkitDirectory -Force | Out-Null
    Write-UTF8NoBOM -Path (Join-Path $workspace 'go.mod') -Contents "module example.invalid/synthetic`n`ngo 1.27.3`n"
    Write-UTF8NoBOM -Path (Join-Path $engkitDirectory 'config.yaml') -Contents "schema_version: engkit.config/v1alpha1`nworkspace:`n  source_layout: auto`nintelligence:`n  enabled: false`n"
    Write-UTF8NoBOM -Path (Join-Path $engkitDirectory '.gitignore') -Contents "/cache/*`n/context/*.db`n/context/*.db-*`n/recovery/*`n"

    $beforeDigest = Get-TreeDigest -Root $workspace
    $recommendations = Invoke-EngkitJSON -Arguments @('profiles', 'recommend', '--root', $workspace)
    Assert-Equal $recommendations.schema_version 'engkit.profile-recommendation/v1alpha1' 'Profile recommendation schema'
    Assert-Equal $recommendations.catalog_version '0.3.0' 'Profile recommendation catalog version'
    $compatible = @($recommendations.recommendations | Where-Object { $_.compatible })
    Assert-Equal $compatible.Count 3 'Compatible Go profile count'

    $disabledAdvice = Invoke-EngkitJSON -Arguments @('advise', 'profiles', '--root', $workspace, '--goal', 'Select synthetic Go guidance')
    Assert-Equal $disabledAdvice.schema_version 'engkit.advice-result/v1alpha1' 'Advice schema'
    Assert-Equal $disabledAdvice.mode 'deterministic' 'Disabled advice mode'
    if (($disabledAdvice.warnings -join ' ') -notmatch 'disabled') {
        throw 'Disabled local intelligence did not report deterministic fallback'
    }
    $afterDigest = Get-TreeDigest -Root $workspace
    Assert-Equal $afterDigest $beforeDigest 'Read-only workspace digest'

    Write-UTF8NoBOM -Path (Join-Path $engkitDirectory 'config.yaml') -Contents "schema_version: engkit.config/v1alpha1`nworkspace:`n  source_layout: auto`nintelligence:`n  enabled: true`n  provider: ollama`n  model: granite4:3b`n"
    $missingContextAdvice = Invoke-EngkitJSON -Arguments @('advise', 'profiles', '--root', $workspace, '--goal', 'Select synthetic Go guidance')
    Assert-Equal $missingContextAdvice.mode 'deterministic' 'Missing-context advice mode'
    if (($missingContextAdvice.warnings -join ' ') -notmatch 'context') {
        throw 'Missing SQLite context did not produce its deterministic fallback warning'
    }

    & $resolvedBinary advise profiles --root $workspace --goal synthetic --model qwen3.5:4b 1> $null 2> $null
    if ($LASTEXITCODE -eq 0) {
        throw 'Local intelligence accepted a model outside the allowlist'
    }

    $savedPath = $env:PATH
    try {
        $env:PATH = $resolvedOutput
        $doctor = Invoke-EngkitJSON -Arguments @('doctor', '--root', $workspace, '--scope', 'integrations') -AllowedExitCodes @(0, 1)
    } finally {
        $env:PATH = $savedPath
    }
    Assert-Equal $doctor.schema_version 'engkit.doctor-result/v1alpha1' 'Doctor schema'
    $ollamaFinding = $doctor.findings | Where-Object { $_.code -eq 'integration.ollama.executable' } | Select-Object -First 1
    Assert-Equal $ollamaFinding.status 'unavailable' 'Ollama absence status'
    $mcpFinding = $doctor.findings | Where-Object { $_.code -eq 'integration.mcp.server' } | Select-Object -First 1
    Assert-Equal $mcpFinding.status 'pass' 'MCP availability status'

    Test-MCPServer -Root $workspace

    $evidence = [ordered]@{
        schema_version = 'engkit.release-qualification/v1alpha1'
        version = $Version
        binary = [System.IO.Path]::GetFileName($resolvedBinary)
        catalog_version = $catalog.version
        profiles = $profileIDs
        scenarios = @(
            'catalog-and-profile-inventory',
            'unified-init-classification',
            'read-only-profile-recommendation',
            'disabled-local-intelligence',
            'missing-sqlite-fallback',
            'model-allowlist-rejection',
            'optional-integration-absence',
            'mcp-stdio-handshake-and-tool-inventory'
        )
    }
    $evidencePath = Join-Path $resolvedOutput 'v030-release-qualification.json'
    Write-UTF8NoBOM -Path $evidencePath -Contents (($evidence | ConvertTo-Json -Depth 10) + "`n")
    Write-Host "v0.3.0 native qualification passed for $resolvedBinary"
    Write-Host "Evidence written to $evidencePath"
} catch {
    Write-Error $_
    throw
}
