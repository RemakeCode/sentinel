package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sentinel/backend/ach"
	"sentinel/backend/config"
	"sentinel/backend/logger"
	"sentinel/backend/notifier"
	"sentinel/backend/steam"
	"sentinel/backend/watcher"
)

type Services struct {
	Config   *config.File
	Ach      *ach.Service
	Steam    *steam.Service
	Watcher  *watcher.Service
	Notifier *notifier.Service
}

type StartOptions struct {
	StartWatcher bool
}

func NewServices() *Services {
	configService := &config.File{}
	achService := &ach.Service{}
	steamService := &steam.Service{
		Config: configService,
		Ach:    achService,
	}
	notifierService := &notifier.Service{
		Config: configService,
	}
	watcherService := &watcher.Service{
		Steam:    steamService,
		Ach:      achService,
		Config:   configService,
		Notifier: notifierService,
	}

	return &Services{
		Config:   configService,
		Ach:      achService,
		Steam:    steamService,
		Watcher:  watcherService,
		Notifier: notifierService,
	}
}

func ConfigureLogger() (*slog.Logger, string) {
	appLogger := logger.New()
	logLevel := "info"
	cfg, err := config.Get()
	if err == nil && cfg != nil {
		logLevel = cfg.LogLevel
		logger.SetLevel(logger.ParseLevel(logLevel))
	}
	slog.SetDefault(appLogger)
	return appLogger, logLevel
}

func StartSharedServices(ctx context.Context, services *Services, options StartOptions) error {
	if err := services.Config.Start(ctx); err != nil {
		return fmt.Errorf("config startup: %w", err)
	}
	if err := services.Steam.Start(ctx); err != nil {
		return fmt.Errorf("steam startup: %w", err)
	}
	if err := services.Ach.Start(ctx); err != nil {
		return fmt.Errorf("achievements startup: %w", err)
	}
	if options.StartWatcher {
		if err := services.Watcher.Startup(ctx); err != nil {
			return fmt.Errorf("watcher startup: %w", err)
		}
	}
	if err := services.Notifier.Start(ctx); err != nil {
		return fmt.Errorf("notifier startup: %w", err)
	}
	return nil
}
