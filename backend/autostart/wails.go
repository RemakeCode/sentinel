//go:build !decky

package autostart

import (
	"context"

	"sentinel/backend"
)

func setEnabled(ctx context.Context, enabled bool) error {
	if backend.IsFlatpak() {
		return setFlatpakAutostart(ctx, enabled)
	}

	return setNativeAutostart(enabled)
}
