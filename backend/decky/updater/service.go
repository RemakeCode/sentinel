package updater

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"sentinel/backend"
	"sentinel/backend/notifier"

	"golang.org/x/mod/semver"
)

var ErrUpdateBusy = errors.New("an update operation is already running")

type Service struct {
	Notifier       *notifier.Service
	ctx            context.Context
	mu             sync.Mutex
	operation      sync.Mutex
	state          Snapshot
	client         *http.Client
	manifestURL    string
	currentVersion string
	arch           string
	directory      string
}

// NextUpdate describes the compatible payload selected during discovery.
type NextUpdate struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

type PreparedUpdate struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

type Snapshot struct {
	Phase      string          `json:"phase"`
	Release    *NextUpdate     `json:"release,omitempty"`
	Prepared   *PreparedUpdate `json:"prepared,omitempty"`
	Downloaded int64           `json:"downloaded"`
	Total      int64           `json:"total"`
	Error      string          `json:"error,omitempty"`
}

type UpdateManifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Version       string `json:"version"`
	Channel       string `json:"channel"`
	Notes         string `json:"notes"`
	Artifacts     []struct {
		URL        string `json:"url"`
		Platform   string `json:"platform"`
		Arch       string `json:"arch"`
		Filetype   string `json:"filetype"`
		Size       int64  `json:"size"`
		DigestAlgo string `json:"digestAlgo"`
		Digest     string `json:"digest"`
	} `json:"artifacts"`
}

// Start initializes shared update discovery and preparation without modifying installed files.
//
//wails:internal
func (s *Service) Start(ctx context.Context) error {
	s.ctx = ctx

	if s.client == nil {
		s.client = &http.Client{Timeout: 10 * time.Minute}
	}

	if s.manifestURL == "" {
		s.manifestURL = "https://github.com/RemakeCode/sentinel/releases/latest/download/update-decky.json"
	}

	if s.currentVersion == "" {
		s.currentVersion = backend.Version
	}

	if s.arch == "" {
		s.arch = runtime.GOARCH
	}

	if s.directory == "" {
		if directory := os.Getenv("DECKY_PLUGIN_RUNTIME_DIR"); filepath.IsAbs(directory) {
			s.directory = filepath.Join(directory, "updates")
		}
	}

	s.state.Phase = "idle"

	// Only completed old releases and abandoned temporary downloads are removed.
	// Newer staged ZIPs may belong to a pending Decky confirmation and are retained.
	s.cleanup()

	return nil
}

//wails:internal
func (s *Service) UpdateState() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.state
	if state.Release != nil {
		release := *state.Release
		state.Release = &release
	}

	if state.Prepared != nil {
		prepared := *state.Prepared
		state.Prepared = &prepared
	}

	return state
}

//wails:internal
func (s *Service) CheckUpdate(ctx context.Context) (Snapshot, error) {
	if !s.operation.TryLock() {
		return s.UpdateState(), ErrUpdateBusy
	}
	defer s.operation.Unlock()

	s.mu.Lock()
	s.state.Phase, s.state.Error = "checking", ""
	s.mu.Unlock()

	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	release, err := s.discover(checkCtx)
	if err != nil {
		return s.failed(err)
	}

	s.mu.Lock()
	s.state = Snapshot{Phase: "current", Release: release}
	if release != nil {
		s.state.Phase = "available"
	}
	s.mu.Unlock()

	return s.UpdateState(), nil
}

//wails:internal
func (s *Service) PrepareUpdate(ctx context.Context, version string) (Snapshot, error) {
	if !s.operation.TryLock() {
		return s.UpdateState(), ErrUpdateBusy
	}
	defer s.operation.Unlock()

	state := s.UpdateState()
	if state.Release == nil || state.Release.Version != version {
		return state, errors.New("check for updates again before preparing this release")
	}

	release := *state.Release

	s.mu.Lock()
	s.state.Phase, s.state.Error, s.state.Downloaded, s.state.Total, s.state.Prepared = "downloading", "", 0, release.Size, nil
	s.mu.Unlock()
	s.publishProgress()

	if !filepath.IsAbs(s.directory) {
		return s.failed(errors.New("Decky plugin runtime directory is unavailable"))
	}

	if plugin := os.Getenv("DECKY_PLUGIN_DIR"); plugin != "" {
		relative, err := filepath.Rel(plugin, s.directory)
		if err != nil || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
			return s.failed(errors.New("update staging must be outside the installed plugin"))
		}
	}

	if err := os.MkdirAll(s.directory, 0700); err != nil {
		return s.failed(err)
	}

	target := filepath.Join(s.directory, "update-"+release.Version+"-"+release.SHA256+".zip")
	// Verify cached bytes again before each handoff; never trust filename alone.
	if err := verifyDownload(target, release); err == nil {
		return s.ready(target, release)
	}

	if err := s.download(ctx, release, target); err != nil {
		return s.failed(err)
	}

	return s.ready(target, release)
}

