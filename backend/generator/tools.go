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

const (
	gseReleaseURL             = "https://github.com/alex47exe/gse_fork_tools/releases/download/" + backend.GSEToolsVersion + "/" + backend.GSEToolsAssetName
	gbeReleaseURL             = "https://github.com/Detanup01/gbe_fork/releases/download/" + backend.GBEForkVersion + "/" + backend.GBEForkAssetName
	gseExecutableRelativePath = "generate_emu_config/generate_emu_config"
	generatorTimeout          = 5 * time.Minute
)

var diagnosticSecret = regexp.MustCompile(`(?i)(refresh[_ -]?token|access[_ -]?token|authorization)[^\r\n]*|[A-Za-z0-9_-]{24,}\.[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]{12,}`)

type ToolManager struct {
	httpClient *http.Client
}

type PreparedTools struct {
	GSEExecutable string
	GBEDLL        string
	StagingDir    string
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

func (m *ToolManager) PrepareTools(ctx context.Context, bitness DLLBitness) (*PreparedTools, error) {
	if err := os.MkdirAll(backend.GeneratorDir, 0755); err != nil {
		return nil, fmt.Errorf("create generator cache: %w", err)
	}
	selectedMemberPath := gbeArchivePath(bitness)
	selectedOutputName := filepath.Base(selectedMemberPath)

	gseDir := filepath.Join(backend.GeneratorDir, "gse", backend.GSEToolsVersion)
	gseArchive := filepath.Join(gseDir, backend.GSEToolsAssetName)
	if err := m.downloadTools(ctx, gseReleaseURL, gseArchive, backend.GSEToolsSHA256); err != nil {
		return nil, fmt.Errorf("prepare gse tools: %w", err)
	}

	gseExecutable := filepath.Join(gseDir, filepath.FromSlash(gseExecutableRelativePath))
	if _, err := os.Stat(gseExecutable); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check cached gse executable: %w", err)
		}

		if err := extractTarBzip2(gseArchive, gseDir); err != nil {
			return nil, fmt.Errorf("extract gse tools: %w", err)
		}
	}

	gbeDir := filepath.Join(backend.GeneratorDir, "gbe", backend.GBEForkVersion)
	gbeArchive := filepath.Join(gbeDir, backend.GBEForkAssetName)
	if err := m.downloadTools(ctx, gbeReleaseURL, gbeArchive, backend.GBEForkSHA256); err != nil {
		return nil, fmt.Errorf("prepare gbe release: %w", err)
	}

	var pending []sevenZipMember
	for _, memberBitness := range []DLLBitness{BitnessX86, BitnessX64} {
		memberPath := gbeArchivePath(memberBitness)
		outputPath := filepath.Join(gbeDir, filepath.Base(memberPath))
		if _, err := os.Stat(outputPath); err == nil {
			continue
		}
		pending = append(pending, sevenZipMember{archivePath: memberPath, outputPath: outputPath})
	}

	if len(pending) > 0 {
		if err := extractSevenZipMembers(gbeArchive, pending); err != nil {
			return nil, fmt.Errorf("extract cached GBE DLLs: %w", err)
		}
	}

	stagingBase := filepath.Join(backend.GeneratorDir, "staging")
	if err := os.MkdirAll(stagingBase, 0700); err != nil {
		return nil, fmt.Errorf("create generator staging: %w", err)
	}

	stagingTemp, err := os.MkdirTemp(stagingBase, "operation-")
	if err != nil {
		return nil, fmt.Errorf("create operation staging: %w", err)
	}

	cachedDLL := filepath.Join(gbeDir, selectedOutputName)
	gbeDLL := filepath.Join(stagingTemp, selectedOutputName)
	if err := copyFile(cachedDLL, gbeDLL, 0755); err != nil {
		_ = os.RemoveAll(stagingTemp)
		return nil, fmt.Errorf("stage gbe DLL: %w", err)
	}

	return &PreparedTools{
		GSEExecutable: gseExecutable,
		GBEDLL:        gbeDLL,
		StagingDir:    stagingTemp,
	}, nil
}

func (m *ToolManager) downloadTools(ctx context.Context, url, destination, expectedSHA256 string) error {
	if info, err := os.Stat(destination); err == nil {
		if info.IsDir() {
			return errors.New("cached asset path is a directory")
		}
		if info.Size() > 0 && verifyFileSHA256(destination, expectedSHA256) == nil {
			return nil
		}
		if err := os.Remove(destination); err != nil {
			return fmt.Errorf("remove invalid cached asset: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(destination), ".download-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		_ = tmp.Close()
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		_ = tmp.Close()
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = tmp.Close()
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := verifyFileSHA256(tmpPath, expectedSHA256); err != nil {
		return fmt.Errorf("verify downloaded asset: %w", err)
	}

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, destination)
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

	info, err := os.Stat(tmpPath)
	if err != nil || info.Mode().Perm() != 0600 {
		return errors.New("temporary authentication handoff is not owner-only")
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

	return output.String(), err
}

func sanitizeDiagnostics(value string) string {
	return diagnosticSecret.ReplaceAllString(value, "[redacted]")
}

func cleanupGeneratorTransientData() error {
	var cleanupErr error
	gseRoot := filepath.Join(backend.GeneratorDir, "gse", backend.GSEToolsVersion)
	_ = filepath.Walk(gseRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}

		if info.Name() == "refresh_tokens.json" || (info.IsDir() && info.Name() == "_OUTPUT") {
			if removeErr := os.RemoveAll(path); removeErr != nil && cleanupErr == nil {
				cleanupErr = removeErr
			}

			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	if err := os.RemoveAll(filepath.Join(backend.GeneratorDir, "staging")); err != nil && cleanupErr == nil {
		cleanupErr = err
	}

	return cleanupErr
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

func gbeArchivePath(bitness DLLBitness) string {
	if bitness == BitnessX86 {
		return "release/experimental/x86/steam_api.dll"
	}
	return "release/experimental/x64/steam_api64.dll"
}
