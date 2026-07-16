Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = (
    Resolve-Path (
        Join-Path `
            $PSScriptRoot `
            ".."
    )
).Path

$script:Failures = @()
$script:Warnings = @()

function Add-Failure {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    $script:Failures += $Message
}

function Add-Warning {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message
    )

    $script:Warnings += $Message
}

function Get-RelativeProjectPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$FullPath
    )

    $AbsolutePath = [System.IO.Path]::GetFullPath(
        $FullPath
    )

    $RelativePath = $AbsolutePath.Substring(
        $ProjectRoot.Length
    )

    $RelativePath = $RelativePath.TrimStart(
        [char]'\',
        [char]'/'
    )

    return $RelativePath.Replace(
        "\",
        "/"
    )
}

function Test-IsExcludedPath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$RelativePath
    )

    return (
        $RelativePath -match "(^|/)\.git(/|$)" -or
        $RelativePath -match "(^|/)node_modules(/|$)" -or
        $RelativePath -match "(^|/)dist(/|$)" -or
        $RelativePath -match "(^|/)\.vite(/|$)"
    )
}

Write-Host ""
Write-Host "========================================" `
    -ForegroundColor Cyan

Write-Host " Plant Monitoring GitHub Safety Check" `
    -ForegroundColor Cyan

Write-Host "========================================" `
    -ForegroundColor Cyan

Write-Host ""

# =========================================================
# 1. Pemeriksaan file wajib
# =========================================================

$RequiredFiles = @(
    ".gitignore",
    ".gitattributes",
    "README.md",
    "backend/.env.example",
    "backend/go.mod",
    "frontend/package.json"
)

foreach ($RequiredFile in $RequiredFiles) {
    $RequiredPath = Join-Path `
        $ProjectRoot `
        $RequiredFile

    if (-not (Test-Path $RequiredPath -PathType Leaf)) {
        Add-Failure `
            "File wajib belum tersedia: $RequiredFile"
    }
}

# =========================================================
# 2. Pemeriksaan aturan .gitignore
#
# Pemeriksaan dilakukan per baris agar kompatibel dengan:
#
# Windows CRLF
# Unix LF
# whitespace di akhir baris
# =========================================================

$GitIgnorePath = Join-Path `
    $ProjectRoot `
    ".gitignore"