func (s *Service) ready(path string, release NextUpdate) (Snapshot, error) {
	s.mu.Lock()
	s.state.Phase, s.state.Downloaded = "ready", release.Size
	s.state.Prepared = &PreparedUpdate{Path: path, Version: release.Version, SHA256: release.SHA256}
	s.mu.Unlock()
	s.publishProgress()

	return s.UpdateState(), nil
}

func (s *Service) failed(err error) (Snapshot, error) {
	s.mu.Lock()
	s.state.Phase, s.state.Error = "failed", err.Error()
	s.mu.Unlock()
	s.publishProgress()

	return s.UpdateState(), err
}

func (s *Service) discover(ctx context.Context) (*NextUpdate, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.manifestURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("check update manifest: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update manifest returned HTTP %d", response.StatusCode)
	}

	var manifest UpdateManifest
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid update manifest: %w", err)
	}

	version := "v" + strings.TrimPrefix(manifest.Version, "v")
	current := "v" + strings.TrimPrefix(s.currentVersion, "v")
	if manifest.SchemaVersion != 1 || manifest.Channel != "stable" || !semver.IsValid(version) || semver.Prerelease(version) != "" || !semver.IsValid(current) {
		return nil, errors.New("unsupported update manifest or current version")
	}

	if semver.Compare(version, current) <= 0 {
		return nil, nil
	}

	for _, artifact := range manifest.Artifacts {
		if artifact.Platform != "linux" || artifact.Arch != s.arch || artifact.Filetype != "zip" {
			continue
		}

		location, err := url.Parse(artifact.URL)
		digest, digestErr := base64.StdEncoding.DecodeString(artifact.Digest)
		if err != nil || location.Scheme != "https" || location.Host == "" || artifact.Size <= 0 || artifact.Size > 1<<30 || artifact.DigestAlgo != "sha256" || digestErr != nil || len(digest) != sha256.Size {
			return nil, errors.New("invalid Decky artifact URL, size, or SHA-256 digest")
		}

		return &NextUpdate{Version: strings.TrimPrefix(version, "v"), Notes: manifest.Notes, URL: artifact.URL, Size: artifact.Size, SHA256: hex.EncodeToString(digest)}, nil
	}

	return nil, errors.New("release has no compatible Decky ZIP")
}

func (s *Service) download(ctx context.Context, release NextUpdate, target string) error {
	stage, err := os.CreateTemp(s.directory, ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(stage.Name())
	defer stage.Close()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return err
	}

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("update download returned HTTP %d", response.StatusCode)
	}

	reader := io.LimitReader(response.Body, release.Size+1)
	buffer := make([]byte, 64<<10)
	for {
		count, readErr := reader.Read(buffer)
		if count > 0 {
			if _, err := stage.Write(buffer[:count]); err != nil {
				return err
			}

			s.mu.Lock()
			s.state.Downloaded += int64(count)
			s.mu.Unlock()
			s.publishProgress()
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return readErr
		}
	}

	if s.UpdateState().Downloaded != release.Size {
		return errors.New("update download size does not match manifest")
	}

	if err := stage.Sync(); err != nil {
		return err
	}

	if err := stage.Close(); err != nil {
		return err
	}

	s.mu.Lock()
	s.state.Phase = "verifying"
	s.mu.Unlock()
	s.publishProgress()

	if err := verifyDownload(stage.Name(), release); err != nil {
		return err
	}

	return os.Rename(stage.Name(), target)
}

func (s *Service) publishProgress() {
	if s.Notifier != nil {
		s.Notifier.SendEvent("updateStatus", s.UpdateState())
	}
}

func (s *Service) cleanup() {
	if !filepath.IsAbs(s.directory) {
		return
	}

	entries, _ := os.ReadDir(s.directory)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		info, err := entry.Info()
		if err != nil || time.Since(info.ModTime()) < 24*time.Hour {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".download-") {
			_ = os.Remove(filepath.Join(s.directory, name))
			continue
		}

		// A stable semantic version has no hyphens; only clean our own archive names.
		parts := strings.Split(name, "-")
		if len(parts) != 3 || parts[0] != "update" || !strings.HasSuffix(parts[2], ".zip") {
			continue
		}

		version, current := "v"+parts[1], "v"+strings.TrimPrefix(s.currentVersion, "v")
		if semver.IsValid(version) && semver.IsValid(current) && semver.Compare(version, current) <= 0 {
			_ = os.Remove(filepath.Join(s.directory, name))
		}
	}
}

func verifyDownload(filename string, update NextUpdate) error {
	info, err := os.Lstat(filename)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != update.Size {
		return errors.New("staged ZIP size or file type is invalid")
	}

	input, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer input.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != update.SHA256 {
		return errors.New("update ZIP SHA-256 does not match manifest")
	}
	return nil
}
