@echo off
setlocal enabledelayedexpansion

:: Set version number
set VERSION=0.1.0
set EXECUTABLE=bin\oraicwinconfig.exe
set CHECKSUM_FILE=bin\SHA256SUMS

:: Create bin directory if it doesn't exist
if not exist "bin" mkdir bin

:: Get build timestamp
for /f "tokens=2 delims==" %%I in ('wmic os get localdatetime /value') do set datetime=%%I
set BUILDTIME=%datetime:~0,8%-%datetime:~8,6%

:: Get git commit hash
for /f %%I in ('git rev-parse HEAD') do set COMMIT=%%I

:: Get Go version
for /f "tokens=3" %%I in ('go version') do set GOVERSION=%%I

:: Build the executable with version information
echo Building version %VERSION%...
go build -o %EXECUTABLE% -ldflags ^
"-X github.com/mghoff/oraicwinconfig/internal.Version=%VERSION% ^
 -X github.com/mghoff/oraicwinconfig/internal.BuildTime=%BUILDTIME% ^
 -X github.com/mghoff/oraicwinconfig/internal.GitCommit=%COMMIT% ^
 -X github.com/mghoff/oraicwinconfig/internal.GoVersion=%GOVERSION%"

:: Generate checksums using certutil
certutil -hashfile %EXECUTABLE% SHA256 | findstr /v "hash" | findstr /v "CertUtil" > %CHECKSUM_FILE%
for /f "tokens=1" %%a in (%CHECKSUM_FILE%) do (
  echo %%a oraicwinconfig.exe > %CHECKSUM_FILE%
)

echo Build v%VERSION% complete!