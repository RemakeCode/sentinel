package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend"
	"sentinel/backend/config"
	"strings"
	"sync"
	"time"
)

var (
	ErrGBESetupRunning    = errors.New("a GBE setup is already running")
	errQRApprovalTimedOut = errors.New("Steam sign-in approval timed out. Start setup again to get a new QR code.")
)

type Phase string

const (
	PhasePreparing     Phase = "preparing"
	PhaseAwaitingQR    Phase = "awaitingQr"
	PhaseGenerating    Phase = "generating"
	PhaseInstalling    Phase = "installing"
	PhaseCompleted     Phase = "completed"
	PhaseCancelled     Phase = "cancelled"
	PhaseTimedOut      Phase = "timedOut"
	PhaseFailed        Phase = "failed"
	PhaseUndoCompleted Phase = "undoCompleted"
)

type Update struct {
	Phase        Phase  `json:"phase"`
	Message      string `json:"message,omitempty"`
	ChallengeURL string `json:"challengeUrl,omitempty"`
	GSEVersion   string `json:"gseVersion,omitempty"`
	GBEVersion   string `json:"gbeVersion,omitempty"`
}

type SetupRequest struct {
	AppID                  string `json:"appId"`
	DLLPath                string `json:"dllPath"`
	ConfirmExistingBackups bool   `json:"confirmExistingBackups"`
}

type SetupResult struct {
	AppID            string `json:"appId"`
	InstallDirectory string `json:"path"`
	GSEVersion       string `json:"gseVersion"`
	GBEVersion       string `json:"gbeVersion"`
}

type EventSink interface {
	SendEvent(messageType string, payload any)
}

type ToolPreparer interface {
	PrepareTools(context.Context, DLLBitness) (*PreparedTools, error)
}

type Service struct {
	Config *config.File
	Tools  ToolPreparer
	Auth   AuthTransport
	Events EventSink

	mu           sync.Mutex
	activeCancel context.CancelFunc
	setupDone    chan struct{}
}

//wails:internal
func (s *Service) Start(_ context.Context) error {
	if s.Config == nil {
		return errors.New("config not injected into generator service")
	}

	if s.Tools == nil {
		s.Tools = NewToolManager()
	}

	if s.Auth == nil {
		s.Auth = NewHTTPAuthTransport(nil)
	}

	return cleanupGeneratorTransientData()
}

func (s *Service) InspectGBEBackups(appID, dllPath string) (BackupStatus, error) {
	target, err := ValidateInstallTarget(appID, dllPath)
	if err != nil {
		return BackupStatus{}, err
	}

	return InspectTargetBackups(target), nil
}

