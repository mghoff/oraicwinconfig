# oraicwinconfig
![Version](https://img.shields.io/badge/version-0.1.1-blue.svg)
[![Project Status: Active – The project has reached a stable, usable state and is being actively developed.](https://www.repostatus.org/badges/latest/active.svg)](https://www.repostatus.org/#active)

A Go-based CLI tool to install and configure [Oracle Instant Client](https://www.oracle.com/database/technologies/instant-client/downloads.html) on Windows 11 for use by Oracle's very own `R` package: [ROracle](https://www.oracle.com/database/technologies/appdev/roracle.html)

## Background

This package was built within the context of needing to install and configure for Windows the `Oracle Instant Client` package and SDK - both of which are needed by the `ROracle` package to effectively interface with Oracle databases.

`Roracle` often needs built from source, further requiring the installation of [RTools](https://cran.r-project.org/bin/windows/Rtools/).

`RTools` will look for the existence of the `Oracle Instant Client` package within your System or User `PATH` Environment Variables during buildtime. As such, it is imperitive that this package and its SDK be installed and properly configured on your Windows system.

This package aims to do just that.

## Requirements

  + Windows 11 OS
  + **Optional:* A `tnsnames.ora` file containing Oracle connection details

## Installation

### Option 1: Download the Pre-Built Binary
1. Go to the [Releases](https://github.com/mghoff/oraicwinconfig/releases) page
2. Download the latest executable file: `oraicwinconfig.exe`
3. Run the executable file and follow the prompts

### Option 2: Build from Source
Clone this repository and build using Go:
```bash
git clone https://github.com/mghoff/oraicwinconfig.git
cd oraicwinconfig
.\scripts\build.cmd
```

Following a successful build, a `.\bin` folder will have been created which contains the `oraicwinconfig.exe` executable file along with a `SHA256SUMS` file. You can then run the exectuable file and follow the prompts in your command terminal.

## Details:

This executable will perform the following...
1. Check for existing installation of Oracle InstantClient by looking for the User Environment Variables: `OCI_LIB64` and `TNS_NAMES`.
    + If no existing installation is found, the user will be prompted to accept the default installation directory: `C:/OraClient`.
    + Upon discovering an existing installation, the user will be prompted to overwrite the existing installation.
      + If you choose to overwrite, the existing installation directory and its respective environment variables will be removed completely. In the case of the `OCI_LIB64` and `TNS_ADMIN` user environment variables, the will be overwritten with the paths specified by new installation.
      + If you choose NOT to overwrite, the existing installation will remain and the new installation will be adjacently installed into the base directory of the existing installation. `OCI_LIB64` and `TNS_NAMES` environment variable values will be overwritten with the new installation paths, and the new `OCI_LIB64` path will be added to the `PATH` User Environment Variable. *Note:* The old `OCI_LIB64` directory will remain  in the `PATH` list. 
      + **FINAL NOTE:** With either choice above, if a valid `tnsnames.ora` file is found, it will be temporarily copied the user Downloads folder and then moved to the proper subdirectory of the new installation.
2. Download the Windows-specific `Oracle Instant Client Basic Lite` package and SDK zip files into the user Downloads folder.
3. Unzip the above files into the specified installation directory.
4. Add the installation directory to the `PATH` User Environment Variable.
5. Create and assign *or* reset the `OCI_LIB64` and `TNS_NAMES` User Environment Variables.

Following successful installation and configuration, you should be able to use `RTools` to build `Roracle` from source...

In R, run: 
```
install.packages("path/to/ROracle.zip", repos = NULL, type = "source")
```

**Note:** If updating/upgrading your existing version of `Oracle InstantClient` for use with `ROracle`, you will need to rebuild from source to properly set the environment variables within the `R` package.
