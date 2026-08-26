package generator

import (
	"context"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeTools struct{ prepared *PreparedTools }

func (f fakeTools) PrepareTools(context.Context, DLLBitness) (*PreparedTools, error) {
	return f.prepared, nil
}

type approvedAuth struct{}

func (approvedAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{ClientID: 1, RequestID: []byte("request"), ChallengeURL: "challenge", Interval: time.Millisecond}, nil
}
func (approvedAuth) Poll(context.Context, uint64, []byte) (AuthPoll, error) {
	return AuthPoll{AccountName: "account", RefreshToken: "secret"}, nil
}

type pendingAuth struct{}

func (pendingAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{ClientID: 1, RequestID: []byte("request"), ChallengeURL: "challenge", Interval: time.Millisecond}, nil
}

func (pendingAuth) Poll(ctx context.Context, _ uint64, _ []byte) (AuthPoll, error) {
	<-ctx.Done()
	return AuthPoll{}, ctx.Err()
}

type capturedEvents struct{ updates []Update }

func (c *capturedEvents) SendEvent(_ string, payload any) {
	c.updates = append(c.updates, payload.(Update))
}

func TestSetupGBEInstallsOnlySuccessfulOpaqueGSEOutput(t *testing.T) {
	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	executable := filepath.Join(gseDir, "generate_emu_config")
	script := "#!/bin/sh\nset -eu\ntest -f refresh_tokens.json\ntest \"$#\" -eq 1\nmkdir -p \"_OUTPUT/$1/steam_settings\"\nprintf '[{},{}]' > \"_OUTPUT/$1/steam_settings/achievements.json\"\nprintf '[{}]' > \"_OUTPUT/$1/steam_settings/stats.json\"\nprintf opaque > \"_OUTPUT/$1/steam_settings/custom.bin\"\n"
	require.NoError(t, os.WriteFile(executable, []byte(script), 0755))
	staging := filepath.Join(root, "staging")
	require.NoError(t, os.Mkdir(staging, 0700))
	stagedDLL := filepath.Join(staging, "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("gbe"), 0644))
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))

	originalConfigPath := backend.ConfigPath
	backend.ConfigPath = filepath.Join(root, "config", "config.json")
	t.Cleanup(func() { backend.ConfigPath = originalConfigPath })
	events := &capturedEvents{}
	service := &Service{
		Config: &config.File{}, Auth: approvedAuth{}, Events: events,
		Tools: fakeTools{prepared: &PreparedTools{GSEExecutable: executable, GBEDLL: stagedDLL, StagingDir: staging}},
	}
	result, err := service.SetupGBE(SetupRequest{AppID: "620", DLLPath: targetDLL})
	require.NoError(t, err)
	require.Equal(t, "620", result.AppID)
	require.FileExists(t, filepath.Join(targetDir, "steam_settings", "custom.bin"))
	require.FileExists(t, targetDLL+".sentinel.bak")
	require.NoFileExists(t, filepath.Join(gseDir, "refresh_tokens.json"))
	require.NoDirExists(t, filepath.Join(gseDir, "_OUTPUT", "620"))
	require.Contains(t, service.ManagedGBESetupAppIDs(), "620")
	for _, update := range events.updates {
		require.NotContains(t, update.Message, "secret")
	}
}

func TestFailedGSELeftoverOutputNeverInstalls(t *testing.T) {
	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	executable := filepath.Join(gseDir, "generate_emu_config")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nmkdir -p \"_OUTPUT/$1/steam_settings\"\nprintf partial > \"_OUTPUT/$1/steam_settings/partial.txt\"\nexit 1\n"), 0755))
	staging := filepath.Join(root, "staging")
	require.NoError(t, os.Mkdir(staging, 0700))
	stagedDLL := filepath.Join(staging, "steam_api64.dll")
	require.NoError(t, os.WriteFile(stagedDLL, []byte("gbe"), 0644))
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))
	service := &Service{
		Config: &config.File{}, Auth: approvedAuth{},
		Tools: fakeTools{prepared: &PreparedTools{GSEExecutable: executable, GBEDLL: stagedDLL, StagingDir: staging}},
	}
	_, err := service.SetupGBE(SetupRequest{AppID: "620", DLLPath: targetDLL})
	require.Error(t, err)
	data, readErr := os.ReadFile(targetDLL)
	require.NoError(t, readErr)
	require.Equal(t, "original", string(data))
	require.NoFileExists(t, targetDLL+".sentinel.bak")
	require.NoDirExists(t, filepath.Join(gseDir, "_OUTPUT", "620"))
	require.NotContains(t, service.ManagedGBESetupAppIDs(), "620")
}

