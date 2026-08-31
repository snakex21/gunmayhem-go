@echo off
setlocal

cd /d "%~dp0"

rem Keep all Go compiler/test scratch data inside the project, never in %%TEMP%%.
set "GOTMPDIR=%CD%\build\go-tmp"
set "GOCACHE=%CD%\build\go-cache"
if not exist "%GOTMPDIR%" mkdir "%GOTMPDIR%"
if not exist "%GOCACHE%" mkdir "%GOCACHE%"

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
echo Running tests...
go test ./...
if errorlevel 1 goto :fail

rem Untagged portable test builds use a valid prerelease version instead of
rem presenting themselves as DEV BUILD. A release tag or explicit environment
rem value still takes precedence.
set "VERSION=v0.16.0"
if defined GUNMAYHEM_VERSION set "VERSION=%GUNMAYHEM_VERSION%"
if not defined GUNMAYHEM_VERSION (
    for /f "delims=" %%V in ('git describe --tags --exact-match 2^>nul') do set "VERSION=%%V"
)
echo Build version: %VERSION%

echo.
echo Building GunMayhem.exe without a console window...
go build -ldflags "-H=windowsgui -X gunmayhem-go/internal/updater.CurrentVersion=%VERSION%" -o GunMayhem.exe .
if errorlevel 1 goto :fail

echo.
echo Packaging LAN-only portable Windows build...
go run .\scripts\package_portable.go
if errorlevel 1 goto :fail

echo.
echo Build completed successfully.
echo Output: %CD%\GunMayhem.exe
echo Output: %CD%\build\GunMayhem-Windows-x64.zip
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
