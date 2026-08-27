package generator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sentinel/backend"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type failingRoundTripper struct{ calls int }

func (f *failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	f.calls++
	return nil, errors.New("network should not be used")
}

type contentRoundTripper struct {
	calls int
	body  string
}

func newToolManager(client *http.Client) *ToolManager {
	return &ToolManager{httpClient: client}
}

func (c *contentRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	c.calls++
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(c.body)),
		Header:     make(http.Header),
	}, nil
}

func TestPinnedAssetsRemainAvailable(t *testing.T) {
	client := &http.Client{Timeout: 20 * time.Second}
	for _, asset := range []struct {
		name string
		url  string
	}{
		{name: "GSE Fork Tools", url: gseForkToolsAsset.releaseURL},
		{name: "gbe_fork DLLs", url: gbeForkDLLAsset.releaseURL},
	} {
		asset := asset
		t.Run(asset.name, func(t *testing.T) {
			response, err := client.Head(asset.url)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestPublishedProviderDirectoriesAreReused(t *testing.T) {
	originalGeneratorDir := backend.GeneratorDir
	backend.GeneratorDir = t.TempDir()
	t.Cleanup(func() { backend.GeneratorDir = originalGeneratorDir })

	generatorDirectory := filepath.Join(backend.GeneratorDir, gseForkToolsAsset.cacheDirectoryName, gseForkToolsAsset.version)
	dllDirectory := filepath.Join(backend.GeneratorDir, gbeForkDLLAsset.cacheDirectoryName, gbeForkDLLAsset.version)
	require.NoError(t, os.MkdirAll(generatorDirectory, 0755))
	require.NoError(t, os.MkdirAll(dllDirectory, 0755))

	transport := &failingRoundTripper{}
	manager := newToolManager(&http.Client{Transport: transport})
	temporaryDirectory := filepath.Join(backend.GeneratorDir, tempDirName)

	generatorExecutable, err := manager.prepareGSEForkToolsExecutable(context.Background(), temporaryDirectory)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(generatorDirectory, filepath.FromSlash(gseForkToolsExecutablePath)), generatorExecutable)

	preparedDLLDirectory, err := manager.prepareGBEForkDLLDirectory(context.Background(), temporaryDirectory)
	require.NoError(t, err)
	require.Equal(t, dllDirectory, preparedDLLDirectory)
	require.Zero(t, transport.calls)
}

func TestDownloadedArchiveIsTemporaryAndVerified(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "gbe-fork")
	transport := &contentRoundTripper{body: "replacement"}
	manager := newToolManager(&http.Client{Transport: transport})

	archive, err := manager.downloadArchive(context.Background(), "https://example.invalid/asset", directory, testSHA256("replacement"))
	require.NoError(t, err)
	require.Equal(t, 1, transport.calls)

	contents, err := os.ReadFile(archive)
	require.NoError(t, err)
	require.Equal(t, "replacement", string(contents))
	require.NoError(t, os.Remove(archive))
	require.NoDirExists(t, archive)
}

func TestDownloadedAssetWithWrongDigestIsRejected(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "gse-fork-tools")
	transport := &contentRoundTripper{body: "unexpected"}
	manager := newToolManager(&http.Client{Transport: transport})

	_, err := manager.downloadArchive(context.Background(), "https://example.invalid/asset", directory, testSHA256("expected"))
	require.ErrorContains(t, err, "SHA-256 mismatch")
	entries, readErr := os.ReadDir(directory)
	require.NoError(t, readErr)
	require.Empty(t, entries)
}

func testSHA256(contents string) string {
	digest := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(digest[:])
}
