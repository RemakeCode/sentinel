package updater

import (
	"testing"

	wup "github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestAssetMatcherSeparatesInstallations(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "sentinel-decky-plugin.zip"},
		{Name: "sentinel-linux.deb"},
		{Name: "sentinel-linux.tar.gz.sig"},
		{Name: "sentinel-flatpak-linux.tar.gz"},
		{Name: "sentinel-bootstrap.tar.gz"},
		{Name: "sentinel-linux.tar.gz"},
	}
	for _, tc := range []struct {
		kind, platform string
		want           int
	}{
		{"native", "linux", 5},
		{"flatpak", "linux", 3},
		{"native", "windows", -1},
	} {
		t.Run(tc.kind+"/"+tc.platform, func(t *testing.T) {
			index := matchArtifact(tc.kind)(wup.CheckRequest{Platform: tc.platform}, assets)
			if index != tc.want {
				t.Fatalf("selected %d; want %d", index, tc.want)
			}
		})
	}
	request := wup.CheckRequest{Platform: "linux"}
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
