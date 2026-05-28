# dev.ps1 - Helper script to run the DMTool backend without port conflicts.
# Usage: ./dev.ps1

$port = 8080

# 1. Find and kill any process using the port
Write-Host "Checking for processes on port $port..." -ForegroundColor Cyan
$process = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess -First 1

if ($process) {
    Write-Host "Terminating existing process (PID: $process) using port $port..." -ForegroundColor Yellow
    Stop-Process -Id $process -Force
    Start-Sleep -Seconds 1
}

# 2. Run the application
Write-Host "Starting DMTool Backend..." -ForegroundColor Green
go run cmd/api/main.go
