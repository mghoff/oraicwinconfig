@echo off
setlocal enabledelayedexpansion

set EXECUTABLE=bin\oraicwinconfig.exe
set CHECKSUM_FILE=bin\SHA256SUMS

:: Create bin directory if it doesn't exist
if not exist "bin" mkdir bin

:: Generate SHA256 checksum using certutil
certutil -hashfile %EXECUTABLE% SHA256 | findstr /v "hash" | findstr /v "CertUtil" > %CHECKSUM_FILE%

:: Add filename to checksum file
for /f "tokens=1" %%a in (%CHECKSUM_FILE%) do (
  echo %%a oraicwinconfig.exe > %CHECKSUM_FILE%
)

echo Generated checksums for %EXECUTABLE%