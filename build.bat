@echo off
setlocal

cd /d "%~dp0"

echo ======================================
echo        Gun Mayhem - Windows Build
echo ======================================
echo.

where go >nul 2>nul
if errorlevel 1 (
    echo ERROR: Go was not found in PATH.
    echo Install Go from https://go.dev/dl/ and run this file again.
    echo.
    pause
    exit /b 1
)

echo Go version:
go version
if errorlevel 1 goto :fail

echo.
echo Building GunMayhem.exe...
go build -o GunMayhem.exe .
if errorlevel 1 goto :fail

echo.
echo Build completed successfully.
echo Output: %CD%\GunMayhem.exe
echo.
pause
exit /b 0

:fail
echo.
echo BUILD FAILED.
echo Check the error messages above.
echo.
pause
exit /b 1
