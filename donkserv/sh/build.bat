@echo off
setlocal

set SCRIPT_DIR=%~dp0
set PROJECT_DIR=%SCRIPT_DIR%..

cd /d "%PROJECT_DIR%"

echo Building donk...
go build -ldflags="-s -w" -o sh\donk.exe ./cmd/...

if %errorlevel% equ 0 (
    echo Build successful: sh\donk.exe
) else (
    echo Build failed!
    exit /b %errorlevel%
)
