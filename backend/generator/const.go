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
		version:            "2026_09_25",
		assetName:          "gen_emu_cfg-Linux-Release.tar.bz2",
		releaseURL:         "https://github.com/RemakeCode/gse_fork_tools/releases/download/2026_09_25/gen_emu_cfg-Linux-Release.tar.bz2",
		sha256:             "195d34caf59549e58080d9229f1a67b05a63539507f8fd08514ece09500e6ec2",
	}

	gbeForkDLLAsset = pinnedArchive{
		cacheDirectoryName: "gbe-fork",
		version:            "release-2026_09_16_2",
		assetName:          "emu-win-release-vs22.7z",
		releaseURL:         "https://github.com/Detanup01/gbe_fork/releases/download/release-2026_09_16_2/emu-win-release-vs22.7z",
		sha256:             "d311deadc2a8a8aed620fe66976646059388123587aa22d408f723c592fc9688",
	}

	gbeForkDLLMembers = []string{
		"release/experimental/x86/steam_api.dll",
		"release/experimental/x64/steam_api64.dll",
		"release/steamclient_experimental/steamclient.dll",
		"release/steamclient_experimental/steamclient64.dll",
	}
)
