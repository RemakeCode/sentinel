package generator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sentinel/backend/config"
	"strings"
	"sync"
	"time"
)

var (
	ErrGBESetupRunning     = errors.New("a GBE setup is already running")
	ErrInvalidSetupRequest = errors.New("invalid GBE setup request")
	errQRApprovalTimedOut  = errors.New("Steam sign-in approval timed out. Start setup again to get a new QR code.")
)

type Phase string

const (
	PhasePreparing     Phase = "preparing"
	PhaseDownloading   Phase = "downloading"
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
	AppID    string `json:"appId"`
	GameName string `json:"gameName"`
	DLLPath  string `json:"dllPath"`
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
	PrepareTools(context.Context, InstallTarget, func(string)) (*PreparedTools, error)
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

	cleanupGeneratorTransientData()
	return nil
}

func (s *Service) SetupGBE(request SetupRequest) (result SetupResult, err error) {
	request.AppID = strings.TrimSpace(request.AppID)
	request.GameName = strings.TrimSpace(request.GameName)
	if request.GameName == "" {
		return result, fmt.Errorf("%w: game name is required", ErrInvalidSetupRequest)
	}
	target, err := ValidateInstallTarget(request.AppID, request.DLLPath)
	if err != nil {
		return result, fmt.Errorf("%w: %v", ErrInvalidSetupRequest, err)
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

	s.emit(Update{Phase: PhasePreparing, Message: "Preparing pinned GSE Fork Tools and gbe_fork DLLs"})
	tools, err := s.Tools.PrepareTools(ctx, target, func(message string) {
		s.emit(Update{Phase: PhaseDownloading, Message: message})
	})
	if err != nil {
		return result, s.reportFailureOrCancellation(request.AppID, err)
	}

	generatorDirectory := filepath.Dir(tools.GeneratorExecutable)
	outputDirectory := filepath.Join(generatorDirectory, gseOutputDirectoryName, target.AppID)
	generatedSettings := filepath.Join(outputDirectory, steamSettingsDirectoryName)
	tokenPath := filepath.Join(generatorDirectory, gseTokenFilename)
	defer s.cleanupOperationTransientData(tools.WorkspaceDir, outputDirectory, tokenPath)

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return result, s.reportFailure(request.AppID, fmt.Errorf("remove stale authentication handoff: %w", err))
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
			return result, s.reportQRApprovalTimeout(request.AppID)
		}

		return result, s.reportFailureOrCancellation(request.AppID, err)
	}

	if authResult.AccountName == "" || authResult.RefreshToken == "" {
		return result, s.reportFailure(request.AppID, errors.New("Steam authentication returned incomplete credentials"))
	}

	if err := writeGSETokenFile(tokenPath, authResult.AccountName, authResult.RefreshToken); err != nil {
		return result, s.reportFailure(request.AppID, fmt.Errorf("create temporary GSE authentication handoff: %w", err))
	}

	s.emit(Update{Phase: PhaseGenerating, Message: "Generating GBE configuration"})
	if err := os.RemoveAll(outputDirectory); err != nil {
		return result, s.reportFailure(request.AppID, fmt.Errorf("clear stale generator output: %w", err))
	}

	diagnostics, err := runGSEGenerator(ctx, tools.GeneratorExecutable, target.AppID)
	if removeErr := os.Remove(tokenPath); removeErr != nil && !os.IsNotExist(removeErr) {
		slog.Warn("remove temporary GSE authentication handoff", "app_id", request.AppID, "error", removeErr)
	}
	if err != nil {
		if sanitized := sanitizeDiagnostics(diagnostics); sanitized != "" {
			slog.Debug("GSE generation diagnostics", "output", sanitized)
		}
		return result, s.reportFailureOrCancellation(request.AppID, fmt.Errorf("GSE generation failed: %w", err))
	}

	if !dirExists(generatedSettings) {
		return result, s.reportFailure(request.AppID, errors.New("GSE exited successfully without an available generated output folder"))
	}

	workspaceSettings := filepath.Join(tools.WorkspaceDir, steamSettingsDirectoryName)
	if err := copyDirectory(generatedSettings, workspaceSettings); err != nil {
		return result, s.reportFailure(request.AppID, fmt.Errorf("copy generated output into operation workspace: %w", err))
	}

	s.emit(Update{Phase: PhaseInstalling, Message: "Backing up existing files and installing GBE setup"})
	if err := InstallGBESetup(target, tools.ReplacementDLL, workspaceSettings); err != nil {
		return result, s.reportFailure(request.AppID, err)
	}

	installDirectory := target.InstallDirectory
	if err := s.Config.SetManagedGBESetup(target.AppID, request.GameName, installDirectory); err != nil {
		// Installation succeeded. Keep the exact backups and report that only the
		// managed index could not be recorded; Undo remains possible manually.
		return result, s.reportFailure(request.AppID, fmt.Errorf("record managed GBE setup: %w", err))
	}
	result = SetupResult{
		AppID: target.AppID, InstallDirectory: installDirectory,
		GSEVersion: gseForkToolsAsset.version, GBEVersion: gbeForkDLLAsset.version,
	}
	s.emit(Update{
		Phase: PhaseCompleted, Message: "GBE setup completed",
		GSEVersion: gseForkToolsAsset.version, GBEVersion: gbeForkDLLAsset.version,
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

type ManagedGBESetupSummary struct {
	AppID string `json:"appId"`
	Name  string `json:"name"`
}

func (s *Service) ManagedGBESetups() []ManagedGBESetupSummary {
	if s.Config == nil {
		return nil
	}

	setups := s.Config.GetManagedGBESetups()
	summaries := make([]ManagedGBESetupSummary, 0, len(setups))
	for _, setup := range setups {
		if setup.AppID != "" && setup.Name != "" && setup.Path != "" {
			summaries = append(summaries, ManagedGBESetupSummary{AppID: setup.AppID, Name: setup.Name})
		}
	}

	return summaries
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
		return s.reportUndoFailure(setup.AppID, err)
	}
	if err := s.Config.RemoveManagedGBESetup(setup.AppID); err != nil {
		return s.reportUndoFailure(setup.AppID, err)
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
			err := errors.New("timed out waiting for active GBE setup to stop")
			slog.Error("GBE generator shutdown timed out", "error", err)
			return err
		}
	}
	cleanupGeneratorTransientData()
	return nil
}

