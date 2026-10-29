[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BinaryPath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v(?:0\.4\.0|0\.5\.0)(?:-rc\.[1-9][0-9]*)?$')]
    [string]$Version,

    [string]$OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$resolvedBinary = (Resolve-Path -LiteralPath $BinaryPath).Path
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("engkit-v040-smoke-" + [guid]::NewGuid().ToString('N'))
}
$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $resolvedOutput) {
    if (@(Get-ChildItem -LiteralPath $resolvedOutput -Force).Count -ne 0) {
        throw "Smoke output directory is not empty: $resolvedOutput"
    }
} else {
    New-Item -ItemType Directory -Path $resolvedOutput | Out-Null
}

function Write-UTF8NoBOM {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][AllowEmptyString()][string]$Contents
    )
    [System.IO.File]::WriteAllText($Path, $Contents, [System.Text.UTF8Encoding]::new($false))
}

function Assert-Equal {
    param($Actual, $Expected, [string]$Description)
    if ($Actual -ne $Expected) {
        throw "$Description did not match the expected value"
    }
}

function Invoke-EngkitJSON {
    param(
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [int[]]$AllowedExitCodes = @(0)
    )

    $stdoutPath = Join-Path $resolvedOutput ("stdout-" + [guid]::NewGuid().ToString('N') + '.json')
    $stderrPath = Join-Path $resolvedOutput ("stderr-" + [guid]::NewGuid().ToString('N') + '.txt')
    try {
        & $resolvedBinary @Arguments 1> $stdoutPath 2> $stderrPath
        $exitCode = $LASTEXITCODE
        if ($AllowedExitCodes -notcontains $exitCode) {
            throw "engkit command exited with an unexpected status"
        }
        $stdoutText = Get-Content -Raw -LiteralPath $stdoutPath
        if ([string]::IsNullOrWhiteSpace($stdoutText)) {
            throw 'engkit command produced no JSON output'
        }
        return $stdoutText | ConvertFrom-Json -Depth 100
    } finally {
        Remove-Item -LiteralPath $stdoutPath, $stderrPath -Force -ErrorAction SilentlyContinue
    }
}

function Invoke-ExpectedFailure {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)

    & $resolvedBinary @Arguments 1> $null 2> $null
    $exitCode = $LASTEXITCODE
    if ($exitCode -eq 0) {
        throw 'engkit command unexpectedly succeeded'
    }
    $global:LASTEXITCODE = 0
}

function Send-MCPRequest {
    param(
        [Parameter(Mandatory = $true)][System.Diagnostics.Process]$Process,
        [Parameter(Mandatory = $true)]$Request,
        [int]$MaximumBytes = 1048576
    )

    $requestJSON = $Request | ConvertTo-Json -Depth 20 -Compress
    $Process.StandardInput.WriteLine($requestJSON)
    $Process.StandardInput.Flush()
    $readTask = $Process.StandardOutput.ReadLineAsync()
    if (-not $readTask.Wait(15000)) {
        throw 'MCP request timed out'
    }
    $line = $readTask.Result
    if ([string]::IsNullOrWhiteSpace($line)) {
        throw 'MCP request returned an empty response'
    }
    if ([System.Text.Encoding]::UTF8.GetByteCount($line) -gt $MaximumBytes) {
        throw 'MCP response exceeded its encoded ceiling'
    }
    return $line | ConvertFrom-Json -Depth 100
}

function Assert-MCPResult {
    param([Parameter(Mandatory = $true)]$Response)

    if ($null -ne $Response.PSObject.Properties['error']) {
        throw 'MCP tool call returned a protocol error'
    }
    if ($null -eq $Response.result -or ($null -ne $Response.result.PSObject.Properties['isError'] -and $Response.result.isError -eq $true)) {
        throw 'MCP tool call returned an application error'
    }
    if ($null -eq $Response.result.structuredContent) {
        throw 'MCP compact result omitted structured content'
    }
    if (@($Response.result.content).Count -ne 1 -or $Response.result.content[0].type -ne 'text') {
        throw 'MCP compact result did not contain one text summary'
    }
}

