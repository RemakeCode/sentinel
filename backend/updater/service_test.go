package updater

import (
	"testing"

	wup "github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestAssetMatcherSeparatesInstallations(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "sentinel-decky-plugin-linux-amd64.zip"},
		{Name: "sentinel-linux-amd64.deb"},
		{Name: "sentinel-linux-amd64.tar.gz.sig"},
		{Name: "sentinel-flatpak-linux-amd64.tar.gz"},
		{Name: "sentinel-linux-arm64.tar.gz"},
		{Name: "sentinel-linux-amd64.tar.gz"},
	}
	for _, tc := range []struct {
		kind, platform, arch string
		want                 int
	}{
		{"native", "linux", "amd64", 5},
		{"flatpak", "linux", "amd64", 3},
		{"native", "linux", "arm64", 4},
		{"flatpak", "linux", "arm64", -1},
		{"native", "windows", "amd64", -1},
	} {
		t.Run(tc.kind+"/"+tc.platform+"/"+tc.arch, func(t *testing.T) {
			index := matchArtifact(tc.kind)(wup.CheckRequest{Platform: tc.platform, Arch: tc.arch}, assets)
			if index != tc.want {
				t.Fatalf("selected %d; want %d", index, tc.want)
			}
		})
	}
	request := wup.CheckRequest{Platform: "linux", Arch: "amd64"}
	if index := matchArtifact("flatpak")(request, assets[4:]); index != -1 {
		t.Fatalf("flatpak fell back to native asset: %d", index)
	}
	if index := matchArtifact("native")(request, assets[3:4]); index != -1 {
		t.Fatalf("native fell back to flatpak asset: %d", index)
	}
	if index := matchArtifact("native")(request, nil); index != -1 {
		t.Fatalf("empty release selected asset: %d", index)
	}
}
