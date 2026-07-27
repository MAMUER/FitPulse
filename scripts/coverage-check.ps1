param()

$coverageFile = Join-Path (Get-Location) "coverage.out"
if (-not (Test-Path $coverageFile)) {
    Write-Error "coverage.out not found. Run 'go test -coverprofile=coverage.out ./...' first."
    exit 1
}

$totalStatements = 0
$totalCovered = 0

Get-Content $coverageFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -match '^mode:') { return }
    if ($line -match '^\s*$') { return }

    if ($line -match '/api/gen/') { return }
    if ($line -match '/mocks/') { return }
    if ($line -match '/internal/grpc/') { return }
    if ($line -match '/internal/db/') { return }
    if ($line -match '/internal/queue/') { return }
    if ($line -match '/internal/middleware/') { return }
    if ($line -match '/internal/crypto/') { return }
    if ($line -match '/internal/totp/') { return }
    if ($line -match '/internal/telemetry/') { return }
    if ($line -match '/internal/testcontainers/') { return }

    $lastSpace = $line.LastIndexOf(' ')
    if ($lastSpace -lt 0) { return }
    $secondLastSpace = $line.LastIndexOf(' ', $lastSpace - 1)
    if ($secondLastSpace -lt 0) { return }

    $numStmts = [long]$line.Substring($secondLastSpace + 1, $lastSpace - $secondLastSpace - 1)
    $count    = [long]$line.Substring($lastSpace + 1)

    $totalStatements += $numStmts
    if ($count -gt 0) {
        $totalCovered += $numStmts
    }
}

if ($totalStatements -eq 0) {
    Write-Error "No business-logic statements found in coverage.out"
    exit 1
}

$coverage = ($totalCovered / $totalStatements) * 100
$coverageRounded = [math]::Round($coverage, 2)

Write-Host "Coverage: $coverageRounded% of statements (business logic only, excluding generated/mocks/infrastructure)"

if ($coverage -lt 75) {
    Write-Error "Coverage threshold (>= 75%) not met. Current: $coverageRounded%"
    exit 1
}

exit 0
