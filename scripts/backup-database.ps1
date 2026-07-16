param(
    [string]$EnvFile = "",

    [string]$BackupDirectory = "",

    [ValidateRange(1, 3650)]
    [int]$RetentionDays = 30,

    [ValidateRange(1, 100)]
    [int]$KeepMinimum = 5
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

if (
    [string]::IsNullOrWhiteSpace(
        $BackupDirectory
    )
) {
    $BackupDirectory = Join-Path `
        $env:LOCALAPPDATA `
        "PlantMonitoringSystem\backups"
}

$Database = Get-DatabaseConfiguration `
    -EnvFile $EnvFile `
    -UseBackupOverrides

$PgDump = Resolve-PostgresTool `
    -ToolName "pg_dump"

$PgRestore = Resolve-PostgresTool `
    -ToolName "pg_restore"

New-Item `
    -ItemType Directory `
    -Path $BackupDirectory `
    -Force |
    Out-Null

$Timestamp = Get-Date `
    -Format "yyyyMMdd_HHmmss"

$BackupSetDirectory = Join-Path `
    $BackupDirectory `
    "backup_$Timestamp"

New-Item `
    -ItemType Directory `
    -Path $BackupSetDirectory `
    -Force |
    Out-Null

$DumpFile = Join-Path `
    $BackupSetDirectory `
    "public.dump"

$ObjectListFile = Join-Path `
    $BackupSetDirectory `
    "objects.txt"

$ManifestFile = Join-Path `
    $BackupSetDirectory `
    "manifest.json"

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
    Write-Host "Membuat backup database..." `
        -ForegroundColor Cyan

    Write-Host "Scope  : schema public"
    Write-Host "Output : $BackupSetDirectory"

    $DumpArguments = @(
        "--host=$($Database.Host)",
        "--port=$($Database.Port)",
        "--username=$($Database.User)",
        "--dbname=$($Database.Database)",
        "--format=custom",
        "--compress=9",
        "--no-owner",
        "--no-privileges",
        "--schema=public",
        "--file=$DumpFile"
    )

    & $PgDump @DumpArguments

    if ($LASTEXITCODE -ne 0) {
        throw "pg_dump gagal dengan exit code $LASTEXITCODE."
    }

    & $PgRestore `
        "--list" `
        $DumpFile |
        Set-Content `
            -Path $ObjectListFile `
            -Encoding UTF8

    if ($LASTEXITCODE -ne 0) {
        throw "Verifikasi pg_restore --list gagal."
    }

    $DumpHash = Get-FileHash `
        -Path $DumpFile `
        -Algorithm SHA256

    $DumpInfo = Get-Item $DumpFile

    $Manifest = [ordered]@{
        created_at_utc = (
            Get-Date
        ).ToUniversalTime().ToString("o")

        backup_scope = "public schema"

        format = "PostgreSQL custom dump"

        database_name = $Database.Database

        size_bytes = $DumpInfo.Length

        sha256 = $DumpHash.Hash.ToLowerInvariant()

        dump_file = $DumpInfo.Name

        object_list_file = (
            Split-Path `
                $ObjectListFile `
                -Leaf
        )
    }

    $Manifest |
        ConvertTo-Json -Depth 5 |
        Set-Content `
            -Path $ManifestFile `
            -Encoding UTF8

    Write-Host ""
    Write-Host "Backup berhasil dibuat." `
        -ForegroundColor Green

    Write-Host "Ukuran : $($DumpInfo.Length) byte"
    Write-Host "SHA256 : $($DumpHash.Hash)"
}
catch {
    if (Test-Path $BackupSetDirectory) {
        Remove-Item `
            $BackupSetDirectory `
            -Recurse `
            -Force `
            -ErrorAction SilentlyContinue
    }

    throw
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

$Cutoff = (Get-Date).AddDays(
    -$RetentionDays
)

$BackupSets = Get-ChildItem `
    -Path $BackupDirectory `
    -Directory `
    -Filter "backup_*" `
    -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTimeUtc -Descending

$RemovableBackupSets = $BackupSets |
    Select-Object -Skip $KeepMinimum |
    Where-Object {
        $_.LastWriteTime -lt $Cutoff
    }

foreach ($OldBackupSet in $RemovableBackupSets) {
    Write-Host `
        "Menghapus backup lama: $($OldBackupSet.Name)" `
        -ForegroundColor DarkGray

    Remove-Item `
        $OldBackupSet.FullName `
        -Recurse `
        -Force
}

Write-Host ""
Write-Host "Lokasi backup:" `
    -ForegroundColor White

Write-Host $BackupSetDirectory `
    -ForegroundColor Green