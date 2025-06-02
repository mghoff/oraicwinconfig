@echo off
setlocal enabledelayedexpansion

:: Set version number
set VERSION=1.0.0

:: Create dist directory if it doesn't exist
if not exist "dist" mkdir dist

:: Get build timestamp
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set datetime=%%I
set BUILDTIME=%datetime:~0,8%-%datetime:~8,6%

:: Get git commit hash
for /f %%I in ('git rev-parse HEAD') do set COMMIT=%%I

:: Build the executable with version information
echo Building version %VERSION%...
go build -o dist\oraicwinconfig.exe -ldflags "-X github.com/mghoff/oraicwinconfig/internal.Version=%VERSION% -X github.com/mghoff/oraicwinconfig/internal.BuildTime=%BUILDTIME% -X github.com/mghoff/oraicwinconfig/internal.GitCommit=%COMMIT%"

:: Generate checksums using certutil
certutil -hashfile dist\oraicwinconfig.exe SHA256 | findstr /v "hash" | findstr /v "CertUtil" > dist\SHA256SUMS
for /f "tokens=1" %%a in (dist\SHA256SUMS) do (
    echo %%a  oraicwinconfig.exe> dist\SHA256SUMS
)

echo Build v%VERSION% complete!