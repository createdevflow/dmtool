@echo off
REM run.bat - Kill port 8080 and start backend

set PORT=8080

echo Checking for process on port %PORT%...
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :%PORT% ^| findstr LISTENING') do (
    echo Terminating process %%a using port %PORT%...
    taskkill /F /PID %%a
)

timeout /t 1 /nobreak > nul

echo Starting DMTool Backend...
go run cmd/api/main.go
