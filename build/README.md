# Build Directory

The build directory is used to house all the build files and assets for your application.

The structure is:

* bin - Output directory
* darwin - macOS specific files
* windows - Windows specific files
* linux - Linux specific files

## Mac

The `darwin` directory holds files specific to Mac builds.
These may be customised and used as part of the build. To return these files to the default state, simply delete them
and build with `task build`.

The directory contains the following files:

- `Info.plist` - the main plist file used for Mac builds. It is used when building using `task build`.
- `Info.dev.plist` - same as the main plist file but used when building using `task dev`.

## Windows

The `windows` directory contains the manifest and rc files used when building with `task build`.
These may be customised for your application. To return these files to the default state, simply delete them and
build with `task build`.

- `icon.ico` - The icon used for the application. This is used when building with `task build`. If you wish to
  use a different icon, simply replace this file with your own. If it is missing, a new `icon.ico` file
  will be created using the `appicon.png` file in the build directory.
- `installer/*` - The files used to create the Windows installer. These are used when building with `task build`.
- `info.json` - Application details used for Windows builds. The data here will be used by the Windows installer,
  as well as the application itself (right click the exe -> properties -> details)
- `wails.exe.manifest` - The main application manifest file.

## Linux

The `linux` directory contains files specific to Linux builds.
These may be customised for your application. To return these files to the default state, simply delete them and
build with `task build`.

## Build Commands

This project uses Wails 3 with Taskfile for build automation:

- `task dev` - Start development server with hot reload
- `task build` - Build production binary
- `task clean` - Clean build artifacts
- `task generate:bindings` - Generate TypeScript bindings from Go code

See the root `Taskfile.yml` for all available tasks.
