param(
    [switch]$SkipBuild,
    [switch]$NoBrowser,

    # Opsional. Bisa diisi path cloudflared.exe secara manual.
    [string]$CloudflaredPath = ""
)

Set-StrictMode -Version Latest

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path `
    -Parent `
    $PSCommandPath

$BackendDirectory = Join-Path `
    $ProjectRoot `
    "backend"

$FrontendDirectory = Join-Path `
    $ProjectRoot `
    "frontend"

$FrontendIndexPath = Join-Path `
    $FrontendDirectory `
    "dist\index.html"

$RuntimeDirectory = Join-Path `
    $env:LOCALAPPDATA `
    "PlantMonitoringSystem\runtime"

$BackendExecutable = Join-Path `
    $RuntimeDirectory `
    "plant-monitoring-backend.exe"

$BackendPidFile = Join-Path `
    $RuntimeDirectory `
    "backend.pid"

$TunnelPidFile = Join-Path `
    $RuntimeDirectory `
    "cloudflared.pid"

$BackendStandardOutput = Join-Path `
    $RuntimeDirectory `
    "backend.stdout.log"

$BackendStandardError = Join-Path `
    $RuntimeDirectory `
    "backend.stderr.log"

$TunnelStandardOutput = Join-Path `
    $RuntimeDirectory `
    "tunnel.stdout.log"

$TunnelStandardError = Join-Path `
    $RuntimeDirectory `
    "tunnel.stderr.log"

$PublicUrlFile = Join-Path `
    $RuntimeDirectory `
    "public-url.txt"

$HealthUrl = "http://127.0.0.1:8080/api/health"

$BackendProcess = $null
$TunnelProcess = $null

