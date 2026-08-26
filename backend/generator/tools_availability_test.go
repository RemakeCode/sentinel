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
		{name: "GSE Tools", url: gseReleaseURL},
		{name: "gbe_fork", url: gbeReleaseURL},
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

func TestVersionedAssetCacheIsReused(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "gse", "2026_02_16", "asset.tar.bz2")
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0755))
	require.NoError(t, os.WriteFile(destination, []byte("cached"), 0644))
	transport := &failingRoundTripper{}
	manager := newToolManager(&http.Client{Transport: transport})
	require.NoError(t, manager.downloadTools(context.Background(), "https://unused.invalid/asset", destination, testSHA256("cached")))
	require.Zero(t, transport.calls)
}

func TestInvalidCachedAssetIsDownloadedAgain(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "gbe", "release", "asset.7z")
	require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0755))
	require.NoError(t, os.WriteFile(destination, []byte("corrupt"), 0644))
	transport := &contentRoundTripper{body: "replacement"}
	manager := newToolManager(&http.Client{Transport: transport})

	require.NoError(t, manager.downloadTools(context.Background(), "https://example.invalid/asset", destination, testSHA256("replacement")))
	require.Equal(t, 1, transport.calls)

	contents, err := os.ReadFile(destination)

	require.NoError(t, err)
	require.Equal(t, "replacement", string(contents))
}

func TestDownloadedAssetWithWrongDigestIsRejected(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "gse", "release", "asset.tar.bz2")
	transport := &contentRoundTripper{body: "unexpected"}
	manager := newToolManager(&http.Client{Transport: transport})

	err := manager.downloadTools(context.Background(), "https://example.invalid/asset", destination, testSHA256("expected"))
	require.ErrorContains(t, err, "SHA-256 mismatch")
	require.NoFileExists(t, destination)
}

func testSHA256(contents string) string {
	digest := sha256.Sum256([]byte(contents))
	return hex.EncodeToString(digest[:])
}
