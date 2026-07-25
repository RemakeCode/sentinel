package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sentinel/backend/api"
	"sentinel/backend/bootstrap"
	"sentinel/backend/decky"
	"sentinel/backend/notifier"
)

func main() {
	bootstrap.ConfigureLogger()
	if err := runDecky(); err != nil {
		slog.Error("Decky backend failed", "error", err)
		os.Exit(1)
	}
}

func runDecky() error {
	services := bootstrap.NewServices()
	services.Notifier.SetDeliveryMode(notifier.DeliveryDecky)
	activeDeckSession := decky.IsActiveDeckSession()

	if !activeDeckSession {
		slog.Info("Decky watcher disabled outside active Decky session")
	}

	return startDecky(
		func() error {
			return bootstrap.StartSharedServices(context.Background(), services, bootstrap.StartOptions{StartWatcher: activeDeckSession})
		},
		func() error { return startDeckyServer(services) },
	)
}

func startDecky(startServices func() error, startServer func() error) error {
	if err := startServices(); err != nil {
		return fmt.Errorf("initialize Decky services: %w", err)
	}
	return startServer()
}

func startDeckyServer(services *bootstrap.Services) error {
	router := api.NewRouter(services.Config, services.Steam, services.Watcher, services.Notifier)
	addr, err := decky.GetAPIAddress()
	if err != nil {
		return err
	}
	slog.Info("Decky API Server starting", "addr", addr)
	return http.ListenAndServe(addr, router.Handler())
}
