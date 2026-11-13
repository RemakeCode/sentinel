package updater

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func pluginZIP(t *testing.T, version string, missing string, remote bool) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	binary := make([]byte, 64)
	copy(binary, "\x7fELF")
	binary[4], binary[5], binary[18] = 2, 1, 62
	pkg := fmt.Sprintf(`{"version":%q}`, version)
	if remote {
		pkg = fmt.Sprintf(`{"version":%q,"remote_binary":[{"url":"https://example.com/bin"}]}`, version)
	}
	for name, data := range map[string][]byte{"plugin.json": []byte(`{"name":"Sentinel","flags":[]}`), "package.json": []byte(pkg), "main.py": []byte("entry point"), "dist/index.js": []byte("frontend"), "bin/sentinel-decky": binary} {
		if name == missing {
			continue
		}
		file, err := writer.Create("sentinel/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestDeckyDiscoveryAndPreparation(t *testing.T) {
	payload := pluginZIP(t, "1.1.0", "", false)
	digest := sha256.Sum256(payload)
	var downloads atomic.Int32
	var failDownload atomic.Bool
	var failManifest atomic.Bool
	version := "1.1.0"
	schema := 1
	platform := "linux"
	digestText := base64.StdEncoding.EncodeToString(digest[:])
	artifactURL := ""
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/payload" {
			downloads.Add(1)
			if failDownload.Load() {
				http.Error(w, "offline", http.StatusServiceUnavailable)
				return
			}
			w.Write(payload)
			return
		}
		if failManifest.Load() {
			http.Error(w, "offline", 503)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"schemaVersion": schema, "version": version, "channel": "stable", "notes": "# Shared notes", "artifacts": []any{map[string]any{"url": artifactURL, "platform": platform, "arch": "amd64", "filetype": "zip", "size": len(payload), "digestAlgo": "sha256", "digest": digestText}}})
	}))
	defer server.Close()
	artifactURL = server.URL + "/payload"
	s := &Service{client: server.Client(), manifestURL: server.URL + "/manifest", currentVersion: "1.0.0", arch: "amd64", directory: t.TempDir()}
	s.Start(context.Background())
	ctx := context.Background()
	for _, test := range []struct {
		version   string
		available bool
	}{{"0.9.0", false}, {"1.0.0", false}, {"1.1.0", true}} {
		version = test.version
		state, err := s.CheckUpdate(ctx)
		if err != nil || (state.Release != nil) != test.available {
			t.Fatalf("version %s: %+v, %v", version, state, err)
		}
	}
	if downloads.Load() != 0 {
		t.Fatal("discovery downloaded payload")
	}
	for _, kind := range []string{"schema", "digest", "platform", "version", "offline"} {
		schema, platform, version, digestText = 1, "linux", "1.1.0", base64.StdEncoding.EncodeToString(digest[:])
		switch kind {
		case "schema":
			schema = 99
		case "digest":
			digestText = "invalid"
		case "platform":
			platform = "windows"
		case "version":
			version = "nonsense"
		case "offline":
			failManifest.Store(true)
		}
		if _, err := s.CheckUpdate(ctx); err == nil {
			t.Fatalf("accepted %s", kind)
		}
		failManifest.Store(false)
	}
	schema, platform, version, digestText = 1, "linux", "1.1.0", base64.StdEncoding.EncodeToString(digest[:])
	if _, err := s.CheckUpdate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareUpdate(ctx, "2.0.0"); err == nil {
		t.Fatal("accepted unselected version")
	}
	failDownload.Store(true)
	if _, err := s.PrepareUpdate(ctx, "1.1.0"); err == nil {
		t.Fatal("accepted failed download")
	}
	failDownload.Store(false)
	// A structurally valid manifest with a wrong digest must never publish a ZIP.
	digestText = base64.StdEncoding.EncodeToString(make([]byte, sha256.Size))
	if _, err := s.CheckUpdate(ctx); err != nil {
		t.Fatal(err)
	}
	failed, err := s.PrepareUpdate(ctx, "1.1.0")
	if err == nil || failed.Prepared != nil || failed.Phase != "failed" {
		t.Fatalf("checksum failure: %+v %v", failed, err)
	}
	if entries, err := os.ReadDir(s.directory); err != nil || len(entries) != 0 {
		t.Fatalf("failed preparation left published files: %v %v", entries, err)
	}
	digestText = base64.StdEncoding.EncodeToString(digest[:])
	if _, err := s.CheckUpdate(ctx); err != nil {
		t.Fatal(err)
	}
	// A changing latest release must not replace the selected metadata mid-download.
	version = "2.0.0"
	state, err := s.PrepareUpdate(ctx, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != "ready" || state.Prepared.Version != "1.1.0" || state.Prepared.SHA256 != hex.EncodeToString(digest[:]) {
		t.Fatalf("bad handoff: %+v", state)
	}
	before := downloads.Load()
	if _, err := s.PrepareUpdate(ctx, "1.1.0"); err != nil || downloads.Load() != before {
		t.Fatalf("cache not reused: %v", err)
	}
	// Corrupt cached bytes must be repaired from a verified download.
	if err := os.WriteFile(state.Prepared.Path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareUpdate(ctx, "1.1.0"); err != nil || downloads.Load() != before+1 {
		t.Fatalf("corrupt cache trusted: %v", err)
	}
	// A new backend startup retains a newer staged file pending Loader confirmation.
	older := time.Now().Add(-48 * time.Hour)
	os.Chtimes(state.Prepared.Path, older, older)
	replacement := &Service{client: server.Client(), currentVersion: "1.0.0", arch: "amd64", directory: s.directory}
	replacement.Start(context.Background())
	if _, err := os.Stat(state.Prepared.Path); err != nil {
		t.Fatal("pending ZIP deleted", err)
	}
	replacement.currentVersion = "1.1.0"
	replacement.cleanup()
	if _, err := os.Stat(state.Prepared.Path); !os.IsNotExist(err) {
		t.Fatal("completed old ZIP not cleaned")
	}
}

func TestDeckyOperationSerialization(t *testing.T) {
	s := &Service{currentVersion: "1.0.0", directory: t.TempDir()}
	s.Start(context.Background())
	s.operation.Lock()
	if _, err := s.CheckUpdate(context.Background()); err != ErrUpdateBusy {
		t.Fatalf("check: %v", err)
	}
	if _, err := s.PrepareUpdate(context.Background(), "1.1.0"); err != ErrUpdateBusy {
		t.Fatalf("prepare: %v", err)
	}
	s.operation.Unlock()
}
