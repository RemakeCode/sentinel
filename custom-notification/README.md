# Qt Widgets custom notification

Qt achievement popups for Sentinel's desktop **Custom** notification option on layer-shell Wayland desktops. GNOME Wayland uses the [GNOME extension](../gnome-extension/README.md). Native notifications are the default; Decky uses Steam toasts.

## Build and development

Requires CMake and Qt 6 Widgets, plus Qt Wayland client support and LayerShellQt on Linux. Arch Linux packages: `cmake`, `qt6-base`, `layer-shell-qt`. Current Arch Qt 6.12 includes Wayland client support in `qt6-base`.

Fedora 44+ provides Wayland client support in `qt6-qtbase-gui` (Qt 6.10 or newer), so the RPM does not require `qt6-qtwayland`. Debian 13 still requires `qt6-wayland`.

```sh
task common:build:custom-notification
task dev
```

Native desktop builds compile both notification renderers before generating bindings and compiling Go. If CMake cannot find Qt, set `CMAKE_PREFIX_PATH`, for example `CMAKE_PREFIX_PATH=/opt/homebrew/opt/qt task dev`.

CMake output goes in `custom-notification/build-<os>-<arch>-dev/` or `build-<os>-<arch>-production/`. The executable is copied to `backend/notifier/out/` for embedding in Go; generated files are ignored by Git. Before running desktop Go commands or binding generation directly, build both bundles with `task common:build:custom-notification` and `task common:build:gnome-extension`. Decky excludes both bundles.

Cross-builds require a helper compiled for the target OS and architecture. DEB, RPM, and Arch package metadata declares the Qt runtime dependencies; clean-host runtime validation and Docker/Flatpak toolchains remain unfinished.

Standalone CMake builds remain available:

```sh
cmake -S custom-notification -B custom-notification/build -DCMAKE_PREFIX_PATH=/path/to/Qt
cmake --build custom-notification/build
```

CMake exports `compile_commands.json`; `.clangd` points at `build`. The macOS build uses an ordinary topmost window and does not validate Linux fullscreen stacking, focus, screen placement, or input transparency.

Development builds (`task dev` or `task common:build:custom-notification DEV=true`) reload source `custom-notification.qss` when available, including atomic editor saves. Standalone CMake builds can enable this with `-DSENTINEL_QSS_HOT_RELOAD=ON`. Production builds load embedded QSS once and exclude the source path and reload machinery. C++ and embedded resource changes require rebuilding. All content and duration come from JSON; there are no preview flags or built-in sample notifications.

## Embedded helper at runtime

On Qt desktops, selecting **Settings → Notifications → Appearance → Custom**, or starting Sentinel with Custom saved, prepares the embedded helper at:

```text
<backend.DataDir>/bin/sentinel-custom-notification
```

Preparation reuses identical bytes, atomically replaces changed bytes, and restores a missing helper with `0700` permissions. Notifications launch this copy without checking its contents. If it is deleted while Sentinel is running, restart the app to restore it.

To use an external development build, set an absolute override:

```sh
export SENTINEL_CUSTOM_NOTIFICATION_EXECUTABLE=/absolute/path/to/sentinel-custom-notification
```

The QSS and fallback artwork are embedded inside the Qt executable. Qt Widgets/Wayland libraries and the LayerShellQt integration plugin must still be available at runtime; embedding does not bundle these shared dependencies.

Go passes JSON on stdin, plays the selected sound after successful process start, and waits in the background to log exit errors. Qt closes itself after the supplied duration. There is no readiness handshake or startup timeout. A later Qt failure may occur after sound playback and does not trigger native fallback. The existing queue delay applies to every renderer.

## Manual notifications

Use the same JSON input as Go delivery, without command-line flags. Replace the executable path below with the standalone build or extracted helper:

```sh
# Normal Settings/SSE test content
/absolute/path/to/sentinel-custom-notification <<'JSON'
{"title":"Test Notification","description":"For those who come after","gameName":"Sentinel","iconPath":"","progress":0,"maxProgress":0,"isProgress":false,"isRare":false,"durationMs":7000}
JSON

# Progress Settings/SSE test content
/absolute/path/to/sentinel-custom-notification <<'JSON'
{"title":"For those who come after","description":"For those who come after","gameName":"Sentinel","iconPath":"","progress":7,"maxProgress":10,"isProgress":true,"isRare":false,"durationMs":4900}
JSON
```

Go supplies typed fields, the existing absolute artwork path, and earned/progress durations. Qt trusts the payload and displays strings as plain text. Set a longer `durationMs` when editing styling. Missing artwork uses the embedded icon. Progress shows exact counts above a clamped bar without percentage text; nonpositive maximums render an empty bar. Earned notifications omit progress widgets. Rare earned notifications use `isRare: true`; Go suppresses rarity for progress.

Linux compositor acceptance and packaged runtime measurements remain pending in the OpenSpec task list.
