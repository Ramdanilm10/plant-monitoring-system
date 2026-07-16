param(
    [string]$EnvFile = "",

    [ValidateRange(1, 3650)]
    [int]$BackupRetentionDays = 30,

    [ValidateRange(30, 3650)]
    [int]$AuditRetentionDays = 90
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$BackupScript = Join-Path `
    $PSScriptRoot `
    "backup-database.ps1"

$CleanupScript = Join-Path `
    $PSScriptRoot `
    "cleanup-audit-logs.ps1"

$BackupParameters = @{
    RetentionDays = $BackupRetentionDays
    KeepMinimum   = 5
}

$CleanupParameters = @{
    AuditRetentionDays = $AuditRetentionDays
    Apply              = $true
}

if (
    -not [string]::IsNullOrWhiteSpace(
        $EnvFile
    )
) {
    $BackupParameters["EnvFile"] = $EnvFile
    $CleanupParameters["EnvFile"] = $EnvFile
}

Write-Host ""
Write-Host "========================================"
Write-Host " Plant Monitoring Database Maintenance"
Write-Host "========================================"
Write-Host ""

& $BackupScript @BackupParameters

& $CleanupScript @CleanupParameters

Write-Host ""
Write-Host "Database maintenance selesai." `
    -ForegroundColor Green