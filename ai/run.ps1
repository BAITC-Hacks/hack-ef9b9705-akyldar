[CmdletBinding()]
param(
    [switch]$ForceFallback,
    [ValidateNotNullOrEmpty()]
    [string]$Address = '127.0.0.1:8080'
)

$ErrorActionPreference = 'Stop'
$allowedNames = @(
    'OPENAI_API_KEY', 'OPENAI_MODEL', 'AI_TIMEOUT_SECONDS',
    'AI_MAX_ATTEMPTS', 'AI_FORCE_FALLBACK'
)
$settings = @{}
$previousEnvironment = @{}
$locationPushed = $false
$exitCode = 1

try {
    $envPath = Join-Path $PSScriptRoot '.env'
    if (Test-Path -LiteralPath $envPath -PathType Leaf) {
        $lineNumber = 0
        foreach ($line in Get-Content -LiteralPath $envPath -Encoding UTF8) {
            $lineNumber++
            $trimmed = $line.Trim()
            if ($trimmed.Length -eq 0 -or $trimmed.StartsWith('#')) {
                continue
            }

            $entry = [regex]::Match($line, '^\s*([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$')
            if (-not $entry.Success) {
                throw [System.IO.InvalidDataException]::new("Malformed .env entry on line $lineNumber.")
            }
            $name = $entry.Groups[1].Value.ToUpperInvariant()
            $value = $entry.Groups[2].Value.Trim()
            if ($allowedNames -notcontains $name) {
                throw [System.IO.InvalidDataException]::new("Unknown .env entry on line $lineNumber.")
            }
            if ($settings.ContainsKey($name)) {
                throw [System.IO.InvalidDataException]::new("Duplicate .env entry on line $lineNumber.")
            }

            # Quotes are delimiters only; never expand or execute their contents.
            if ($value.Length -gt 0) {
                $first = $value[0]
                $last = $value[$value.Length - 1]
                $startsQuoted = $first -eq '"' -or $first -eq "'"
                $endsQuoted = $last -eq '"' -or $last -eq "'"
                if ($startsQuoted -or $endsQuoted) {
                    if (-not $startsQuoted -or $value.Length -lt 2 -or $last -ne $first) {
                        throw [System.IO.InvalidDataException]::new("Unmatched .env quotes on line $lineNumber.")
                    }
                    $value = $value.Substring(1, $value.Length - 2)
                }
                if ($value.IndexOf([char]0) -ge 0) {
                    throw [System.IO.InvalidDataException]::new("Malformed .env value on line $lineNumber.")
                }
            }
            $settings[$name] = $value
        }
    }

    if ($ForceFallback) {
        $settings['AI_FORCE_FALLBACK'] = 'true'
    }

    $goCommand = Get-Command go -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($null -ne $goCommand) {
        $goPath = $goCommand.Path
    } else {
        $goPath = Join-Path $PSScriptRoot '.local\tools\go\bin\go.exe'
        if (-not (Test-Path -LiteralPath $goPath -PathType Leaf)) {
            throw [System.IO.InvalidDataException]::new('Go was not found on PATH or in ai/.local/tools/go/bin/go.exe.')
        }
    }

    $settings['GOCACHE'] = Join-Path $PSScriptRoot '.local\gocache'
    $settings['GOMODCACHE'] = Join-Path $PSScriptRoot '.local\gomodcache'
    $settings['GOTOOLCHAIN'] = 'local'
    foreach ($cacheName in @('GOCACHE', 'GOMODCACHE')) {
        New-Item -ItemType Directory -Path $settings[$cacheName] -Force | Out-Null
    }

    foreach ($name in $settings.Keys) {
        $previousEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
        [Environment]::SetEnvironmentVariable($name, $settings[$name], 'Process')
    }

    Push-Location -LiteralPath $PSScriptRoot
    $locationPushed = $true
    & $goPath run ./cmd/demo -addr $Address
    $exitCode = $LASTEXITCODE
} catch [System.IO.InvalidDataException] {
    Write-Error -Message $_.Exception.Message -ErrorAction Continue
} catch {
    # Do not include exception details that might contain configuration values.
    Write-Error -Message 'Unable to launch the AI demo. Check Go, .env access, and local cache permissions.' -ErrorAction Continue
} finally {
    foreach ($name in $previousEnvironment.Keys) {
        [Environment]::SetEnvironmentVariable($name, $previousEnvironment[$name], 'Process')
    }
    if ($locationPushed) {
        Pop-Location
    }
}

exit $exitCode
