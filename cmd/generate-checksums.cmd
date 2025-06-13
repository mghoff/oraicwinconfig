@echo off
setlocal enabledelayedexpansion

set EXECUTABLE=dist\oraicwinconfig.exe
set CHECKSUM_FILE=dist\SHA256SUMS

:: Create dist directory if it doesn't exist
if not exist "dist" mkdir dist

:: Generate SHA256 checksum using certutil
certutil -hashfile %EXECUTABLE% SHA256 | findstr /v "hash" | findstr /v "CertUtil" > %CHECKSUM_FILE%

:: Add filename to checksum file
for /f "tokens=1" %%a in (%CHECKSUM_FILE%) do (
    echo %%a  %EXECUTABLE%> %CHECKSUM_FILE%
)

echo Generated checksums for %EXECUTABLE%