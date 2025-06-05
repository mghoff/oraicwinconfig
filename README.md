# oraicwinconfig
Go package for installing and configuring Oracle Instant Client on Windows 11 to be used by the `ROracle` R package.

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)

## Background

This package was built within the context of needing to install and configure for Windows the [Oracle Instant Client](https://www.oracle.com/database/technologies/instant-client/downloads.html) package and SDK - both of which are needed for use by [ROracle](https://www.oracle.com/database/technologies/appdev/roracle.html) for interfacing with Oracle databases.

`Roracle` often needs built from source, further requiring the installation of [RTools](https://cran.r-project.org/bin/windows/Rtools/).

`RTools` will look for the existence of the `Oracle Instant Client` package within your System or User `Path` Environment Variables during buildtime, so it is imperitive to have this package and its SDK installed and properly configured on your Windows system.

This package aims to do just that.

## Requirements

  + Windows 11 OS
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

Following a successful build, the `oraicwinconfig.exe` executable will be created in `.\dist` which you can then run and follow the prompts therein.

## Details:

This executable will perform the following...
1. Download into the user's Downloads folder the Windows-specific `Oracle Instant Client Basic Lite` package and `SDK` zip files.
2. Unzip the above files into either the default or the user-specified installation directory.
3. Add the installation directory to the `PATH` User Environment Variable.
4. Create two new User Environment Variables (`OCI_LIB64` and `TNS_NAMES`) with associated directory paths.
    + **Note:** If you have a `tnsnames.ora` file, you must copy it to the directory specified by the `TNS_NAMES` environment variable.

Following successful installation and configuration, you should
be able to use `RTools` to build `Roracle` from source...

In R, run: 
```
install.packages("path/to/ROracle.zip", repos = NULL, type = "source")
```
