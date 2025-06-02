# oraicwinconfig
Go package for installing and configuring Oracle Instant Client on Windows 11 to be used by the ROracle R package.

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)

## Requirements

  + Windows 11 OS
  + R version >= 4.0
  + Rtools version >= 4.0
  + A `tnsnames.ora` file containing Oracle connection details

## Installation

### Option 1: Download the Pre-Built Binary
1. Go to the [Releases](https://github.com/mghoff/oraicwinconfig/releases) page
2. Download the latest `oraicwinconfig.exe`
3. Run the executable and follow the prompts

### Option 2: Build from Source
Clone this repository and build using Go:
```bash
git clone https://github.com/mghoff/oraicwinconfig.git
cd oraicwinconfig
.\scripts\build.cmd
```

## Notes:

This executable will perform the following...
1. Download into the user's Downloads folder the Windows-specific Oracle Instant Client Basic Lite package and SDK zip files.
2. Unzip the above two zip files into the user-specified installation directory.
3. Add the installation directory to the `PATH` User Environment Variable.
4. Create two new User Environment Variables (`OCI_LIB64` and `TNS_NAMES`) with associated directory paths.
