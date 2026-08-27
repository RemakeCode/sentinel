package generator

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sentinel/backend"
	"strings"
	"time"

	"github.com/bodgit/sevenzip"
)

var diagnosticSecret = regexp.MustCompile(`(?i)(refresh[_ -]?token|access[_ -]?token|authorization)[^\r\n]*|[A-Za-z0-9_-]{24,}\.[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}`)

type ToolManager struct {
	httpClient *http.Client
}

type PreparedTools struct {
	GeneratorExecutable string
	ReplacementDLL      string
	WorkspaceDir        string
}

type sevenZipMember struct {
	archivePath string
	outputPath  string
}

type diagnosticBuffer struct {
	bytes.Buffer
}

func (b *diagnosticBuffer) Write(data []byte) (int, error) {
	const limit = 32 * 1024

	original := len(data)
	remaining := limit - b.Len()
	if remaining > 0 {
		if len(data) > remaining {
			data = data[:remaining]
		}

		_, _ = b.Buffer.Write(data)
	}

	return original, nil
}

func (b *diagnosticBuffer) String() string {
	return strings.TrimSpace(b.Buffer.String())
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		httpClient: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (m *ToolManager) PrepareTools(ctx context.Context, target InstallTarget) (*PreparedTools, error) {
	if err := os.MkdirAll(backend.GeneratorDir, 0755); err != nil {
		return nil, fmt.Errorf("create generator asset directory: %w", err)
	}

	temporaryDirectory := filepath.Join(backend.GeneratorDir, tempDirName)
	defer os.RemoveAll(temporaryDirectory)

	generatorExecutable, err := m.prepareGSEForkToolsExecutable(ctx, temporaryDirectory)
	if err != nil {
		return nil, err
	}

	gbeForkDLLDirectory, err := m.prepareGBEForkDLLDirectory(ctx, temporaryDirectory)
	if err != nil {
		return nil, err
	}

	workspaceRoot := filepath.Join(backend.GeneratorDir, "operations")
	if err := os.MkdirAll(workspaceRoot, 0700); err != nil {
		return nil, fmt.Errorf("create operation workspace directory: %w", err)
	}

	workspace, err := os.MkdirTemp(workspaceRoot, "operation-")
	if err != nil {
		return nil, fmt.Errorf("create operation workspace: %w", err)
	}

	replacementName := filepath.Base(target.ReplacementDLLPath)
	assetDLL := filepath.Join(gbeForkDLLDirectory, replacementName)
	workspaceDLL := filepath.Join(workspace, replacementName)
	if err := copyFile(assetDLL, workspaceDLL, 0755); err != nil {
		_ = os.RemoveAll(workspace)
		return nil, fmt.Errorf("copy GBE DLL into operation workspace: %w", err)
	}

	return &PreparedTools{
		GeneratorExecutable: generatorExecutable,
		ReplacementDLL:      workspaceDLL,
		WorkspaceDir:        workspace,
	}, nil
}

func (m *ToolManager) prepareGSEForkToolsExecutable(ctx context.Context, temporaryDirectory string) (string, error) {
	gseForkToolsDirectory := filepath.Join(backend.GeneratorDir, gseForkToolsAsset.cacheDirectoryName, gseForkToolsAsset.version)
	if dirExists(gseForkToolsDirectory) {
		return filepath.Join(gseForkToolsDirectory, filepath.FromSlash(gseForkToolsExecutablePath)), nil
	}

	gseForkToolsRoot := filepath.Dir(gseForkToolsDirectory)
	if err := os.MkdirAll(gseForkToolsRoot, 0755); err != nil {
		return "", fmt.Errorf("create GSE Fork Tools asset directory: %w", err)
	}

	archive, err := m.downloadArchive(ctx, gseForkToolsAsset.releaseURL, temporaryDirectory, gseForkToolsAsset.sha256)
	if err != nil {
		return "", fmt.Errorf("download GSE Fork Tools archive: %w", err)
	}
	defer os.Remove(archive)

	extractionDir, err := os.MkdirTemp(temporaryDirectory, "gse-extract-")
	if err != nil {
		return "", fmt.Errorf("create temporary GSE Fork Tools extraction directory: %w", err)
	}
	defer os.RemoveAll(extractionDir)

	if err := extractTarBzip2(archive, extractionDir); err != nil {
		return "", fmt.Errorf("extract GSE Fork Tools archive: %w", err)
	}
	if err := finalizeVersionedAssetDirectory(extractionDir, gseForkToolsDirectory); err != nil {
		return "", fmt.Errorf("finalize GSE Fork Tools asset directory: %w", err)
	}

	return filepath.Join(gseForkToolsDirectory, filepath.FromSlash(gseForkToolsExecutablePath)), nil
}

func (m *ToolManager) prepareGBEForkDLLDirectory(ctx context.Context, temporaryDirectory string) (string, error) {
	gbeForkDLLDirectory := filepath.Join(backend.GeneratorDir, gbeForkDLLAsset.cacheDirectoryName, gbeForkDLLAsset.version)
	if dirExists(gbeForkDLLDirectory) {
		return gbeForkDLLDirectory, nil
	}

	gbeRoot := filepath.Dir(gbeForkDLLDirectory)
	if err := os.MkdirAll(gbeRoot, 0755); err != nil {
		return "", fmt.Errorf("create gbe_fork DLL asset directory: %w", err)
	}

	archive, err := m.downloadArchive(ctx, gbeForkDLLAsset.releaseURL, temporaryDirectory, gbeForkDLLAsset.sha256)
	if err != nil {
		return "", fmt.Errorf("download gbe_fork DLL archive: %w", err)
	}
	defer os.Remove(archive)

	extractionDir, err := os.MkdirTemp(temporaryDirectory, "gbe-extract-")
	if err != nil {
		return "", fmt.Errorf("create temporary gbe_fork DLL extraction directory: %w", err)
	}
	defer os.RemoveAll(extractionDir)

	members := make([]sevenZipMember, 0, len(gbeForkDLLMembers))
	for _, archivePath := range gbeForkDLLMembers {
		members = append(members, sevenZipMember{
			archivePath: archivePath,
			outputPath:  filepath.Join(extractionDir, filepath.Base(archivePath)),
		})
	}
	if err := extractSevenZipMembers(archive, members); err != nil {
		return "", fmt.Errorf("extract gbe_fork DLL archive members: %w", err)
	}
	if err := finalizeVersionedAssetDirectory(extractionDir, gbeForkDLLDirectory); err != nil {
		return "", fmt.Errorf("finalize gbe_fork DLL asset directory: %w", err)
	}

	return gbeForkDLLDirectory, nil
}

func (m *ToolManager) downloadArchive(ctx context.Context, url, directory, expectedSHA256 string) (string, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}

	tmp, err := os.CreateTemp(directory, ".download-*")
	if err != nil {
		return "", err
	}

	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		_ = tmp.Close()
		return "", err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		_ = tmp.Close()
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = tmp.Close()
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return "", err
	}

	if err := tmp.Close(); err != nil {
		return "", err
	}

	if err := verifyFileSHA256(tmpPath, expectedSHA256); err != nil {
		return "", fmt.Errorf("verify downloaded asset: %w", err)
	}

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return "", err
	}

	success = true
	return tmpPath, nil
}

