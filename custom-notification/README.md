# Qt Widgets custom notification

Companion renderer for the Wails desktop app. Native notifications remain the default; Decky keeps SSE/Steam toasts. Linux Qt rendering targets layer-shell Wayland compositors. GNOME Wayland uses the bundled extension described in `../gnome-extension/README.md`; Linux runtime dependency packaging remains deferred.

## Build and development

Install CMake and Qt 6 Widgets; Linux also needs Qt Wayland and LayerShellQt. On Arch Linux the development packages are `cmake`, `qt6-base`, `qt6-wayland`, and `layer-shell-qt`.

```sh
task common:build:custom-notification
task dev
```

The Darwin and Linux native build tasks compile the helper and GNOME extension, then build the frontend (including bindings), then compile Go. The shared binding task has no helper dependency. CMake builds incrementally. `task dev` watches C++, QSS, resource files, and CMakeLists changes and embeds the updated executable when Go rebuilds. Set CMake's standard `CMAKE_PREFIX_PATH` environment variable if it cannot locate Qt, for example `CMAKE_PREFIX_PATH=/opt/homebrew/opt/qt task dev`.

Generated CMake files live in `custom-notification/build-<os>-<arch>/`; the executable is copied to `backend/notifier/out/sentinel-custom-notification-<os>-<arch>` for `//go:embed`. Both directories are ignored by Git. The tasks use the existing `ARCH` value; C++ OS conditionals select platform behavior, while CMake selects the compiled CPU architecture. Build the helper and run `task common:build:gnome-extension` before invoking `go build`, `go test`, or `wails3 generate bindings` directly on desktop packages: binding generation loads Go packages and therefore needs both generated embed outputs. Decky builds exclude the helper embed.

Cross-compilation still needs a Qt helper compiled for the target OS/architecture. The new task covers native builds; Docker/Flatpak Qt toolchains and runtime bundles remain deferred. A helper for another platform cannot substitute for the matching target binary.

Standalone CMake builds remain available:

```sh
cmake -S custom-notification -B custom-notification/build -DCMAKE_PREFIX_PATH=/path/to/Qt
cmake --build custom-notification/build
```

CMake exports `compile_commands.json`; `.clangd` points at `build`. The macOS build uses an ordinary topmost window and does not validate Linux fullscreen stacking, focus, screen placement, or input transparency.

The helper reloads source `custom-notification.qss` when available, including atomic editor saves. Otherwise it uses its embedded stylesheet. C++ and embedded resource changes require rebuilding. All content and duration come from JSON; there are no preview flags or built-in sample notifications.

## Embedded helper at runtime

Select **Settings → Notifications → Appearance → Custom**. On the next custom notification, Sentinel extracts its embedded helper into:

```text
<backend.DataDir>/bin/sentinel-custom-notification
```

This is persistent app data, not the cache directory. The helper has `0700` permissions. Sentinel reuses an identical copy, atomically replaces a changed copy after an app update, and recreates it if removed. It selects the embedded executable matching the running OS and architecture.

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
