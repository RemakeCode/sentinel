package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sentinel/backend/api"
	"sentinel/backend/bootstrap"
	"sentinel/backend/decky"
	"sentinel/backend/notifier"
	"syscall"
	"time"
)

func main() {
	bootstrap.ConfigureLogger()
	if err := runDecky(); err != nil {
		slog.Error("Decky backend failed", "error", err)
		os.Exit(1)
	}
}

func runDecky() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	services := bootstrap.NewServices()
	services.Notifier.SetDeliveryMode(notifier.DeliveryDecky)
	sessionSupervisor := newDeckSessionSupervisor(services.Watcher, decky.IsActiveDeckSession)

	if err := bootstrap.StartSharedServices(ctx, services, bootstrap.StartOptions{StartWatcher: false}); err != nil {
		return fmt.Errorf("initialize Decky services: %w", err)
	}
	defer bootstrap.ShutdownSharedServices(services)

	go sessionSupervisor.Run(ctx)
	return startDeckyServer(ctx, services)
}

func startDecky(startServices func() error, startSupervisor func(), startServer func() error) error {
	if err := startServices(); err != nil {
		return fmt.Errorf("initialize Decky services: %w", err)
	}
	startSupervisor()
	return startServer()
}

func startDeckyServer(ctx context.Context, services *bootstrap.Services) error {
	router := api.NewRouter(services.Config, services.Steam, services.Watcher, services.Notifier, services.Generator)
	addr, err := decky.GetAPIAddress()
	if err != nil {
		return err
	}
	slog.Info("Decky API Server starting", "addr", addr)
	server := &http.Server{Addr: addr, Handler: router.Handler(), ReadHeaderTimeout: 10 * time.Second}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