func (s *Service) SetupGBE(request SetupRequest) (result SetupResult, err error) {
	target, err := ValidateInstallTarget(request.AppID, request.DLLPath)
	if err != nil {
		return result, err
	}

	preflight := InspectTargetBackups(target)
	if (preflight.HasDLLBackup || preflight.HasSettingsBackup) && !request.ConfirmExistingBackups {
		return result, errors.New("existing Sentinel backups require confirmation")
	}

	s.mu.Lock()
	if s.activeCancel != nil {
		s.mu.Unlock()
		return result, ErrGBESetupRunning
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.activeCancel = cancel
	s.setupDone = done
	s.mu.Unlock()
	defer func() {
		cancel()
		close(done)
		s.mu.Lock()
		s.activeCancel = nil
		s.setupDone = nil
		s.mu.Unlock()
	}()

	s.emit(Update{Phase: PhasePreparing, Message: "Preparing pinned GSE Tools and GBE assets"})
	tools, err := s.Tools.PrepareTools(ctx, target.Bitness)
	if err != nil {
		return result, s.reportFailureOrCancellation(err)
	}

	gseDir := filepath.Dir(tools.GSEExecutable)
	outputRoot := filepath.Join(gseDir, "_OUTPUT", target.AppID)
	tokenPath := filepath.Join(gseDir, "refresh_tokens.json")
	defer os.RemoveAll(tools.StagingDir)
	defer os.RemoveAll(outputRoot)
	defer os.Remove(tokenPath)

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return result, s.reportFailure(fmt.Errorf("remove stale authentication handoff: %w", err))
	}

	authResult, err := Authenticate(ctx, s.Auth, func(challenge AuthChallenge) {
		s.emit(Update{
			Phase:        PhaseAwaitingQR,
			Message:      "Scan this QR code, then approve the sign-in in the Steam Mobile app.",
			ChallengeURL: challenge.ChallengeURL,
		})
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return result, s.reportQRApprovalTimeout()
		}

		return result, s.reportFailureOrCancellation(err)
	}

	if authResult.AccountName == "" || authResult.RefreshToken == "" {
		return result, s.reportFailure(errors.New("Steam authentication returned incomplete credentials"))
	}

	if err := writeGSETokenFile(tokenPath, authResult.AccountName, authResult.RefreshToken); err != nil {
		return result, s.reportFailure(fmt.Errorf("create temporary GSE authentication handoff: %w", err))
	}

	// Do not retain secrets in the operation result after materializing GSE's
	// restricted handoff file.
	authResult.AccountName = ""
	authResult.RefreshToken = ""

	s.emit(Update{Phase: PhaseGenerating, Message: "Generating GBE configuration"})
	if err := os.RemoveAll(outputRoot); err != nil {
		return result, s.reportFailure(fmt.Errorf("clear stale generator output: %w", err))
	}

	diagnostics, err := runGSEGenerator(ctx, tools.GSEExecutable, target.AppID)
	_ = os.Remove(tokenPath)
	if err != nil {
		if sanitized := sanitizeDiagnostics(diagnostics); sanitized != "" {
			slog.Debug("GSE generation diagnostics", "output", sanitized)
		}
		return result, s.reportFailureOrCancellation(fmt.Errorf("GSE generation failed: %w", err))
	}

	generatedSettings := filepath.Join(outputRoot, "steam_settings")
	if !dirExists(generatedSettings) {
		return result, s.reportFailure(errors.New("GSE exited successfully without an available generated output folder"))
	}

	s.emit(Update{Phase: PhaseInstalling, Message: "Backing up existing files and installing GBE setup"})
	if err := InstallGBESetup(target, tools.GBEDLL, generatedSettings); err != nil {
		return result, s.reportFailure(err)
	}

	installDirectory := filepath.Dir(target.DLLPath)
	if err := s.Config.SetManagedGBESetup(target.AppID, installDirectory); err != nil {
		// Installation succeeded. Keep the exact backups and report that only the
		// managed index could not be recorded; Undo remains possible manually.
		return result, s.reportFailure(fmt.Errorf("record managed GBE setup: %w", err))
	}
	result = SetupResult{
		AppID: target.AppID, InstallDirectory: installDirectory,
		GSEVersion: backend.GSEToolsVersion, GBEVersion: backend.GBEForkVersion,
	}
	s.emit(Update{
		Phase: PhaseCompleted, Message: "GBE setup completed",
		GSEVersion: backend.GSEToolsVersion, GBEVersion: backend.GBEForkVersion,
	})
	return result, nil
}

func (s *Service) CancelGBESetup() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeCancel == nil {
		return false
	}
	s.activeCancel()
	return true
}

func (s *Service) ManagedGBESetupAppIDs() []string {
	if s.Config == nil {
		return nil
	}

	setups := s.Config.GetManagedGBESetups()
	appIDs := make([]string, 0, len(setups))
	for _, setup := range setups {
		if setup.AppID != "" && setup.Path != "" {
			appIDs = append(appIDs, setup.AppID)
		}
	}

	return appIDs
}

func (s *Service) UndoGBESetup(appID string) error {
	if s.Config == nil {
		return errors.New("config not injected into generator service")
	}
	setup, ok := s.Config.GetManagedGBESetup(strings.TrimSpace(appID))
	if !ok {
		return errors.New("game is not managed by Sentinel")
	}
	restored, err := RestoreSentinelBackups(setup.Path)
	if err != nil {
		return err
	}
	if err := s.Config.RemoveManagedGBESetup(setup.AppID); err != nil {
		return err
	}
	message := "Achievement setup was undone"
	if !restored {
		message = "Managed GBE setup was no longer present"
	}
	s.emit(Update{Phase: PhaseUndoCompleted, Message: message})
	return nil
}

//wails:internal
func (s *Service) ServiceShutdown() error {
	s.mu.Lock()
	cancel := s.activeCancel
	done := s.setupDone
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			return errors.New("timed out waiting for active GBE setup to stop")
		}
	}
	return cleanupGeneratorTransientData()
}

func (s *Service) reportFailure(err error) error {
	s.emit(Update{Phase: PhaseFailed, Message: err.Error()})
	return err
}

func (s *Service) reportQRApprovalTimeout() error {
	s.emit(Update{Phase: PhaseTimedOut, Message: errQRApprovalTimedOut.Error()})
	return errQRApprovalTimedOut
}

func (s *Service) reportFailureOrCancellation(err error) error {
	if errors.Is(err, context.Canceled) {
		s.emit(Update{
			Phase:   PhaseCancelled,
			Message: "Achievement setup was cancelled before installation. The game files were left unchanged.",
		})
		return err
	}
	return s.reportFailure(err)
}