func writeGSETokenFile(path, accountName, refreshToken string) error {
	data, err := json.Marshal(map[string]string{accountName: refreshToken})
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".refresh_tokens-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func runGSEGenerator(parent context.Context, executable, appID string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, generatorTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, executable, appID)
	command.Dir = filepath.Dir(executable)

	var output diagnosticBuffer
	command.Stdout = &output
	command.Stderr = &output

	err := command.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output.String(), errors.New("generator timed out after five minutes")
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return output.String(), ctx.Err()
	}

	return output.String(), err
}

func sanitizeDiagnostics(value string) string {
	return diagnosticSecret.ReplaceAllString(value, "[redacted]")
}

func cleanupGeneratorTransientData() {
	gseForkToolsDirectory := filepath.Join(backend.GeneratorDir, gseForkToolsAsset.cacheDirectoryName, gseForkToolsAsset.version)
	generatorExecutable := filepath.Join(gseForkToolsDirectory, filepath.FromSlash(gseForkToolsExecutablePath))
	removeGeneratorTransientPath(filepath.Join(filepath.Dir(generatorExecutable), gseTokenFilename), "stale GSE authentication handoff")
	removeGeneratorTransientPath(filepath.Join(filepath.Dir(generatorExecutable), gseOutputDirectoryName), "stale GSE generated output")
	removeGeneratorTransientPath(filepath.Join(backend.GeneratorDir, "operations"), "stale GBE operation workspaces")
	removeGeneratorTransientPath(filepath.Join(backend.GeneratorDir, tempDirName), "stale generator temporary files")
}

