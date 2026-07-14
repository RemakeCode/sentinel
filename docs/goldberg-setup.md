# Goldberg Steam Emulator + Sentinel Setup Guide (Wine/Proton)

This guide covers setting up the [Goldberg Steam Emulator Fork](https://github.com/Detanup01/gbe_fork) for achievements with [Sentinel](https://github.com/RemakeCode/sentinel) on Linux, focused on games running under Wine or Proton.

---

## 1. Install Goldberg Emulator

Download the latest Windows release (`emu-win-release.7z`) from [gbe_fork releases](https://github.com/Detanup01/gbe_fork/releases). Extract the archive (it creates a `release/` folder) and locate the DLLs inside it:

| Game bitness | DLL path in the archive |
|---|---|
| 64-bit (most modern games) | `release/experimental/x64/steam_api64.dll` |
| 32-bit | `release/experimental/x86/steam_api.dll` |

Use the DLLs from the `release/experimental/` folder — these are the builds that work with achievements.

Replace the original `steam_api64.dll` or `steam_api.dll` wherever it is found in the game's install directory. Note that the DLL may not always be in the same folder as the game's `.exe` — for example, Unreal Engine games often place it in `Game/Binaries/Win64/`.

**Before replacing any files, backup the original `steam_api64.dll` and the `steam_settings` folder if it already exists.**

---

## 2. Generate steam_settings with gse_fork_tools

The fastest way to generate a complete `steam_settings` folder (including achievements, DLCs, configs) is using [gse_fork_tools](https://github.com/alex47exe/gse_fork_tools).

- Download the **Linux** binary: [`gen_emu_cfg-Linux-Release.tar.bz2`](https://github.com/alex47exe/gse_fork_tools/releases/)
- Extract and run the tool with a Steam account login:
```
./generate_emu_config <AppID>
```

You will be prompted for your Steam username and password on first run (or you can create a `my_login.txt` file with your username on line 1 and password on line 2 to automate it).

The tool outputs a complete `steam_settings/` directory into `output/<AppId>/`.

**Copy the generated `steam_settings/` folder into the game's install directory** (next to `steam_api64.dll`).

---

## 3. Achievement Save Paths

Goldberg writes achievement progress to JSON files inside a prefix's virtual `AppData`. Sentinel watches these paths automatically.

### Heroic Games Launcher

```
~/Games/Heroic/Prefixes/<Game Name>/pfx/drive_c/users/steamuser/AppData/Roaming/GSE Saves/
```

Alternative path (depends on Goldberg config):

```
.../Goldberg SteamEmu Saves/
```

### Steam + Proton

```
~/.local/share/Steam/steamapps/compatdata/<AppID>/pfx/drive_c/users/steamuser/AppData/Roaming/GSE Saves/
```

**Note:** The emulator auto-generates the `achievements.json` progress file itself — either on the game's first run or when the first achievement is unlocked. It lands at `AppData/Roaming/<AppID>/achievements.json` inside the prefix. You do **not** need to manually create or add an `achievements.json`; just make sure the game launches with the Goldberg DLL in place and Sentinel will pick it up once it appears.

---

## 4. Known Issue: Steam AppID Override (#549)

**Bug:** Since commit [`e0a4dd8`](https://github.com/Detanup01/gbe_fork/commit/e0a4dd8846ea970d0312ef778fb7082ea0657f81) (March 2026), environment variables (`SteamAppId`, `SteamGameId`) take priority over `steam_settings/steam_appid.txt`. When Steam launches a Proton game, it sets these env vars, which can force Goldberg to use a different AppID than the one you configured.

**Reference:** [Issue #549](https://github.com/Detanup01/gbe_fork/issues/549)

**Workaround:** This only affects games launched directly through Steam's Proton runtime. Use **Heroic Games Launcher**, **Fargus**, or **Lutris** instead — they manage their own Wine/Proton prefixes without setting Steam environment variables, so `steam_settings/steam_appid.txt` works correctly. You can then add the game to Steam from the launcher if needed.

---

## 5. Sentinel Configuration

1. Launch Sentinel
2. Go to **Settings → Prefix Paths** and add the full path to your Wine/Proton prefix (e.g. the `pfx` folder from step 3)

The default emulator paths, notifications, and data source are already set correctly. Sentinel will automatically detect games, scan for `achievements.json` files, and send desktop notifications when achievements are unlocked.
