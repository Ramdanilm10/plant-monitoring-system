param(
    [string]$EnvFile = "",

    [ValidateRange(30, 3650)]
    [int]$AuditRetentionDays = 90,

    [switch]$Apply
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

. (
    Join-Path `
        $PSScriptRoot `
        "database-common.ps1"
)

$ProjectRoot = Get-PlantMonitoringProjectRoot

if ([string]::IsNullOrWhiteSpace($EnvFile)) {
    $EnvFile = Join-Path `
        $ProjectRoot `
        "backend\.env"
}

$Database = Get-DatabaseConfiguration `
    -EnvFile $EnvFile

$Psql = Resolve-PostgresTool `
    -ToolName "psql"

$OldPassword = [Environment]::GetEnvironmentVariable(
    "PGPASSWORD",
    "Process"
)

$OldSSLMode = [Environment]::GetEnvironmentVariable(
    "PGSSLMODE",
    "Process"
)

try {
    [Environment]::SetEnvironmentVariable(
        "PGPASSWORD",
        $Database.Password,
        "Process"
    )

    [Environment]::SetEnvironmentVariable(
        "PGSSLMODE",
        "require",
        "Process"
    )

    $ConnectionArguments = @(
        "--host=$($Database.Host)",
        "--port=$($Database.Port)",
        "--username=$($Database.User)",
        "--dbname=$($Database.Database)",
        "--set=ON_ERROR_STOP=1"
    )

    $CountQuery = @"
SELECT COUNT(*)
FROM public.audit_logs
WHERE occurred_at < NOW() - INTERVAL '$AuditRetentionDays days';
"@

    $CountResult = & $Psql `
        @ConnectionArguments `
        "--tuples-only" `
        "--no-align" `
        "--command=$CountQuery"

    if ($LASTEXITCODE -ne 0) {
        throw "Gagal menghitung audit log lama."
    }

    $ExpiredCount = (
        $CountResult |
        Out-String
    ).Trim()

    Write-Host ""
    Write-Host "Retensi audit log" `
        -ForegroundColor Cyan

    Write-Host "Batas retensi : $AuditRetentionDays hari"
    Write-Host "Data kedaluwarsa: $ExpiredCount baris"

    if (-not $Apply) {
        Write-Host ""
        Write-Host "DRY RUN — tidak ada data yang dihapus." `
            -ForegroundColor Yellow

        Write-Host "Tambahkan -Apply untuk menjalankan penghapusan."

        return
    }

    $DeleteQuery = @"
DELETE FROM public.audit_logs
WHERE occurred_at < NOW() - INTERVAL '$AuditRetentionDays days';
"@

    & $Psql `
        @ConnectionArguments `
        "--command=$DeleteQuery"

    if ($LASTEXITCODE -ne 0) {
        throw "Penghapusan audit log lama gagal."
    }

    & $Psql `
        @ConnectionArguments `
        "--command=VACUUM (ANALYZE) public.audit_logs;"

    if ($LASTEXITCODE -ne 0) {
        throw "VACUUM audit_logs gagal."
    }

    Write-Host ""
    Write-Host "Retensi audit log berhasil dijalankan." `
        -ForegroundColor Green
}
finally {
    [Environment]::SetEnvironmentVariable(
        "PGPASSWORD",
        $OldPassword,
        "Process"
    )

    [Environment]::SetEnvironmentVariable(
        "PGSSLMODE",
        $OldSSLMode,
        "Process"
    )
}