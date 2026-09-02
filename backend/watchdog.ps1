# Keeps corti-backend.exe running. Checked every minute by the
# XtekAI-Backend-Watchdog scheduled task. If the process isn't running (or
# the port isn't answering), it's (re)started. Logs to watchdog.log next to
# this script for troubleshooting.
$ErrorActionPreference = "SilentlyContinue"
$exePath = "D:\Yitao_Wang\medical-transcription\backend\corti-backend.exe"
$workDir = "D:\Yitao_Wang\medical-transcription\backend"
$logFile = "D:\Yitao_Wang\medical-transcription\backend\watchdog.log"

function Log($msg) {
    "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') $msg" | Out-File -Append -FilePath $logFile
}

$healthy = $false
try {
    $resp = Invoke-WebRequest -Uri "http://localhost:8090/health" -TimeoutSec 3 -UseBasicParsing
    if ($resp.StatusCode -eq 200) { $healthy = $true }
} catch {}

if (-not $healthy) {
    Get-Process -Name "corti-backend" -ErrorAction SilentlyContinue | Stop-Process -Force
    Start-Sleep -Seconds 1
    Start-Process -FilePath $exePath -WorkingDirectory $workDir -WindowStyle Hidden
    Log "Backend was down - restarted it."
} else {
    # Quiet on the happy path so the log doesn't grow unbounded; uncomment for verbose mode.
    # Log "Backend healthy."
}
