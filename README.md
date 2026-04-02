# Sentinel

![Banner](.github/assets/banner.png)

**An achievement watcher for Steam emulator games on Linux**

Sentinel monitors your Steam emulator save files and sends real-time desktop notifications when achievements are unlocked or progress is updated. It also serves as a library viewer with completion stats, global achievement percentages, and more.

---

## Features

- Real-time desktop notifications via `notify-send`
- Progress tracking for multi-step achievements
- Game library with completion stats and sorting
- Global achievement percentages from [Steam API](https://steamcommunity.com/dev)
- Custom notification sounds (10 platform-themed options)
- System tray support (runs in background)
- Supports multiple Steam emulators ([Goldberg](https://github.com/Detanup01/gbe_fork), [CreamAPI](https://cs.rin.ru/forum/viewtopic.php?t=70576), etc.)
- Choice of [Steam Web API key](https://steamcommunity.com/dev/apikey) or free external data source ([SteamHunters](https://steamhunters.com))

## Screenshots

![Dashboard](.github/assets/dashboard.png)
*Game library with completion progress*

![Game Details](.github/assets/game-details.png)
*Achievement list with global percentages*

![Notification](.github/assets/notification.png)
*Desktop notification with achievement icon*

![Settings](.github/assets/settings.png)
*Configuration panel*

## Installation

### Linux Packages

Download the latest release from [GitHub Releases](https://github.com/RemakeCode/sentinel/releases).

**Debian/Ubuntu (.deb):**
```bash
sudo dpkg -i sentinel_<version>_amd64.deb
```

**Fedora/RHEL (.rpm):**
```bash
sudo dnf install sentinel-<version>.x86_64.rpm
```

**Arch Linux:**
```bash
sudo pacman -U sentinel-<version>-x86_64.pkg.tar.zst
```

### System Requirements

- **GTK 4** ([libgtk-4-1](https://www.gtk.org/))
- **WebKitGTK 6.0** ([libwebkitgtk-6.0-4](https://webkitgtk.org/))
- **libnotify** ([libnotify-bin](https://gitlab.gnome.org/GNOME/libnotify))

### Build from Source

**Prerequisites:**
- [Go 1.26+](https://go.dev/dl/)
- [Node.js 24+](https://nodejs.org/)
- [Wails v3](https://wails.io/)
- GTK4 development libraries (see above)

```bash
# Install Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

# Clone and build
git clone https://github.com/RemakeCode/sentinel.git
cd sentinel
task build
```

## Quick Start

1. **Configure Prefix Paths** — Add your Wine/Proton prefix directories where emulated games are installed
2. **Configure Emulator Paths** — Add paths to emulator save directories (default: `AppData/Roaming/GSE Saves`)
3. **Choose Data Source** — Use a [Steam API key](https://steamcommunity.com/dev/apikey) for faster data, or the free external source

Sentinel will automatically scan for games and watch for achievement changes.

## How It Works

```
Wine/Proton Prefixes ──┐
                       ├──→ Sentinel discovers games ──→ Fetches metadata from Steam
Emulator Save Paths ───┘                                         │
                                                                 ↓
                                                    Watches achievements.json ──→ Desktop notification
```

1. **Discovery** — Sentinel scans your prefix directories for emulated games
2. **Metadata** — Fetches game info and achievement data from Steam (cached locally)
3. **Watching** — Monitors `achievements.json` files for changes using `fsnotify`
4. **Notifications** — Sends desktop notifications with icons and custom sounds

## Configuration

Config file location: `~/.cache/sentinel/config.json`

## FAQ

### What emulators are supported?
Any emulator that writes `achievements.json` files in a `GSE Saves` directory structure. This includes [Goldberg Steam Emulator](https://github.com/Detanup01/gbe_fork), CreamAPI, and others.

### Where are my emulator save files?
By default, most emulators use `AppData/Roaming/GSE Saves` inside the Wine/Proton prefix. You can configure custom paths in Settings.

### Do I need a Steam API key?
No. Sentinel defaults to using [SteamHunters](https://steamhunters.com) and Steam Community pages as a free data source. A [Steam Web API key](https://steamcommunity.com/dev/apikey) is optional and provides faster, more reliable data.

### Why aren't notifications showing?
- Ensure `libnotify-bin` is installed: `sudo apt install libnotify-bin`
- Check that your desktop environment supports D-Bus notifications
- Verify notification paths are enabled in Settings

### Can I use this on Windows or macOS?
Sentinel is Linux-first. Build targets exist for other platforms, but notifications and system tray are Linux-specific. Contributions for cross-platform support are welcome.

### How do I add a new game after setup?
Sentinel automatically rescans prefix directories every 5 seconds. New games appear in the library automatically.

## Acknowledgments

- [Wails v3](https://wails.io/) — Desktop app framework
- [React](https://react.dev/) — Frontend UI library
- [fsnotify](https://github.com/fsnotify/fsnotify) — File system watcher
- [Goldberg Emulator](https://github.com/Detanup01/gbe_fork) — Inspiration and compatibility

## License

[MIT](LICENSE)
