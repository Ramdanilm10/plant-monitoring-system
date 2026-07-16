param(
    [Parameter(Mandatory = $true)]
    [string]$BackupFile,

    [Parameter(Mandatory = $true)]
    [string]$TargetEnvFile,

    [switch]$ConfirmRestore,

    [switch]$CleanExistingObjects,

    [switch]$AllowLiveTarget
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

. (
    Join-Path `
        $PSScriptRoot `
        "database-common.ps1"
)

if (-not $ConfirmRestore) {
    throw @"
Restore dibatalkan.

Tambahkan parameter -ConfirmRestore setelah memastikan
database tujuan sudah benar.
"@
}

$ProjectRoot = Get-PlantMonitoringProjectRoot

$ResolvedTargetEnv = (
    Resolve-Path $TargetEnvFile
).Path

$LiveEnvironmentFile = [System.IO.Path]::GetFullPath(
    (
        Join-Path `
            $ProjectRoot `
            "backend\.env"
    )
)

$TargetEnvironmentFile = [System.IO.Path]::GetFullPath(
    $ResolvedTargetEnv
)

$TargetsLiveDatabase = [string]::Equals(
    $LiveEnvironmentFile,
    $TargetEnvironmentFile,
    [System.StringComparison]::OrdinalIgnoreCase
)

if (
    $TargetsLiveDatabase -and
    -not $AllowLiveTarget
) {
    throw @"
Target mengarah ke backend\.env yang digunakan aplikasi live.

Gunakan database proyek uji atau tambahkan -AllowLiveTarget
hanya ketika recovery database live memang diperlukan.
"@
}

$ResolvedBackupPath = (
    Resolve-Path $BackupFile
).Path

if (
    Test-Path `
        $ResolvedBackupPath `
        -PathType Container
) {
    $ResolvedBackupPath = Join-Path `
        $ResolvedBackupPath `
        "public.dump"
}

if (
    -not (
        Test-Path `
            $ResolvedBackupPath `
            -PathType Leaf
    )
) {
    throw "File public.dump tidak ditemukan."
}

$Database = Get-DatabaseConfiguration `
    -EnvFile $ResolvedTargetEnv `
    -UseBackupOverrides

$PgRestore = Resolve-PostgresTool `
    -ToolName "pg_restore"

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

    Write-Host ""
    Write-Host "Memulai restore database..." `
        -ForegroundColor Yellow

    Write-Host "Backup : $ResolvedBackupPath"
    Write-Host "Target : $($Database.Database)"

    $RestoreArguments = @(
        "--host=$($Database.Host)",
        "--port=$($Database.Port)",
        "--username=$($Database.User)",
        "--dbname=$($Database.Database)",
        "--no-owner",
        "--no-privileges",
        "--exit-on-error"
    )

    if ($CleanExistingObjects) {
        $RestoreArguments += "--clean"
        $RestoreArguments += "--if-exists"
    }

    $RestoreArguments += $ResolvedBackupPath

    & $PgRestore @RestoreArguments

    if ($LASTEXITCODE -ne 0) {
        throw "pg_restore gagal dengan exit code $LASTEXITCODE."
    }

    $ValidationQuery = @"
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE';
"@

    $ValidationArguments = @(
        "--host=$($Database.Host)",
        "--port=$($Database.Port)",
        "--username=$($Database.User)",
        "--dbname=$($Database.Database)",
        "--set=ON_ERROR_STOP=1",
        "--tuples-only",
        "--no-align",
        "--command=$ValidationQuery"
    )

    $TableCount = (
        & $Psql @ValidationArguments
    ).Trim()

    if ($LASTEXITCODE -ne 0) {
        throw "Validasi database setelah restore gagal."
    }

    Write-Host ""
    Write-Host "Restore berhasil." `
        -ForegroundColor Green

    Write-Host "Jumlah tabel public: $TableCount"
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