[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$BinaryPath,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v0\.5\.0(?:-rc\.[1-9][0-9]*)?$')]
    [string]$Version,

    [string]$OutputDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$resolvedBinary = (Resolve-Path -LiteralPath $BinaryPath).Path
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("engkit-v050-smoke-" + [guid]::NewGuid().ToString('N'))
}
$resolvedOutput = [System.IO.Path]::GetFullPath($OutputDirectory)
if (Test-Path -LiteralPath $resolvedOutput) {
    if (@(Get-ChildItem -LiteralPath $resolvedOutput -Force).Count -ne 0) {
        throw "Smoke output directory is not empty: $resolvedOutput"
    }
} else {
    New-Item -ItemType Directory -Path $resolvedOutput | Out-Null
}

$strictUTF8 = [System.Text.UTF8Encoding]::new($false, $true)
$summaryWorkspace = Join-Path $resolvedOutput ("summary-disclosure-marker-" + [guid]::NewGuid().ToString('N'))
$absentWorkspace = Join-Path $resolvedOutput ("absent-disclosure-marker-" + [guid]::NewGuid().ToString('N'))
$v040Output = Join-Path $resolvedOutput 'v040-regression'
$v040Workspace = Join-Path $v040Output 'synthetic-context-workspace'

function Assert-True {
    param([bool]$Condition, [string]$Description)
    if (-not $Condition) {
        throw $Description
    }
}

function Get-SHA256 {
    param([Parameter(Mandatory = $true)][byte[]]$Bytes)
    $algorithm = [System.Security.Cryptography.SHA256]::Create()
    try {
        return ([Convert]::ToHexString($algorithm.ComputeHash($Bytes))).ToLowerInvariant()
    } finally {
        $algorithm.Dispose()
    }
}

function Get-LineMetrics {
    param([Parameter(Mandatory = $true)][byte[]]$Bytes)

    $lineCount = 0
    $currentBytes = 0
    $maximumBytes = 0
    foreach ($value in $Bytes) {
        if ($value -eq 10) {
            $lineCount++
            if ($currentBytes -gt $maximumBytes) {
                $maximumBytes = $currentBytes
            }
            $currentBytes = 0
        } else {
            $currentBytes++
        }
    }
    if ($currentBytes -gt 0) {
        $lineCount++
        if ($currentBytes -gt $maximumBytes) {
            $maximumBytes = $currentBytes
        }
    }
    return [pscustomobject]@{
        line_count = $lineCount
        maximum_line_bytes = $maximumBytes
    }
}

function ConvertFrom-StrictUTF8 {
    param([Parameter(Mandatory = $true)][byte[]]$Bytes, [string]$Description)
    try {
        $text = $strictUTF8.GetString($Bytes)
    } catch {
        throw "$Description was not valid UTF-8"
    }
    if ($text.Length -gt 0 -and $text[0] -eq [char]0xFEFF) {
        throw "$Description contained an unexpected UTF-8 BOM"
    }
    return $text
}

function Invoke-EngkitCapture {
    param(
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [int[]]$AllowedExitCodes = @(0)
    )

    $stdoutPath = Join-Path $resolvedOutput ("stdout-" + [guid]::NewGuid().ToString('N') + '.bin')
    $stderrPath = Join-Path $resolvedOutput ("stderr-" + [guid]::NewGuid().ToString('N') + '.bin')
    try {
        & $resolvedBinary @Arguments 1> $stdoutPath 2> $stderrPath
        $exitCode = $LASTEXITCODE
        if ($AllowedExitCodes -notcontains $exitCode) {
            throw "engkit command exited with unexpected status $exitCode"
        }
        $stdoutBytes = [System.IO.File]::ReadAllBytes($stdoutPath)
        $stderrBytes = [System.IO.File]::ReadAllBytes($stderrPath)
        if ($stderrBytes.Length -ne 0) {
            throw 'engkit command wrote unexpected standard error output'
        }
        $global:LASTEXITCODE = 0
        return [pscustomobject]@{
            exit_code = $exitCode
            bytes = $stdoutBytes
            text = ConvertFrom-StrictUTF8 -Bytes $stdoutBytes -Description 'engkit standard output'
        }
    } finally {
        Remove-Item -LiteralPath $stdoutPath, $stderrPath -Force -ErrorAction SilentlyContinue
    }
}

function Get-OutputEvidence {
    param([Parameter(Mandatory = $true)]$Capture)
    $metrics = Get-LineMetrics -Bytes $Capture.bytes
    return [ordered]@{
        sha256 = Get-SHA256 -Bytes $Capture.bytes
        bytes = $Capture.bytes.Length
        line_count = $metrics.line_count
        maximum_line_bytes = $metrics.maximum_line_bytes
    }
}