func TestSetupGBETimesOutQRApproval(t *testing.T) {
	originalTimeout := qrApprovalTimeout
	qrApprovalTimeout = time.Millisecond
	t.Cleanup(func() { qrApprovalTimeout = originalTimeout })

	root := t.TempDir()
	gseDir := filepath.Join(root, "gse")
	staging := filepath.Join(root, "staging")
	targetDir := filepath.Join(root, "game")
	require.NoError(t, os.MkdirAll(gseDir, 0755))
	require.NoError(t, os.Mkdir(staging, 0700))
	require.NoError(t, os.Mkdir(targetDir, 0755))
	targetDLL := filepath.Join(targetDir, "steam_api64.dll")
	require.NoError(t, os.WriteFile(targetDLL, []byte("original"), 0644))

	service := &Service{
		Config: &config.File{}, Auth: pendingAuth{},
		Tools: fakeTools{prepared: &PreparedTools{
			GSEExecutable: filepath.Join(gseDir, "generate_emu_config"),
			GBEDLL:        filepath.Join(staging, "steam_api64.dll"),
			StagingDir:    staging,
		}},
	}

	_, err := service.SetupGBE(SetupRequest{AppID: "620", DLLPath: targetDLL})
	require.ErrorIs(t, err, errQRApprovalTimedOut)
	require.FileExists(t, targetDLL)
	require.NoFileExists(t, targetDLL+".sentinel.bak")
}

func TestUndoRemovesStaleManagedEntryWithoutChangingDirectory(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "keep.txt")
	require.NoError(t, os.WriteFile(marker, []byte("keep"), 0644))
	originalConfigPath := backend.ConfigPath
	backend.ConfigPath = filepath.Join(t.TempDir(), "config.json")
	t.Cleanup(func() { backend.ConfigPath = originalConfigPath })
	configuration := &config.File{ManagedGBESetups: []config.ManagedGBESetup{{AppID: "620", Path: root}}}
	service := &Service{Config: configuration}
	require.NoError(t, service.UndoGBESetup("620"))
	require.NotContains(t, service.ManagedGBESetupAppIDs(), "620")
	require.FileExists(t, marker)
}

func TestCleanupRemovesTransientDataAndRetainsCachedTool(t *testing.T) {
	originalGeneratorDir := backend.GeneratorDir
	backend.GeneratorDir = t.TempDir()
	t.Cleanup(func() { backend.GeneratorDir = originalGeneratorDir })
	gseDir := filepath.Join(backend.GeneratorDir, "gse", backend.GSEToolsVersion, "distribution")
	require.NoError(t, os.MkdirAll(filepath.Join(gseDir, "_OUTPUT", "620"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(gseDir, "refresh_tokens.json"), []byte("secret"), 0600))
	tool := filepath.Join(gseDir, "generate_emu_config")
	require.NoError(t, os.WriteFile(tool, []byte("tool"), 0755))
	require.NoError(t, cleanupGeneratorTransientData())
	require.FileExists(t, tool)
	require.NoFileExists(t, filepath.Join(gseDir, "refresh_tokens.json"))
	require.NoDirExists(t, filepath.Join(gseDir, "_OUTPUT"))
}

func TestRunGSETerminatesWhenCancelled(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "generate_emu_config")
	require.NoError(t, os.WriteFile(executable, []byte("#!/bin/sh\nsleep 30\n"), 0755))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	_, err := runGSEGenerator(ctx, executable, "620")
	require.Error(t, err)
	require.Less(t, time.Since(started), 2*time.Second)
	require.Equal(t, 5*time.Minute, generatorTimeout)
}
