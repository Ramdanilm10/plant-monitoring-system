Set-StrictMode -Version Latest

function Get-PlantMonitoringProjectRoot {
    $ProjectRoot = Join-Path `
        $PSScriptRoot `
        ".."

    return (
        Resolve-Path $ProjectRoot
    ).Path
}

function Read-DotEnvFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    if (-not (Test-Path $Path -PathType Leaf)) {
        throw "File environment tidak ditemukan: $Path"
    }

    $Values = @{}

    foreach ($RawLine in Get-Content $Path) {
        $Line = $RawLine.Trim()

        if (
            $Line.Length -eq 0 -or
            $Line.StartsWith("#")
        ) {
            continue
        }

        $SeparatorIndex = $Line.IndexOf("=")

        if ($SeparatorIndex -lt 1) {
            continue
        }

        $Name = $Line.Substring(
            0,
            $SeparatorIndex
        ).Trim()

        $Value = $Line.Substring(
            $SeparatorIndex + 1
        ).Trim()

        if (
            $Value.Length -ge 2 -and
            (
                (
                    $Value.StartsWith('"') -and
                    $Value.EndsWith('"')
                ) -or
                (
                    $Value.StartsWith("'") -and
                    $Value.EndsWith("'")
                )
            )
        ) {
            $Value = $Value.Substring(
                1,
                $Value.Length - 2
            )
        }

        $Values[$Name] = $Value
    }

    return $Values
}

function Get-RequiredEnvironmentValue {
    param(
        [Parameter(Mandatory = $true)]
        [hashtable]$Values,

        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (
        -not $Values.ContainsKey($Name) -or
        [string]::IsNullOrWhiteSpace(
            [string]$Values[$Name]
        )
    ) {
        throw "Variabel $Name belum diatur."
    }

    return [string]$Values[$Name]
}

function Get-OptionalEnvironmentValue {
    param(
        [Parameter(Mandatory = $true)]
        [hashtable]$Values,

        [Parameter(Mandatory = $true)]
        [string]$Name,

        [string]$Fallback = ""
    )

    if (
        $Values.ContainsKey($Name) -and
        -not [string]::IsNullOrWhiteSpace(
            [string]$Values[$Name]
        )
    ) {
        return [string]$Values[$Name]
    }

    return $Fallback
}

function Get-DatabaseConfiguration {
    param(
        [Parameter(Mandatory = $true)]
        [string]$EnvFile,

        [switch]$UseBackupOverrides
    )

    $Values = Read-DotEnvFile `
        -Path $EnvFile

    $DefaultHost = Get-RequiredEnvironmentValue `
        -Values $Values `
        -Name "SUPABASE_DB_HOST"

    $DefaultPort = Get-OptionalEnvironmentValue `
        -Values $Values `
        -Name "SUPABASE_DB_PORT" `
        -Fallback "5432"

    $DefaultUser = Get-RequiredEnvironmentValue `
        -Values $Values `
        -Name "SUPABASE_DB_USER"

    $DefaultPassword = Get-RequiredEnvironmentValue `
        -Values $Values `
        -Name "SUPABASE_DB_PASSWORD"

    $DefaultDatabase = Get-OptionalEnvironmentValue `
        -Values $Values `
        -Name "SUPABASE_DB_NAME" `
        -Fallback "postgres"

    if ($UseBackupOverrides) {
        $DatabaseHost = Get-OptionalEnvironmentValue `
            -Values $Values `
            -Name "SUPABASE_DB_BACKUP_HOST" `
            -Fallback $DefaultHost

        $DatabasePort = Get-OptionalEnvironmentValue `
            -Values $Values `
            -Name "SUPABASE_DB_BACKUP_PORT" `
            -Fallback $DefaultPort

        $DatabaseUser = Get-OptionalEnvironmentValue `
            -Values $Values `
            -Name "SUPABASE_DB_BACKUP_USER" `
            -Fallback $DefaultUser

        $DatabasePassword = Get-OptionalEnvironmentValue `
            -Values $Values `
            -Name "SUPABASE_DB_BACKUP_PASSWORD" `
            -Fallback $DefaultPassword

        $DatabaseName = Get-OptionalEnvironmentValue `
            -Values $Values `
            -Name "SUPABASE_DB_BACKUP_NAME" `
            -Fallback $DefaultDatabase
    }
    else {
        $DatabaseHost = $DefaultHost
        $DatabasePort = $DefaultPort
        $DatabaseUser = $DefaultUser
        $DatabasePassword = $DefaultPassword
        $DatabaseName = $DefaultDatabase
    }

    if ($DatabasePort -notmatch "^\d+$") {
        throw "Port database tidak valid."
    }

    return [PSCustomObject]@{
        Host     = $DatabaseHost
        Port     = $DatabasePort
        User     = $DatabaseUser
        Password = $DatabasePassword
        Database = $DatabaseName
    }
}

function Resolve-PostgresTool {
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet(
            "pg_dump",
            "pg_restore",
            "psql"
        )]
        [string]$ToolName
    )

    foreach ($CommandName in @(
        "$ToolName.exe",
        $ToolName
    )) {
        $Command = Get-Command `
            $CommandName `
            -CommandType Application `
            -ErrorAction SilentlyContinue |
            Select-Object -First 1

        if ($null -ne $Command) {
            if (
                -not [string]::IsNullOrWhiteSpace(
                    $Command.Path
                )
            ) {
                return $Command.Path
            }

            if (
                -not [string]::IsNullOrWhiteSpace(
                    $Command.Source
                )
            ) {
                return $Command.Source
            }
        }
    }

    $SearchRoots = @(
        "$env:ProgramFiles\PostgreSQL",
        "${env:ProgramFiles(x86)}\PostgreSQL"
    )

    foreach ($SearchRoot in $SearchRoots) {
        if (
            [string]::IsNullOrWhiteSpace(
                $SearchRoot
            ) -or
            -not (Test-Path $SearchRoot)
        ) {
            continue
        }

        $Tool = Get-ChildItem `
            -Path $SearchRoot `
            -Filter "$ToolName.exe" `
            -File `
            -Recurse `
            -ErrorAction SilentlyContinue |
            Sort-Object FullName -Descending |
            Select-Object -First 1

        if ($null -ne $Tool) {
            return $Tool.FullName
        }
    }

    throw @"
$ToolName tidak ditemukan.

Install PostgreSQL command-line tools dan pastikan
pg_dump.exe, pg_restore.exe, serta psql.exe tersedia.
"@
}