function Assert-SafeIndexedPath {
    param([Parameter(Mandatory = $true)][string]$Path)

    $normalized = $Path.Replace('\', '/').ToLowerInvariant()
    foreach ($forbidden in @('private', '.env', '.git', 'dist', '.engkit-private')) {
        if ($normalized -eq $forbidden -or $normalized.StartsWith($forbidden + '/')) {
            throw 'Excluded path appeared in an indexed result'
        }
    }
}

function Test-MCPContext {
    param(
        [Parameter(Mandatory = $true)][string]$Root,
        [Parameter(Mandatory = $true)]$CLIStatus,
        [Parameter(Mandatory = $true)]$CLISearch,
        [Parameter(Mandatory = $true)]$CLIRead
    )

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
        $initialize = Send-MCPRequest -Process $process -Request ([ordered]@{
            jsonrpc = '2.0'
            id = 1
            method = 'initialize'
            params = [ordered]@{
                protocolVersion = '2025-11-25'
                capabilities = [ordered]@{}
                clientInfo = [ordered]@{ name = 'engkit-v040-release-smoke'; version = '1.0.0' }
            }
        })
        Assert-Equal $initialize.result.protocolVersion '2025-11-25' 'MCP protocol version'
        $capabilityNames = @($initialize.result.capabilities.PSObject.Properties.Name)
        if ($null -eq $initialize.result.capabilities.tools -or $capabilityNames -contains 'prompts' -or $capabilityNames -contains 'resources') {
            throw 'MCP server advertised unexpected capabilities'
        }

        $initialized = [ordered]@{ jsonrpc = '2.0'; method = 'notifications/initialized'; params = [ordered]@{} } | ConvertTo-Json -Depth 10 -Compress
        $process.StandardInput.WriteLine($initialized)
        $process.StandardInput.Flush()

        $tools = Send-MCPRequest -Process $process -Request ([ordered]@{ jsonrpc = '2.0'; id = 2; method = 'tools/list'; params = [ordered]@{} })
        $actualTools = @($tools.result.tools | ForEach-Object { $_.name } | Sort-Object)
        $expectedTools = @('catalog_inspect', 'context_read', 'context_search', 'context_sources', 'context_status', 'engkit_doctor', 'plan_inspect', 'profiles_recommend', 'skills_list', 'skills_validate', 'workspace_inspect', 'workspace_tree')
        if (($actualTools -join ',') -ne ($expectedTools -join ',')) {
            throw 'MCP tool inventory did not match the fixed twelve-tool contract'
        }

        $status = Send-MCPRequest -Process $process -MaximumBytes 16384 -Request ([ordered]@{ jsonrpc = '2.0'; id = 3; method = 'tools/call'; params = [ordered]@{ name = 'context_status'; arguments = [ordered]@{} } })
        Assert-MCPResult $status
        Assert-Equal $status.result.structuredContent.projection_fingerprint $CLIStatus.projection_fingerprint 'MCP status projection'
        if ($status.result.content[0].text.Contains($CLIStatus.projection_fingerprint)) {
            throw 'MCP compact status summary duplicated structured data'
        }

        $sources = Send-MCPRequest -Process $process -MaximumBytes 65536 -Request ([ordered]@{ jsonrpc = '2.0'; id = 4; method = 'tools/call'; params = [ordered]@{ name = 'context_sources'; arguments = [ordered]@{ limit = 200 } } })
        Assert-MCPResult $sources
        foreach ($item in @($sources.result.structuredContent.items)) {
            Assert-SafeIndexedPath $item.path
        }

        $search = Send-MCPRequest -Process $process -MaximumBytes 32768 -Request ([ordered]@{ jsonrpc = '2.0'; id = 5; method = 'tools/call'; params = [ordered]@{ name = 'context_search'; arguments = [ordered]@{ query = 'QualificationNeedle'; limit = 5; max_content_bytes = 4096 } } })
        Assert-MCPResult $search
        Assert-Equal @($search.result.structuredContent.results).Count @($CLISearch.results).Count 'MCP search result count'
        Assert-Equal $search.result.structuredContent.results[0].chunk_id $CLISearch.results[0].chunk_id 'MCP search chunk identity'
        foreach ($item in @($search.result.structuredContent.results)) {
            Assert-SafeIndexedPath $item.path
            if ($search.result.content[0].text.Contains($item.snippet)) {
                throw 'MCP compact search summary duplicated snippet content'
            }
        }

        $readArguments = [ordered]@{ path = $CLISearch.results[0].path; chunk = $CLISearch.results[0].chunk_id; max_bytes = 2048 }
        $read = Send-MCPRequest -Process $process -MaximumBytes 40960 -Request ([ordered]@{ jsonrpc = '2.0'; id = 6; method = 'tools/call'; params = [ordered]@{ name = 'context_read'; arguments = $readArguments } })
        Assert-MCPResult $read
        Assert-Equal $read.result.structuredContent.content_sha256 $CLIRead.content_sha256 'MCP read content digest'
        if ($read.result.content[0].text.Contains($read.result.structuredContent.content)) {
            throw 'MCP compact read summary duplicated excerpt content'
        }

        $tree = Send-MCPRequest -Process $process -MaximumBytes 65536 -Request ([ordered]@{ jsonrpc = '2.0'; id = 7; method = 'tools/call'; params = [ordered]@{ name = 'workspace_tree'; arguments = [ordered]@{ max_depth = 8; limit = 500 } } })
        Assert-MCPResult $tree
        foreach ($entry in @($tree.result.structuredContent.entries)) {
            Assert-SafeIndexedPath $entry.path
        }

        $textSearch = Send-MCPRequest -Process $process -MaximumBytes 32768 -Request ([ordered]@{ jsonrpc = '2.0'; id = 8; method = 'tools/call'; params = [ordered]@{ name = 'context_search'; arguments = [ordered]@{ query = 'QualificationNeedle'; response_mode = 'text' } } })
        if ($null -ne $textSearch.PSObject.Properties['error'] -or ($null -ne $textSearch.result.PSObject.Properties['isError'] -and $textSearch.result.isError -eq $true)) {
            throw 'MCP text compatibility call failed'
        }
        if ($null -ne $textSearch.result.PSObject.Properties['structuredContent']) {
            throw 'MCP text compatibility result included structured content'
        }
        if (@($textSearch.result.content).Count -ne 1) {
            throw 'MCP text compatibility result did not contain exactly one block'
        }
        $textPayload = $textSearch.result.content[0].text | ConvertFrom-Json -Depth 100
        Assert-Equal $textPayload.schema_version 'engkit.context-index/v1alpha1' 'MCP text compatibility schema'

        $invalid = Send-MCPRequest -Process $process -Request ([ordered]@{ jsonrpc = '2.0'; id = 9; method = 'tools/call'; params = [ordered]@{ name = 'context_status'; arguments = [ordered]@{ unknown = $true } } })
        if ($null -eq $invalid.PSObject.Properties['error']) {
            throw 'MCP accepted an unknown context argument'
        }
        $escape = Send-MCPRequest -Process $process -Request ([ordered]@{ jsonrpc = '2.0'; id = 10; method = 'tools/call'; params = [ordered]@{ name = 'context_read'; arguments = [ordered]@{ path = '../escape'; start_line = 1; end_line = 1 } } })
        if ($null -eq $escape.PSObject.Properties['error']) {
            throw 'MCP accepted an escaping context path'
        }

        return [ordered]@{
            tool_count = $actualTools.Count
            tree_entry_count = @($tree.result.structuredContent.entries).Count
            compact_search_bytes = $search.result.structuredContent.applied_content_bytes
            compact_read_bytes = $read.result.structuredContent.applied_bytes
        }
    } finally {
        try {
            $process.StandardInput.Close()
        } catch {
            # The process may already have closed its standard input stream.
        }
        if (-not $process.WaitForExit(15000)) {
            $process.Kill($true)
            $process.WaitForExit()
        }
        $process.Dispose()
    }
}

