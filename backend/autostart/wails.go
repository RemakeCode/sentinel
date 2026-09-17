//go:build !decky

package autostart

import (
	"context"
	"os"
)

func setEnabled(ctx context.Context, enabled bool) error {
	if _, err := os.Stat("/.flatpak-info"); err == nil {
		return setFlatpakAutostart(ctx, enabled)
	}

	return setNativeAutostart(enabled)
}
