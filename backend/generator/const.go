package generator

import "time"

type pinnedArchive struct {
	cacheDirectoryName string
	version            string
	assetName          string
	releaseURL         string
	sha256             string
}

const (
	steamAuthenticationEndpoint = "https://api.steampowered.com/IAuthenticationService/"

	steamSettingsDirectoryName    = "steam_settings"
	sentinelBackupSuffix          = ".sentinel.bak"
	sentinelTemporaryBackupSuffix = sentinelBackupSuffix + ".temp"

	gseForkToolsExecutablePath = "generate_emu_config/generate_emu_config"
	gseOutputDirectoryName     = "_OUTPUT"
	gseTokenFilename           = "refresh_tokens.json"
	tempDirName                = "temp"
	generatorTimeout           = 5 * time.Minute
)

var (
	// qrApprovalTimeout is mutable only to keep timeout tests fast.
	qrApprovalTimeout = 5 * time.Minute

	gseForkToolsAsset = pinnedArchive{
		cacheDirectoryName: "gse-fork-tools",
		version:            "2026_02_16",
		assetName:          "gen_emu_cfg-Linux-Release.tar.bz2",
		releaseURL:         "https://github.com/alex47exe/gse_fork_tools/releases/download/2026_02_16/gen_emu_cfg-Linux-Release.tar.bz2",
		sha256:             "6a70b7af7db253d80a1133c4a8e27f259e7d3201906cd1f76346f3f12a732ab1",
	}

	gbeForkDLLAsset = pinnedArchive{
		cacheDirectoryName: "gbe-fork",
		version:            "release-2026_07_19",
		assetName:          "emu-win-release.7z",
		releaseURL:         "https://github.com/Detanup01/gbe_fork/releases/download/release-2026_07_19/emu-win-release.7z",
		sha256:             "3ba855ef962205136a54fb32519a46362e0cc5b42fc2bb3667e4d21307d972e5",
	}

	gbeForkDLLMembers = []string{
		"release/experimental/x86/steam_api.dll",
		"release/experimental/x64/steam_api64.dll",
		"release/steamclient_experimental/steamclient.dll",
		"release/steamclient_experimental/steamclient64.dll",
	}
)