try {
    $versionResult = Invoke-EngkitJSON -Arguments @('version', '--json')
    Assert-Equal $versionResult.version $Version 'Release version'

    $workspace = Join-Path $resolvedOutput 'synthetic-context-workspace'
    $engkitDirectory = Join-Path $workspace '.engkit'
    $sourceDirectory = Join-Path $workspace 'src'
    $privateDirectory = Join-Path $workspace 'private'
    New-Item -ItemType Directory -Path $engkitDirectory, $sourceDirectory, $privateDirectory -Force | Out-Null
    Write-UTF8NoBOM -Path (Join-Path $workspace 'go.mod') -Contents "module example.invalid/v040-qualification`n`ngo 1.27.3`n"
    Write-UTF8NoBOM -Path (Join-Path $workspace 'README.md') -Contents "# Synthetic qualification workspace`n"
    Write-UTF8NoBOM -Path (Join-Path $sourceDirectory 'service.go') -Contents "package synthetic`n`nfunc QualificationNeedle() string { return `"bounded`" }`n"
    Write-UTF8NoBOM -Path (Join-Path $privateDirectory 'hidden.go') -Contents "package private`n`nfunc QualificationNeedle() string { return `"excluded`" }`n"
    Write-UTF8NoBOM -Path (Join-Path $workspace '.env') -Contents "QUALIFICATION_SECRET=must-not-index`n"
    Write-UTF8NoBOM -Path (Join-Path $engkitDirectory 'config.yaml') -Contents "schema_version: engkit.config/v1alpha1`ncontext:`n  source_roots: [.]`n  exclude_paths: [private]`n"

    $rebuild = Invoke-EngkitJSON -Arguments @('context', 'rebuild', '--root', $workspace)
    Assert-Equal $rebuild.schema_version 'engkit.context-index/v1alpha1' 'Context rebuild schema'
    if ($rebuild.source_count -lt 1 -or $rebuild.chunk_count -lt 1 -or $rebuild.excluded_count -lt 2) {
        throw 'Context rebuild did not report the expected bounded inventory'
    }

    $status = Invoke-EngkitJSON -Arguments @('context', 'status', '--root', $workspace)
    Assert-Equal $status.schema_version 'engkit.context-status/v1alpha1' 'Context status schema'
    if (-not $status.fresh -or -not $status.supported -or $status.projection_fingerprint -ne $rebuild.source_fingerprint) {
        throw 'Context status did not report the fresh rebuilt projection'
    }

    $sources = Invoke-EngkitJSON -Arguments @('context', 'sources', '--root', $workspace, '--limit', '200')
    Assert-Equal $sources.schema_version 'engkit.context-sources/v1alpha1' 'Context sources schema'
    foreach ($item in @($sources.items)) {
        Assert-SafeIndexedPath $item.path
    }

    $search = Invoke-EngkitJSON -Arguments @('context', 'search', '--root', $workspace, '--limit', '5', '--max-content-bytes', '4096', 'QualificationNeedle')
    Assert-Equal $search.schema_version 'engkit.context-index/v1alpha1' 'Context search schema'
    if (@($search.results).Count -ne 1 -or $search.applied_content_bytes -gt 4096) {
        throw 'Context search did not return the single bounded eligible result'
    }
    Assert-SafeIndexedPath $search.results[0].path

    $read = Invoke-EngkitJSON -Arguments @('context', 'read', '--root', $workspace, '--path', $search.results[0].path, '--chunk', $search.results[0].chunk_id, '--max-bytes', '2048')
    Assert-Equal $read.schema_version 'engkit.context-read/v1alpha1' 'Context read schema'
    if ($read.applied_bytes -gt 2048 -or $read.content -notmatch 'QualificationNeedle') {
        throw 'Context read did not return the bounded indexed fixture'
    }

    $mcpSummary = Test-MCPContext -Root $workspace -CLIStatus $status -CLISearch $search -CLIRead $read

    Write-UTF8NoBOM -Path (Join-Path $sourceDirectory 'service.go') -Contents "package synthetic`n`nfunc QualificationNeedle() string { return `"changed`" }`n"
    $stale = Invoke-EngkitJSON -Arguments @('context', 'status', '--root', $workspace)
    if ($stale.fresh -or $stale.reason -ne 'stale') {
        throw 'Context status did not detect the modified synthetic source'
    }
    Invoke-ExpectedFailure -Arguments @('context', 'search', '--root', $workspace, 'QualificationNeedle')

    $evidence = [ordered]@{
        schema_version = 'engkit.v040-release-qualification/v1alpha1'
        version = $Version
        binary = [System.IO.Path]::GetFileName($resolvedBinary)
        policy_version = $status.policy_version
        projection_fingerprint = $status.projection_fingerprint
        source_count = $status.source_count
        chunk_count = $status.chunk_count
        excluded_count = $status.excluded_count
        tool_count = $mcpSummary.tool_count
        tree_entry_count = $mcpSummary.tree_entry_count
        compact_search_bytes = $mcpSummary.compact_search_bytes
        compact_read_bytes = $mcpSummary.compact_read_bytes
        scenarios = @(
            'context-v2-rebuild-and-freshness',
            'mandatory-and-user-exclusions',
            'bounded-search-and-indexed-read',
            'cli-and-mcp-result-equivalence',
            'twelve-tool-stdio-inventory',
            'indexed-workspace-tree',
            'compact-result-non-duplication',
            'text-compatibility-single-representation',
            'strict-input-and-path-refusal',
            'stale-projection-refusal'
        )
    }
    $evidencePath = Join-Path $resolvedOutput 'v040-release-qualification.json'
    Write-UTF8NoBOM -Path $evidencePath -Contents (($evidence | ConvertTo-Json -Depth 10) + "`n")
    Write-Host "v0.4.0 native qualification passed for $resolvedBinary"
    Write-Host "Evidence written to $evidencePath"
} catch {
    Write-Error $_
    throw
}