if (Test-Path $GitIgnorePath -PathType Leaf) {
    $GitIgnoreRules = @(
        Get-Content $GitIgnorePath |
        ForEach-Object {
            $_.Trim()
        } |
        Where-Object {
            $_ -ne "" -and
            -not $_.StartsWith("#")
        }
    )

    $RequiredGitIgnoreRules = @(
        "**/.env",
        "*.dump",
        "*.zip",
        "frontend/node_modules/",
        "frontend/dist/"
    )

    foreach ($RequiredRule in $RequiredGitIgnoreRules) {
        if ($GitIgnoreRules -notcontains $RequiredRule) {
            Add-Failure `
                ".gitignore belum memiliki aturan: $RequiredRule"
        }
    }

    $EnvironmentExampleAllowed =
        $GitIgnoreRules -contains "!**/.env.example"

    if (-not $EnvironmentExampleAllowed) {
        Add-Failure `
            ".gitignore belum mengizinkan !**/.env.example"
    }
}
else {
    Add-Failure `
        "File .gitignore tidak ditemukan pada root project."
}

# =========================================================
# 3. Pemeriksaan file arsip, backup, dan private key
# =========================================================

$ForbiddenExtensions = @(
    ".zip",
    ".rar",
    ".7z",
    ".dump",
    ".backup",
    ".bak",
    ".pem",
    ".key",
    ".pfx",
    ".p12",
    ".jks"
)

$ProjectFiles = @(
    Get-ChildItem `
        -Path $ProjectRoot `
        -File `
        -Recurse `
        -ErrorAction SilentlyContinue
)

foreach ($File in $ProjectFiles) {
    $RelativePath = Get-RelativeProjectPath `
        -FullPath $File.FullName

    if (Test-IsExcludedPath $RelativePath) {
        continue
    }

    $Extension = $File.Extension.ToLowerInvariant()

    if ($ForbiddenExtensions -contains $Extension) {
        Add-Failure `
            "File sensitif atau arsip masih berada dalam project: $RelativePath"
    }
}

# =========================================================
# 4. Pemeriksaan kemungkinan secret di source code
# =========================================================

$AllowedTextExtensions = @(
    ".go",
    ".js",
    ".jsx",
    ".ts",
    ".tsx",
    ".json",
    ".css",
    ".html",
    ".sql",
    ".md",
    ".ps1",
    ".yml",
    ".yaml",
    ".ino",
    ".cpp",
    ".h",
    ".hpp",
    ".example"
)

$SecretPatterns = @(
    @{
        Name = "GitHub access token"

        Pattern =
            "github_pat_[A-Za-z0-9_]{20,}|gh[pousr]_[A-Za-z0-9]{20,}"
    },

    @{
        Name = "Private key"

        Pattern =
            "-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----"
    },

    @{
        Name = "PostgreSQL URI dengan password"

        Pattern =
            "postgres(?:ql)?://[^:\s]+:[^@\s]+@"
    },

    @{
        Name = "Hardcoded application credential"

        Pattern =
            '(?i)^\s*(JWT_SECRET|SUPABASE_DB_PASSWORD|DEVICE_API_KEY|BLYNK_TOKEN|BLYNK_AUTH_TOKEN|WIFI_PASSWORD)\s*[:=]\s*["'']?[^"''\s#]{8,}'
    },

    @{
        Name = "Kemungkinan credential firmware"

        Pattern =
            '(?i)\b(auth|wifi_password|wifiPassword)\b[^=\r\n]{0,30}=\s*["''][^"'']{8,}["'']'
    }
)

$PlaceholderPattern = @(
    "(?i)YOUR_",
    "(?i)GENERATE_",
    "(?i)CHANGE_ME",
    "(?i)PLACEHOLDER",
    "(?i)EXAMPLE",
    "(?i)DUMMY",
    "(?i)SAMPLE",
    "(?i)REPLACE_ME",
    "(?i)SHA256_",
    "(?i)example\.supabase\.co",
    "(?i)example_database_user",
    "(?i)os\.Getenv",
    "(?i)Getenv",
    "(?i)process\.env",
    "(?i)import\.meta\.env"
) -join "|"

foreach ($File in $ProjectFiles) {
    $RelativePath = Get-RelativeProjectPath `
        -FullPath $File.FullName

    if (Test-IsExcludedPath $RelativePath) {
        continue
    }

    # Jangan memeriksa regex checker terhadap dirinya sendiri.
    if (
        $RelativePath -eq
        "scripts/pre-github-check.ps1"
    ) {
        continue
    }

    # File .env asli tidak dibaca agar nilainya tidak
    # tercetak atau diproses checker.
    if (
        $RelativePath -match
        "(^|/)\.env($|\.)" -and
        $RelativePath -notmatch
        "\.example$"
    ) {
        continue
    }

    $Extension = $File.Extension.ToLowerInvariant()

    if ($AllowedTextExtensions -notcontains $Extension) {
        continue
    }

    $LineNumber = 0

    foreach ($Line in Get-Content $File.FullName) {
        $LineNumber++

        if ($Line -match $PlaceholderPattern) {
            continue
        }

        foreach ($SecretPattern in $SecretPatterns) {
            if ($Line -match $SecretPattern.Pattern) {
                Add-Failure (
                    "{0} terdeteksi pada {1}:{2}" -f
                    $SecretPattern.Name,
                    $RelativePath,
                    $LineNumber
                )
            }
        }
    }
}

# =========================================================
# 5. Pemeriksaan konfigurasi Git
# =========================================================

$GitCommand = Get-Command `
    "git" `
    -CommandType Application `
    -ErrorAction SilentlyContinue

if ($null -eq $GitCommand) {
    Add-Warning `
        "Git belum ditemukan pada PATH."
}
else {
    Push-Location $ProjectRoot

    try {
        & git rev-parse `
            --is-inside-work-tree `
            1>$null `
            2>$null

        $IsGitRepository = (
            $LASTEXITCODE -eq 0
        )

        if (-not $IsGitRepository) {
            Add-Warning `
                "Repository Git belum diinisialisasi."
        }
        else {
            $TrackedFiles = @(
                & git ls-files
            )

            foreach ($TrackedFile in $TrackedFiles) {
                $NormalizedTrackedFile =
                    $TrackedFile.Replace(
                        "\",
                        "/"
                    )

                $IsEnvironmentFile =
                    $NormalizedTrackedFile -match
                    "(^|/)\.env($|\.)"

                $IsEnvironmentExample =
                    $NormalizedTrackedFile -match
                    "\.env(\..+)?\.example$" -or
                    $NormalizedTrackedFile -match
                    "\.env\.example$"

                $IsSensitiveExtension =
                    $NormalizedTrackedFile -match
                    "\.(pem|key|pfx|p12|jks|dump|backup|bak|zip|rar|7z)$"

                if (
                    (
                        $IsEnvironmentFile -and
                        -not $IsEnvironmentExample
                    ) -or
                    $IsSensitiveExtension
                ) {
                    Add-Failure `
                        "File sensitif sudah masuk Git index: $NormalizedTrackedFile"
                }
            }

            $BackendEnvPath = Join-Path `
                $ProjectRoot `
                "backend\.env"

            if (Test-Path $BackendEnvPath -PathType Leaf) {
                & git check-ignore `
                    -q `
                    "backend/.env"

                if ($LASTEXITCODE -ne 0) {
                    Add-Failure `
                        "backend/.env belum diabaikan oleh Git."
                }
            }

            & git check-ignore `
                -q `
                "backend/.env.example"

            if ($LASTEXITCODE -eq 0) {
                Add-Failure `
                    "backend/.env.example ikut diabaikan Git dan tidak dapat di-commit."
            }
        }
    }
    finally {
        Pop-Location
    }
}

# =========================================================
# 6. Hasil pemeriksaan
# =========================================================

if ($script:Warnings.Count -gt 0) {
    Write-Host "PERINGATAN:" `
        -ForegroundColor Yellow

    foreach ($Warning in $script:Warnings) {
        Write-Host `
            "  - $Warning" `
            -ForegroundColor Yellow
    }

    Write-Host ""
}

if ($script:Failures.Count -gt 0) {
    Write-Host "PEMERIKSAAN GAGAL:" `
        -ForegroundColor Red

    foreach ($Failure in $script:Failures) {
        Write-Host `
            "  - $Failure" `
            -ForegroundColor Red
    }

    Write-Host ""
    Write-Host `
        "Repository belum aman untuk GitHub." `
        -ForegroundColor Red

    exit 1
}

Write-Host `
    "Semua pemeriksaan berhasil." `
    -ForegroundColor Green

Write-Host `
    "Repository siap untuk proses Git berikutnya." `
    -ForegroundColor Green