//go:build !decky

package steam

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func (s *Service) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return s.Start(ctx)
}
