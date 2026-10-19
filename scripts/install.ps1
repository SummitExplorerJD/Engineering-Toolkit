[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-rc\.[1-9][0-9]*)?$')]
    [string]$Version,

    [string]$InstallDir,

    [switch]$Upgrade,

    [switch]$AddToPath,

    [string]$AssetDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$isWindowsPlatform = $env:OS -eq 'Windows_NT'
if (-not $isWindowsPlatform) {
    throw 'install.ps1 supports Windows only; use install.sh on Linux or macOS'
}

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        throw 'LOCALAPPDATA is unavailable; specify -InstallDir explicitly'
    }
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\engkit'
}
$InstallDir = [System.IO.Path]::GetFullPath($InstallDir)
if ($AddToPath -and $InstallDir -match '[;\r\n]') {
    throw 'InstallDir cannot contain a semicolon or newline when -AddToPath is used'
}

$architectureName = if (-not [string]::IsNullOrWhiteSpace($env:PROCESSOR_ARCHITEW6432)) {
    $env:PROCESSOR_ARCHITEW6432
} else {
    $env:PROCESSOR_ARCHITECTURE
}
if ([string]::IsNullOrWhiteSpace($architectureName)) {
    throw 'Windows processor architecture is unavailable'
}
$architecture = switch ($architectureName.ToUpperInvariant()) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    default { throw "Unsupported Windows architecture: $architectureName" }
}

$filename = "engkit-$Version-windows-$architecture.exe"
$target = Join-Path $InstallDir 'engkit.exe'
$targetExists = Test-Path -LiteralPath $target -PathType Leaf
if ($targetExists -and -not $Upgrade) {
    throw "Engineering Toolkit is already installed at $target; use -Upgrade to replace it explicitly"
}
if (Test-Path -LiteralPath $target -PathType Container) {
    throw "Installation target is a directory: $target"
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
$transactionDirectory = Join-Path $InstallDir ('.engkit-install-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $transactionDirectory | Out-Null
$stagedBinary = Join-Path $transactionDirectory $filename
$publishBinary = Join-Path $transactionDirectory 'engkit.publish.exe'
$stagedChecksums = Join-Path $transactionDirectory 'SHA256SUMS'
$backup = Join-Path $transactionDirectory 'engkit.previous.exe'
$committed = $false

try {
    if ([string]::IsNullOrWhiteSpace($AssetDirectory)) {
        if ($PSVersionTable.PSEdition -eq 'Desktop') {
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        }
        $baseUrl = "https://github.com/SummitExplorerJD/Engineering-Toolkit/releases/download/$Version"
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/$filename" -OutFile $stagedBinary
        Invoke-WebRequest -UseBasicParsing -Uri "$baseUrl/SHA256SUMS" -OutFile $stagedChecksums
    } else {
        $resolvedAssets = (Resolve-Path -LiteralPath $AssetDirectory).Path
        Copy-Item -LiteralPath (Join-Path $resolvedAssets $filename) -Destination $stagedBinary
        Copy-Item -LiteralPath (Join-Path $resolvedAssets 'SHA256SUMS') -Destination $stagedChecksums
    }

    $matchingLines = @(Get-Content -LiteralPath $stagedChecksums | Where-Object {
        $_ -match '^([0-9a-fA-F]{64})  (.+)$' -and $Matches[2] -ceq $filename
    })
    if ($matchingLines.Count -ne 1) {
        throw "SHA256SUMS must contain exactly one entry for $filename"
    }
    $null = $matchingLines[0] -match '^([0-9a-fA-F]{64})  (.+)$'
    $expectedHash = $Matches[1].ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $stagedBinary -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -cne $expectedHash) {
        throw "SHA-256 mismatch for $filename"
    }

    $reportedVersion = (& $stagedBinary version --json | ConvertFrom-Json).version
    if ($LASTEXITCODE -ne 0 -or $reportedVersion -cne $Version) {
        throw "Downloaded executable reported '$reportedVersion', want '$Version'"
    }
    Copy-Item -LiteralPath $stagedBinary -Destination $publishBinary
    $publishHash = (Get-FileHash -LiteralPath $publishBinary -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($publishHash -cne $expectedHash) {
        throw "Staged publication copy changed unexpectedly for $filename"
    }

    if ($targetExists) {
        Move-Item -LiteralPath $target -Destination $backup
    }
    try {
        if ($AssetDirectory -and $env:ENGKIT_INSTALL_TEST_FAIL_AFTER_BACKUP -eq '1') {
            throw 'Injected installer failure after backup'
        }
        Move-Item -LiteralPath $publishBinary -Destination $target
        $committed = $true
    } catch {
        if ($targetExists) {
            if (Test-Path -LiteralPath $target -PathType Leaf) {
                Remove-Item -LiteralPath $target -Force
            }
            if (Test-Path -LiteralPath $backup -PathType Leaf) {
                Move-Item -LiteralPath $backup -Destination $target
            }
        }
        throw
    }

    if ($AddToPath) {
        $testPathFile = if ($AssetDirectory) { $env:ENGKIT_INSTALL_TEST_USER_PATH_FILE } else { $null }
        if (-not [string]::IsNullOrWhiteSpace($testPathFile)) {
            $userPath = if (Test-Path -LiteralPath $testPathFile) {
                Get-Content -Raw -LiteralPath $testPathFile
            } else {
                ''
            }
        } else {
            $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        }
        $entries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
        $containsInstallDir = $false
        foreach ($entry in $entries) {
            if ($entry.TrimEnd('\') -ieq $InstallDir.TrimEnd('\')) {
                $containsInstallDir = $true
                break
            }
        }
        if (-not $containsInstallDir) {
            $newUserPath = (@($entries) + $InstallDir) -join ';'
            if (-not [string]::IsNullOrWhiteSpace($testPathFile)) {
                Set-Content -LiteralPath $testPathFile -Value $newUserPath -Encoding utf8NoBOM -NoNewline
            } else {
                [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
            }
        }
    }
} finally {
    if ($committed -and (Test-Path -LiteralPath $backup -PathType Leaf)) {
        Remove-Item -LiteralPath $backup -Force
    }
    if (Test-Path -LiteralPath $transactionDirectory -PathType Container) {
        if (-not $committed -and (Test-Path -LiteralPath $backup -PathType Leaf)) {
            Write-Warning "Automatic rollback could not complete; the previous executable is preserved at $backup"
        } else {
            Remove-Item -LiteralPath $transactionDirectory -Recurse -Force
        }
    }
}

Write-Host "Engineering Toolkit $Version installed at $target"
if (-not $AddToPath) {
    Write-Host "The installer did not modify PATH. Add '$InstallDir' explicitly if needed."
}

