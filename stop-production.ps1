Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RuntimeDirectory = Join-Path `
    $env:LOCALAPPDATA `
    "PlantMonitoringSystem\runtime"

$BackendPidFile = Join-Path `
    $RuntimeDirectory `
    "backend.pid"

$TunnelPidFile = Join-Path `
    $RuntimeDirectory `
    "cloudflared.pid"

function Stop-ProcessFromPidFile {
    param(
        [Parameter(Mandatory = $true)]
        [string]$PidFile,

        [Parameter(Mandatory = $true)]
        [string]$DisplayName
    )

    if (-not (Test-Path $PidFile)) {
        Write-Host `
            "$DisplayName tidak sedang dikelola oleh launcher." `
            -ForegroundColor DarkGray

        return
    }

    $StoredProcessId = Get-Content `
        $PidFile `
        -ErrorAction SilentlyContinue |
        Select-Object `
            -First 1

    if (
        [string]::IsNullOrWhiteSpace(
            $StoredProcessId
        )
    ) {
        Remove-Item `
            $PidFile `
            -Force `
            -ErrorAction SilentlyContinue

        return
    }

    $Process = Get-Process `
        -Id ([int]$StoredProcessId) `
        -ErrorAction SilentlyContinue

    if ($null -eq $Process) {
        Write-Host `
            "$DisplayName sudah berhenti." `
            -ForegroundColor DarkGray
    }
    else {
        Stop-Process `
            -Id $Process.Id `
            -Force

        Write-Host `
            "$DisplayName berhasil dihentikan." `
            -ForegroundColor Green
    }

    Remove-Item `
        $PidFile `
        -Force `
        -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "Menghentikan Plant Monitoring System..." `
    -ForegroundColor Cyan

# Tunnel dihentikan terlebih dahulu agar akses publik
# langsung tertutup sebelum backend dimatikan.
Stop-ProcessFromPidFile `
    -PidFile $TunnelPidFile `
    -DisplayName "Cloudflare Tunnel"

Stop-ProcessFromPidFile `
    -PidFile $BackendPidFile `
    -DisplayName "Backend"

Write-Host ""
Write-Host "Plant Monitoring System sudah berhenti." `
    -ForegroundColor Green