func removeGeneratorTransientPath(path, resource string) {
	if err := os.RemoveAll(path); err != nil {
		slog.Warn("remove generator transient data", "resource", resource, "error", err)
	}
}

func verifyFileSHA256(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	digest := sha256.New()
	if _, err := io.Copy(digest, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(digest.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("SHA-256 mismatch: expected %s, got %s", expected, actual)
	}
	return nil
}

func extractTarBzip2(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := tar.NewReader(bzip2.NewReader(file))

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		rel, err := sanitizeArchivePath(header.Name)
		if err != nil {
			return err
		}

		if rel == "." {
			continue
		}

		target := filepath.Join(destination, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, archiveMode(header.Mode, 0755)); err != nil {
				return err
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			out, err := os.OpenFile(
				target,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
				archiveMode(header.Mode, 0644),
			)

			if err != nil {
				return err
			}

			_, copyErr := io.Copy(out, reader)
			closeErr := out.Close()

			if copyErr != nil {
				return copyErr
			}

			if closeErr != nil {
				return closeErr
			}
		}
	}
	return nil
}

func extractSevenZipMembers(archivePath string, members []sevenZipMember) error {
	reader, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	pending := make(map[string]string, len(members))
	for _, member := range members {
		pending[member.archivePath] = member.outputPath
	}

	for _, file := range reader.File {
		member := filepath.ToSlash(file.Name)
		outputPath, ok := pending[member]
		if !ok {
			continue
		}

		if file.FileInfo().IsDir() {
			return fmt.Errorf("archive member %q is a directory", member)
		}
		if err := writeSevenZipMember(file, outputPath); err != nil {
			return err
		}

		delete(pending, member)
		if len(pending) == 0 {
			return nil
		}
	}

	for member := range pending {
		return fmt.Errorf("archive member %q not found", member)
	}

	return nil
}

func writeSevenZipMember(file *sevenzip.File, outputPath string) error {
	in, err := file.Open()
	if err != nil {
		return err
	}

	out, err := os.CreateTemp(filepath.Dir(outputPath), ".gbe-member-*")
	if err != nil {
		_ = in.Close()
		return err
	}
	temporary := out.Name()
	defer os.Remove(temporary)

	_, copyErr := io.Copy(out, in)
	inputCloseErr := in.Close()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}

	if inputCloseErr != nil {
		return inputCloseErr
	}

	if closeErr != nil {
		return closeErr
	}

	if err := os.Chmod(temporary, 0755); err != nil {
		return err
	}

	return os.Rename(temporary, outputPath)
}

func sanitizeArchivePath(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))

	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("archive contains absolute path: %q", name)
	}

	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive path escapes destination: %q", name)
	}

	return clean, nil
}

func archiveMode(mode int64, fallback os.FileMode) os.FileMode {
	value := os.FileMode(mode).Perm()
	if value == 0 {
		return fallback
	}

	return value
}

func finalizeVersionedAssetDirectory(temporaryDirectory, versionedDirectory string) error {
	if err := os.RemoveAll(versionedDirectory); err != nil {
		return err
	}
	return os.Rename(temporaryDirectory, versionedDirectory)
}
