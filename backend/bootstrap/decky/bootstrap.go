package decky

import (
	"context"
	"fmt"

	"sentinel/backend/bootstrap"
	"sentinel/backend/decky/updater"
	"sentinel/backend/notifier"
)

type Services struct {
	*bootstrap.Services
	Updater *updater.Service
}

func NewServices() *Services {
	services := bootstrap.NewServices()
	services.Notifier.SetDeliveryMode(notifier.DeliveryDecky)

	return &Services{
		Services: services,
		Updater:  &updater.Service{Notifier: services.Notifier},
	}
}

func StartServices(ctx context.Context, services *Services) error {
	if err := bootstrap.StartSharedServices(ctx, services.Services, bootstrap.StartOptions{StartWatcher: false}); err != nil {
		return err
	}

	if err := services.Updater.Start(ctx); err != nil {
		return fmt.Errorf("updater startup: %w", err)
	}

	return nil
}