func (s *Service) reportFailure(appID string, err error) error {
	slog.Error("GBE setup failed", "app_id", appID, "error", err)
	s.emit(Update{Phase: PhaseFailed, Message: err.Error()})
	return err
}

func (s *Service) reportQRApprovalTimeout(appID string) error {
	slog.Warn("GBE setup timed out awaiting Steam approval", "app_id", appID)
	s.emit(Update{Phase: PhaseTimedOut, Message: errQRApprovalTimedOut.Error()})
	return errQRApprovalTimedOut
}

func (s *Service) reportFailureOrCancellation(appID string, err error) error {
	if errors.Is(err, context.Canceled) {
		slog.Info("GBE setup cancelled", "app_id", appID)
		s.emit(Update{
			Phase:   PhaseCancelled,
			Message: "Achievement setup was cancelled before installation. The game files were left unchanged.",
		})
		return err
	}
	return s.reportFailure(appID, err)
}

func (s *Service) reportUndoFailure(appID string, err error) error {
	slog.Error("GBE undo failed", "app_id", appID, "error", err)
	return err
}

func (s *Service) cleanupOperationTransientData(workspaceDirectory, outputDirectory, tokenPath string) {
	if err := os.RemoveAll(workspaceDirectory); err != nil {
		slog.Warn("remove GBE operation workspace", "error", err)
	}

	if err := os.RemoveAll(outputDirectory); err != nil {
		slog.Warn("remove GSE generated output", "error", err)
	}

	if err := os.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("remove temporary GSE authentication handoff", "error", err)
	}
}
