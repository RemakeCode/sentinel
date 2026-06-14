package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"sentinel/backend/api"
	"sentinel/backend/bootstrap"
	"sentinel/backend/decky"
	"sentinel/backend/notifier"
)

func init() {
	flag.Bool("decky", false, "Run in Decky plugin mode")
}

func main() {
	flag.Parse()
	bootstrap.ConfigureLogger()
	services := bootstrap.NewServices()
	services.Notifier.SetDeliveryMode(notifier.DeliveryDecky)
	activeDecky := decky.IsDecky()

	if !activeDecky {
		slog.Info("Decky watcher disabled outside active Decky session")
	}

	if err := bootstrap.StartSharedServices(context.Background(), services, bootstrap.StartOptions{StartWatcher: activeDecky}); err != nil {
		slog.Error("Failed to initialize Decky backend", "error", err)
	}
	if err := startDeckyServer(services); err != nil {
		slog.Error("Decky API Server failed", "error", err)
	}
}

func startDeckyServer(services *bootstrap.Services) error {
	router := api.NewRouter(services.Config, services.Steam, services.Watcher, services.Notifier)
	port := decky.GetPort()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	slog.Info("Decky API Server starting", "addr", addr)
	return http.ListenAndServe(addr, router.Handler())
}
