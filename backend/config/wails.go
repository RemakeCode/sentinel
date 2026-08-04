//go:build !decky

package config

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (c *File) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return c.Start(ctx)
}
