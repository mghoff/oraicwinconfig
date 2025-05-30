# oraicwinconfig
Go package for installing and configuring Oracle Instant Client on Windows 11 to be used by the ROracle R package.

## Requirements:

  + Windows 11 OS
  + R version >= 4.0
  + Rtools version >= 4.0
  + A `tnsnames.ora` file containing Oracle connection details

## How to use:

  + Either clone this entire repo to your local drive or simply download just the executable file: `oraicwinconfig.exe`.
  + Once downloaded, run `oraicwinconfig.exe` and follow the prompts.
  + Following successful installation and configuration, copy your `tnsnames.ora` file to the `install/path/location/network/admin` folder.

## Notes:

This executable will perform the following...
  1. Download into the to the user's Downloads folder the Oracle Instant Client Basic-Lite package and SDK zip files specific to Windows.
  2. Unzip the above two zip files into the user-specified installation directory.
  3. Add an additional path to the User `PATH` Environment Variable.
  4. Create two new User Environment Variables based on the user prompt entries: `OCI_LIB64` and `TNS_NAMES`
  