function Assert-SummaryDisclosureBoundary {
    param(
        [Parameter(Mandatory = $true)]$Capture,
        [Parameter(Mandatory = $true)][string[]]$ForbiddenMarkers
    )

    Assert-True ($Capture.bytes.Length -gt 0) 'Doctor summary was empty'
    Assert-True (-not $Capture.bytes.Contains([byte]13)) 'Doctor summary contained a carriage return'
    Assert-True (-not $Capture.bytes.Contains([byte]9)) 'Doctor summary contained a tab'
    $metrics = Get-LineMetrics -Bytes $Capture.bytes
    Assert-True ($metrics.maximum_line_bytes -le 1040) 'Doctor summary exceeded the 1,040-byte line ceiling'
    Assert-True (-not $Capture.text.TrimStart().StartsWith('{')) 'Doctor summary exposed a JSON object'
    Assert-True (-not $Capture.text.Contains('"evidence"')) 'Doctor summary exposed an evidence field'
    foreach ($marker in $ForbiddenMarkers) {
        Assert-True (-not $Capture.text.Contains($marker)) 'Doctor summary exposed a synthetic path marker'
    }
}

try {
    $v040Script = Join-Path $PSScriptRoot 'test-v040-release-binary.ps1'
    & $v040Script -BinaryPath $resolvedBinary -Version $Version -OutputDirectory $v040Output
    $v040Record = Join-Path $v040Output 'v040-release-qualification.json'
    if (-not (Test-Path -LiteralPath $v040Record -PathType Leaf)) {
        throw 'The v0.4.0 regression harness did not produce its qualification record'
    }

    $jsonCapture = Invoke-EngkitCapture -Arguments @('doctor', '--scope', 'core')
    $jsonResult = $jsonCapture.text | ConvertFrom-Json -Depth 100
    Assert-True ($jsonResult.schema_version -ceq 'engkit.doctor-result/v1alpha1') 'Doctor JSON schema changed'
    Assert-True ($jsonResult.summary.healthy -eq $true) 'Core doctor JSON was not healthy'
    Assert-True ($jsonResult.summary.required_errors -eq 0) 'Core doctor JSON reported a required error'

    $longSummary = Invoke-EngkitCapture -Arguments @('doctor', '--scope', 'core', '--summary')
    $shortSummary = Invoke-EngkitCapture -Arguments @('doctor', '--scope', 'core', '-s')
    Assert-True ([System.Linq.Enumerable]::SequenceEqual[byte]($longSummary.bytes, $shortSummary.bytes)) 'Doctor summary aliases were not byte-identical'
    Assert-True ($longSummary.text.Contains('[OK]')) 'Healthy doctor summary omitted [OK]'
    Assert-True ($longSummary.text.Contains('Overall: OK')) 'Healthy doctor summary omitted Overall: OK'
    $expectedCounts = "Passed: $($jsonResult.summary.pass)  Warnings: $($jsonResult.summary.warning)  Errors: $($jsonResult.summary.error)  Unavailable: $($jsonResult.summary.unavailable)"
    Assert-True ($longSummary.text.Contains($expectedCounts)) 'Doctor summary counts did not match canonical JSON'

    $shortHelp = Invoke-EngkitCapture -Arguments @('doctor', '-h')
    $longHelp = Invoke-EngkitCapture -Arguments @('doctor', '--help')
    Assert-True ([System.Linq.Enumerable]::SequenceEqual[byte]($shortHelp.bytes, $longHelp.bytes)) 'Doctor help aliases were not byte-identical'
    Assert-True ($shortHelp.text.Contains('Usage:')) 'Doctor help omitted usage'
    Assert-True (-not $shortHelp.text.Contains('Engineering Toolkit doctor')) 'Doctor help executed the summary renderer'
    Assert-True (-not $shortHelp.text.Contains('[OK]')) 'Doctor help contained a diagnostic status'

    New-Item -ItemType Directory -Path $summaryWorkspace | Out-Null
    $optionalSummary = Invoke-EngkitCapture -Arguments @('doctor', '--root', $summaryWorkspace, '--scope', 'workspace', '--summary')
    Assert-True ($optionalSummary.exit_code -eq 0) 'Optional-state doctor summary did not exit 0'
    Assert-True ($optionalSummary.text.Contains('[WARN]')) 'Optional-state doctor summary omitted [WARN]'
    Assert-True ($optionalSummary.text.Contains('[N/A]')) 'Optional-state doctor summary omitted [N/A]'
    Assert-True ($optionalSummary.text.Contains('Overall: OK')) 'Optional-state doctor summary was not healthy'

    $requiredErrorSummary = Invoke-EngkitCapture -Arguments @('doctor', '--root', $absentWorkspace, '--scope', 'workspace', '--summary') -AllowedExitCodes @(1)
    Assert-True ($requiredErrorSummary.exit_code -eq 1) 'Required-error doctor summary did not exit 1'
    Assert-True ($requiredErrorSummary.text.Contains('[ERROR]')) 'Required-error doctor summary omitted [ERROR]'
    Assert-True ($requiredErrorSummary.text.Contains('Overall: NEEDS ATTENTION')) 'Required-error doctor summary omitted the unhealthy label'

    $forbiddenMarkers = @(
        [System.IO.Path]::GetFileName($summaryWorkspace),
        [System.IO.Path]::GetFileName($absentWorkspace),
        $summaryWorkspace,
        $absentWorkspace,
        $summaryWorkspace.Replace('\', '/'),
        $absentWorkspace.Replace('\', '/')
    )
    foreach ($capture in @($longSummary, $shortSummary, $optionalSummary, $requiredErrorSummary)) {
        Assert-SummaryDisclosureBoundary -Capture $capture -ForbiddenMarkers $forbiddenMarkers
    }

    $v040Bytes = [System.IO.File]::ReadAllBytes($v040Record)
    $relativeV040Record = [System.IO.Path]::GetRelativePath($resolvedOutput, $v040Record).Replace('\', '/')
    $evidence = [ordered]@{
        schema_version = 'engkit.v050-release-qualification/v1alpha1'
        version = $Version
        binary = [System.IO.Path]::GetFileName($resolvedBinary)
        doctor_schema = $jsonResult.schema_version
        doctor_healthy = [bool]$jsonResult.summary.healthy
        healthy_exit_code = $longSummary.exit_code
        optional_exit_code = $optionalSummary.exit_code
        required_error_exit_code = $requiredErrorSummary.exit_code
        outputs = [ordered]@{
            canonical_json = Get-OutputEvidence -Capture $jsonCapture
            long_summary = Get-OutputEvidence -Capture $longSummary
            short_summary = Get-OutputEvidence -Capture $shortSummary
            short_help = Get-OutputEvidence -Capture $shortHelp
            long_help = Get-OutputEvidence -Capture $longHelp
        }
        regression = [ordered]@{
            relative_path = $relativeV040Record
            sha256 = Get-SHA256 -Bytes $v040Bytes
        }
        scenarios = @(
            [ordered]@{ id = 'doctor-json-compatibility'; outcome = 'pass' },
            [ordered]@{ id = 'doctor-summary-alias-equivalence'; outcome = 'pass' },
            [ordered]@{ id = 'doctor-help-isolation'; outcome = 'pass' },
            [ordered]@{ id = 'doctor-optional-state-health'; outcome = 'pass' },
            [ordered]@{ id = 'doctor-required-error-health'; outcome = 'pass' },
            [ordered]@{ id = 'doctor-summary-disclosure-boundary'; outcome = 'pass' }
        )
    }

    $evidencePath = Join-Path $resolvedOutput 'v050-release-qualification.json'
    [System.IO.File]::WriteAllText(
        $evidencePath,
        (($evidence | ConvertTo-Json -Depth 10) + "`n"),
        [System.Text.UTF8Encoding]::new($false)
    )

    Remove-Item -LiteralPath $summaryWorkspace -Recurse -Force
    Remove-Item -LiteralPath $v040Workspace -Recurse -Force
    $summaryWorkspace = $null
    $v040Workspace = $null

    $retainedFiles = @(Get-ChildItem -LiteralPath $resolvedOutput -Recurse -File)
    if ($retainedFiles.Count -ne 2 -or
        -not (Test-Path -LiteralPath $evidencePath -PathType Leaf) -or
        -not (Test-Path -LiteralPath $v040Record -PathType Leaf)) {
        throw 'Qualification output retained files other than the two bounded evidence records'
    }

    Write-Host "v0.5.0 native qualification passed for $resolvedBinary"
    Write-Host "Evidence written to $evidencePath"
} catch {
    Write-Error $_
    throw
} finally {
    if ($null -ne $summaryWorkspace -and (Test-Path -LiteralPath $summaryWorkspace -PathType Container)) {
        Remove-Item -LiteralPath $summaryWorkspace -Recurse -Force
    }
    if ($null -ne $v040Workspace -and (Test-Path -LiteralPath $v040Workspace -PathType Container)) {
        Remove-Item -LiteralPath $v040Workspace -Recurse -Force
    }
}