function Resolve-Executable {
    param(
        [Parameter(Mandatory = $true)]
        [string]$DisplayName,

        [Parameter(Mandatory = $true)]
        [string[]]$CommandNames,

        [string]$ExplicitPath = "",

        [string[]]$AdditionalPaths = @()
    )

    # Prioritas 1:
    # Path eksplisit dari parameter script.
    if (
        -not [string]::IsNullOrWhiteSpace(
            $ExplicitPath
        )
    ) {
        $ExpandedPath = [Environment]::ExpandEnvironmentVariables(
            $ExplicitPath
        )

        if (Test-Path $ExpandedPath -PathType Leaf) {
            return (
                Resolve-Path $ExpandedPath
            ).Path
        }

        throw "$DisplayName tidak ditemukan pada path: $ExplicitPath"
    }

    # Prioritas 2:
    # Cari menggunakan Get-Command.
    foreach ($CommandName in $CommandNames) {
        $Command = Get-Command `
            $CommandName `
            -CommandType Application `
            -ErrorAction SilentlyContinue |
            Select-Object `
                -First 1

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
                ) -and
                (Test-Path $Command.Source -PathType Leaf)
            ) {
                return $Command.Source
            }
        }
    }

    # Prioritas 3:
    # Cari menggunakan where.exe.
    $WhereCommand = Get-Command `
        "where.exe" `
        -CommandType Application `
        -ErrorAction SilentlyContinue

    if ($null -ne $WhereCommand) {
        foreach ($CommandName in $CommandNames) {
            try {
                $WhereResults = & $WhereCommand.Path `
                    $CommandName `
                    2>$null

                foreach ($WhereResult in $WhereResults) {
                    $Candidate = String($WhereResult).Trim()

                    if (
                        -not [string]::IsNullOrWhiteSpace(
                            $Candidate
                        ) -and
                        (Test-Path $Candidate -PathType Leaf)
                    ) {
                        return (
                            Resolve-Path $Candidate
                        ).Path
                    }
                }
            }
            catch {
                # Lanjut ke metode pencarian berikutnya.
            }
        }
    }

    # Prioritas 4:
    # Cari pada daftar lokasi instalasi umum.
    foreach ($AdditionalPath in $AdditionalPaths) {
        if (
            [string]::IsNullOrWhiteSpace(
                $AdditionalPath
            )
        ) {
            continue
        }

        $ExpandedPath = [Environment]::ExpandEnvironmentVariables(
            $AdditionalPath
        )

        if (Test-Path $ExpandedPath -PathType Leaf) {
            return (
                Resolve-Path $ExpandedPath
            ).Path
        }
    }

    throw @"
$DisplayName tidak ditemukan.

Command yang dicari:
$($CommandNames -join ", ")

Pastikan aplikasi sudah terinstal atau jalankan launcher dengan:
-CloudflaredPath "C:\lokasi\cloudflared.exe"
"@
}

function Resolve-CloudflaredExecutable {
    param(
        [string]$ExplicitPath = ""
    )

    $CommonPaths = @(
        "C:\Tools\cloudflared\cloudflared.exe",

        "$env:ProgramFiles\cloudflared\cloudflared.exe",

        "${env:ProgramFiles(x86)}\cloudflared\cloudflared.exe",

        "$env:LOCALAPPDATA\cloudflared\cloudflared.exe",

        "$env:LOCALAPPDATA\Programs\cloudflared\cloudflared.exe",

        "$env:LOCALAPPDATA\Microsoft\WinGet\Links\cloudflared.exe",

        "$env:USERPROFILE\.cloudflared\cloudflared.exe"
    )

    try {
        return Resolve-Executable `
            -DisplayName "Cloudflared" `
            -CommandNames @(
                "cloudflared.exe",
                "cloudflared"
            ) `
            -ExplicitPath $ExplicitPath `
            -AdditionalPaths $CommonPaths
    }
    catch {
        # WinGet sering menyimpan cloudflared di:
        #
        # AppData\Local\Microsoft\WinGet\Packages\
        # Cloudflare.cloudflared_...\cloudflared.exe

        $WinGetPackagesDirectory = Join-Path `
            $env:LOCALAPPDATA `
            "Microsoft\WinGet\Packages"

        if (Test-Path $WinGetPackagesDirectory) {
            $PackageDirectories = Get-ChildItem `
                -Path $WinGetPackagesDirectory `
                -Directory `
                -Filter "Cloudflare.cloudflared_*" `
                -ErrorAction SilentlyContinue

            foreach ($PackageDirectory in $PackageDirectories) {
                $CloudflaredFile = Get-ChildItem `
                    -Path $PackageDirectory.FullName `
                    -Filter "cloudflared.exe" `
                    -File `
                    -Recurse `
                    -ErrorAction SilentlyContinue |
                    Select-Object `
                        -First 1

                if ($null -ne $CloudflaredFile) {
                    return $CloudflaredFile.FullName
                }
            }
        }

        throw $_
    }
}

function Stop-ManagedProcess {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PidFile,

        [Parameter(Mandatory = $true)]
        [string]$ProcessName
    )

    if (-not (Test-Path $PidFile)) {
        return
    }

    $StoredProcessId = Get-Content `
        $PidFile `
        -ErrorAction SilentlyContinue |
        Select-Object `
            -First 1

    if (
        -not [string]::IsNullOrWhiteSpace(
            $StoredProcessId
        )
    ) {
        $ExistingProcess = Get-Process `
            -Id ([int]$StoredProcessId) `
            -ErrorAction SilentlyContinue

        if ($null -ne $ExistingProcess) {
            Write-Host `
                "Menghentikan proses $ProcessName lama..." `
                -ForegroundColor Yellow

            Stop-Process `
                -Id $ExistingProcess.Id `
                -Force `
                -ErrorAction SilentlyContinue
        }
    }

    Remove-Item `
        $PidFile `
        -Force `
        -ErrorAction SilentlyContinue
}

function Test-LocalPort {
    param(
        [Parameter(Mandatory = $true)]
        [string]$HostName,

        [Parameter(Mandatory = $true)]
        [int]$Port
    )

    $TcpClient = New-Object `
        System.Net.Sockets.TcpClient

    try {
        $ConnectTask = $TcpClient.ConnectAsync(
            $HostName,
            $Port
        )

        $Completed = $ConnectTask.Wait(
            700
        )

        return (
            $Completed -and
            $TcpClient.Connected
        )
    }
    catch {
        return $false
    }
    finally {
        $TcpClient.Dispose()
    }
}

function Get-LogContent {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Paths
    )

    $LogParts = @()

    foreach ($Path in $Paths) {
        if (-not (Test-Path $Path)) {
            continue
        }

        try {
            $Content = Get-Content `
                $Path `
                -Raw `
                -ErrorAction Stop

            if (
                -not [string]::IsNullOrWhiteSpace(
                    $Content
                )
            ) {
                $LogParts += $Content
            }
        }
        catch {
            # File mungkin sedang ditulis.
        }
    }

    return $LogParts -join "`n"
}

