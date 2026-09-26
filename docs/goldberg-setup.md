# Goldberg Steam Emulator + Sentinel Setup Guide (Wine/Proton)

This guide covers setting up the Goldberg Steam Emulator Fork for achievements with Sentinel on Linux, focused on games running under Wine or Proton. You can let Sentinel set up a game automatically, or install the emulator files yourself.

## Automatic setup with Sentinel

Sentinel can generate the game configuration and install the required Goldberg files for you.

1. In Sentinel, open **Settings → Achievement Setup** and search for the game.
2. Select the game's existing `steam_api64.dll` or `steam_api.dll` from its installation folder. The DLL may be in a subfolder; Unreal Engine games, for example, often keep it under `Game/Binaries/Win64/`.
3. Continue to start setup. Sentinel displays a Steam sign-in QR code. Scan it with the Steam Mobile app and approve the sign-in.
4. Leave Sentinel open while it generates the configuration and installs the setup. Sentinel saves a `.sentinel.bak` copy of the DLL it replaces and backs up any existing `steam_settings` folder as `steam_settings.sentinel.bak`. You can restore them later from **Configured Games → Undo**.

If a sibling `coldclient/` directory contains the matching `steamclient.dll` (32-bit) or `steamclient64.dll` (64-bit), Sentinel installs the GBE SteamClient DLL and generated `steam_settings` there. Otherwise, it installs them beside the selected Steam API DLL.

The QR approval is required to retrieve the game's Steam data. Sentinel uses Steam Mobile approval instead of asking you to type your Steam password or a Steam Guard code into Sentinel.

## Manual setup

Use these steps if you prefer to install the emulator files yourself or are setting up a game outside Sentinel's wizard.

### 1. Install Goldberg Emulator

Download the latest Windows release (`emu-win-release.7z`) from [gbe_fork releases](https://github.com/Detanup01/gbe_fork/releases). Extract the archive (it creates a `release/` folder) and locate the DLL for the game's bitness:

| Game bitness | DLL path in the archive |
|---|---|
| 64-bit (most modern games) | `release/experimental/x64/steam_api64.dll` |
| 32-bit | `release/experimental/x86/steam_api.dll` |

Use the DLLs from `release/experimental/` for achievement support. Replace the game's original `steam_api64.dll` or `steam_api.dll` wherever it is found in the game installation directory.

**Back up the original DLL and any existing `steam_settings` folder before replacing files.**

### 2. Generate `steam_settings` with gse_fork_tools

Download the [Linux generator archive](https://github.com/RemakeCode/gse_fork_tools/releases/download/2026_09_25/gen_emu_cfg-Linux-Release.tar.bz2), extract it, and run the generator with the game's Steam AppID:

```bash
./generate_emu_config <AppID>
```

The standalone tool prompts for Steam credentials. Steam may also ask you to complete its normal Steam Guard verification. You can automate the credential prompt with a `my_login.txt` file beside the tool, with your username on the first line and password on the second. Keep this file private and remove it when you no longer need it.

The tool writes the generated game files under `_OUTPUT/<AppID>/`. Copy the generated `steam_settings/` folder into the game's installation directory, beside the Goldberg DLL.