function Stop-StartedProcesses {
    if (
        $null -ne $TunnelProcess -and
        -not $TunnelProcess.HasExited
    ) {
        Stop-Process `
            -Id $TunnelProcess.Id `
            -Force `
            -ErrorAction SilentlyContinue
    }

    if (
        $null -ne $BackendProcess -and
        -not $BackendProcess.HasExited
    ) {
        Stop-Process `
            -Id $BackendProcess.Id `
            -Force `
            -ErrorAction SilentlyContinue
    }

    Remove-Item `
        $TunnelPidFile `
        -Force `
        -ErrorAction SilentlyContinue

    Remove-Item `
        $BackendPidFile `
        -Force `
        -ErrorAction SilentlyContinue
}

try {
    Write-Host ""
    Write-Host "========================================" `
        -ForegroundColor Cyan

    Write-Host " Plant Monitoring Production Launcher" `
        -ForegroundColor Cyan

    Write-Host "========================================" `
        -ForegroundColor Cyan

    Write-Host ""

    if (-not (Test-Path $BackendDirectory)) {
        throw "Folder backend tidak ditemukan: $BackendDirectory"
    }

    if (-not (Test-Path $FrontendDirectory)) {
        throw "Folder frontend tidak ditemukan: $FrontendDirectory"
    }

    Write-Host `
        "Mencari executable yang diperlukan..." `
        -ForegroundColor Cyan

    $GoCommand = Resolve-Executable `
        -DisplayName "Go" `
        -CommandNames @(
            "go.exe",
            "go"
        )

    $NpmCommand = Resolve-Executable `
        -DisplayName "NPM" `
        -CommandNames @(
            "npm.cmd",
            "npm.exe",
            "npm"
        )

    $CloudflaredCommand = Resolve-CloudflaredExecutable `
        -ExplicitPath $CloudflaredPath

    Write-Host `
        "Go          : $GoCommand" `
        -ForegroundColor DarkGray

    Write-Host `
        "NPM         : $NpmCommand" `
        -ForegroundColor DarkGray

    Write-Host `
        "Cloudflared : $CloudflaredCommand" `
        -ForegroundColor DarkGray

    New-Item `
        -ItemType Directory `
        -Path $RuntimeDirectory `
        -Force |
        Out-Null

    Stop-ManagedProcess `
        -PidFile $TunnelPidFile `
        -ProcessName "Cloudflare Tunnel"

    Stop-ManagedProcess `
        -PidFile $BackendPidFile `
        -ProcessName "backend"

    Remove-Item `
        $BackendStandardOutput, `
        $BackendStandardError, `
        $TunnelStandardOutput, `
        $TunnelStandardError, `
        $PublicUrlFile `
        -Force `
        -ErrorAction SilentlyContinue

    if (
        Test-LocalPort `
            -HostName "127.0.0.1" `
            -Port 8080
    ) {
        throw @"
Port 8080 sedang digunakan oleh proses lain.

Stop backend yang masih aktif dengan Ctrl + C,
kemudian jalankan launcher kembali.
"@
    }

    if (-not $SkipBuild) {
        Write-Host `
            "[1/5] Build frontend production..." `
            -ForegroundColor Cyan

        Push-Location $FrontendDirectory

        try {
            & $NpmCommand run build

            if ($LASTEXITCODE -ne 0) {
                throw "npm run build gagal dengan exit code $LASTEXITCODE"
            }
        }
        finally {
            Pop-Location
        }
    }
    else {
        Write-Host `
            "[1/5] Build frontend dilewati." `
            -ForegroundColor Yellow
    }

    if (-not (Test-Path $FrontendIndexPath)) {
        throw @"
Frontend production build tidak ditemukan.

Jalankan launcher tanpa parameter -SkipBuild.
"@
    }

    Write-Host `
        "[2/5] Build backend executable..." `
        -ForegroundColor Cyan

    Push-Location $BackendDirectory

    try {
        & $GoCommand build `
            -o $BackendExecutable `
            .

        if ($LASTEXITCODE -ne 0) {
            throw "go build gagal dengan exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }

    Write-Host `
        "[3/5] Menjalankan backend..." `
        -ForegroundColor Cyan

    $BackendProcess = Start-Process `
        -FilePath $BackendExecutable `
        -WorkingDirectory $BackendDirectory `
        -RedirectStandardOutput $BackendStandardOutput `
        -RedirectStandardError $BackendStandardError `
        -WindowStyle Hidden `
        -PassThru

    Set-Content `
        -Path $BackendPidFile `
        -Value $BackendProcess.Id

    $BackendReady = $false

    for (
        $Attempt = 1;
        $Attempt -le 45;
        $Attempt++
    ) {
        if ($BackendProcess.HasExited) {
            $BackendLogs = Get-LogContent `
                -Paths @(
                    $BackendStandardOutput,
                    $BackendStandardError
                )

            throw @"
Backend berhenti sebelum siap.

Log backend:
$BackendLogs
"@
        }

        try {
            $HealthResponse = Invoke-RestMethod `
                -Method Get `
                -Uri $HealthUrl `
                -TimeoutSec 2

            if (
                $HealthResponse.status -eq "ok"
            ) {
                $BackendReady = $true

                break
            }
        }
        catch {
            # Backend masih startup.
        }

        Start-Sleep `
            -Seconds 1
    }

    if (-not $BackendReady) {
        $BackendLogs = Get-LogContent `
            -Paths @(
                $BackendStandardOutput,
                $BackendStandardError
            )

        throw @"
Backend tidak siap setelah 45 detik.

Log backend:
$BackendLogs
"@
    }

    Write-Host `
        "Backend aktif: http://localhost:8080" `
        -ForegroundColor Green

    Write-Host `
        "[4/5] Membuka Cloudflare Quick Tunnel..." `
        -ForegroundColor Cyan

    $TunnelProcess = Start-Process `
        -FilePath $CloudflaredCommand `
        -ArgumentList @(
            "tunnel",
            "--url",
            "http://127.0.0.1:8080"
        ) `
        -WorkingDirectory $ProjectRoot `
        -RedirectStandardOutput $TunnelStandardOutput `
        -RedirectStandardError $TunnelStandardError `
        -WindowStyle Hidden `
        -PassThru

    Set-Content `
        -Path $TunnelPidFile `
        -Value $TunnelProcess.Id

    $PublicUrl = $null

    $PublicUrlPattern =
        "https://[a-z0-9-]+\.trycloudflare\.com"

    for (
        $Attempt = 1;
        $Attempt -le 60;
        $Attempt++
    ) {
        if ($TunnelProcess.HasExited) {
            $TunnelLogs = Get-LogContent `
                -Paths @(
                    $TunnelStandardOutput,
                    $TunnelStandardError
                )

            throw @"
Cloudflare Tunnel berhenti sebelum URL tersedia.

Log tunnel:
$TunnelLogs
"@
        }

        $TunnelLogs = Get-LogContent `
            -Paths @(
                $TunnelStandardOutput,
                $TunnelStandardError
            )

        $UrlMatch = [regex]::Match(
            $TunnelLogs,
            $PublicUrlPattern
        )

        if ($UrlMatch.Success) {
            $PublicUrl = $UrlMatch.Value

            break
        }

        Start-Sleep `
            -Seconds 1
    }

    if (
        [string]::IsNullOrWhiteSpace(
            $PublicUrl
        )
    ) {
        $TunnelLogs = Get-LogContent `
            -Paths @(
                $TunnelStandardOutput,
                $TunnelStandardError
            )

        throw @"
URL Cloudflare tidak ditemukan setelah 60 detik.

Log tunnel:
$TunnelLogs
"@
    }

    Set-Content `
        -Path $PublicUrlFile `
        -Value $PublicUrl

    Write-Host `
        "[5/5] Sistem berhasil dijalankan." `
        -ForegroundColor Green

    Write-Host ""
    Write-Host "LOCAL URL:" `
        -ForegroundColor White

    Write-Host `
        "http://localhost:8080" `
        -ForegroundColor Green

    Write-Host ""
    Write-Host "PUBLIC URL:" `
        -ForegroundColor White

    Write-Host `
        $PublicUrl `
        -ForegroundColor Green

    Write-Host ""
    Write-Host "HEALTH CHECK:" `
        -ForegroundColor White

    Write-Host `
        "$PublicUrl/api/health" `
        -ForegroundColor Green

    Write-Host ""
    Write-Host "Untuk menghentikan sistem:" `
        -ForegroundColor White

    Write-Host `
        ".\stop-production.ps1" `
        -ForegroundColor Yellow

    if (-not $NoBrowser) {
        Start-Process $PublicUrl
    }
}
catch {
    Write-Host ""
    Write-Host "STARTUP GAGAL" `
        -ForegroundColor Red

    Write-Host `
        $_.Exception.Message `
        -ForegroundColor Red

    Stop-StartedProcesses

    exit 1
}