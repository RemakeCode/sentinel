//go:build !decky

package autostart

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"sentinel/backend"
	"sentinel/backend/config"

	"github.com/godbus/dbus/v5"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	portalDestination = "org.freedesktop.portal.Desktop"
	portalPath        = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	backgroundIface   = "org.freedesktop.portal.Background"
	requestIface      = "org.freedesktop.portal.Request"
	requestMethod     = backgroundIface + ".RequestBackground"
	responseSignal    = requestIface + ".Response"
	portalTimeout     = 30 * time.Second
	startMinimized    = "--startminimized"
)

type Service struct {
	Config *config.File
}

func NewService(cfg *config.File) *Service {
	return &Service{Config: cfg}
}

func (s *Service) SetEnabled(enabled bool) error {
	if err := s.Config.SetStartOnLogin(enabled); err != nil {
		return err
	}
	return setEnabled(context.Background(), enabled)
}

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	if err := setEnabled(ctx, s.Config.GetStartOnLogin()); err != nil {
		slog.Error("Failed to sync autostart", "error", err)
	}
	return nil
}

func setFlatpakAutostart(ctx context.Context, enabled bool) error {
	ctx, cancel := context.WithTimeout(ctx, portalTimeout)
	defer cancel()

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect to session bus: %w", err)
	}
	defer conn.Close()

	token, err := newHandleToken()
	if err != nil {
		return fmt.Errorf("create portal request token: %w", err)
	}

	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)

	matchOptions := []dbus.MatchOption{
		dbus.WithMatchInterface(requestIface),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignalContext(ctx, matchOptions...); err != nil {
		return fmt.Errorf("subscribe to portal response: %w", err)
	}
	defer conn.RemoveMatchSignal(matchOptions...)

	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"reason":       dbus.MakeVariant("Start Sentinel in the background when you sign in."),
		"autostart":    dbus.MakeVariant(enabled),
		"commandline":  dbus.MakeVariant([]string{"sentinel", startMinimized}),
	}
	object := conn.Object(portalDestination, portalPath)
	var handle dbus.ObjectPath
	call := object.CallWithContext(ctx, requestMethod, 0, "", options)
	if err := call.Store(&handle); err != nil {
		return fmt.Errorf("request portal background permission: %w", err)
	}
	if !handle.IsValid() {
		return fmt.Errorf("portal returned invalid request handle %q", handle)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case signal, ok := <-signals:
			if !ok {
				return errors.New("portal response channel closed")
			}
			if signal == nil || signal.Path != handle || signal.Name != responseSignal {
				continue
			}
			return parsePortalResponse(signal.Body, enabled)
		}
	}
}

func setNativeAutostart(enabled bool) error {
	app := application.Get()
	if app == nil {
		return nil
	}

	var err error
	if enabled {
		err = app.Autostart.EnableWithOptions(application.AutostartOptions{
			Identifier: backend.ApplicationID,
			Arguments:  []string{startMinimized},
		})
	} else {
		err = app.Autostart.Disable()
	}

	if errors.Is(err, application.ErrAutostartNotSupported) {
		return nil
	}
	return err
}

func parsePortalResponse(body []any, requested bool) error {
	if len(body) != 2 {
		return fmt.Errorf("portal response has %d values, want 2", len(body))
	}

	response, ok := body[0].(uint32)
	if !ok {
		return fmt.Errorf("portal response code has type %T, want uint32", body[0])
	}
	if response != 0 {
		return fmt.Errorf("portal request was not completed (response code %d)", response)
	}

	results, ok := body[1].(map[string]dbus.Variant)
	if !ok {
		return fmt.Errorf("portal response results have type %T, want map[string]dbus.Variant", body[1])
	}
	autostart, ok := results["autostart"]
	if !ok {
		return errors.New("portal response omitted autostart state")
	}
	value, ok := autostart.Value().(bool)
	if !ok {
		return fmt.Errorf("portal autostart state has type %T, want bool", autostart.Value())
	}
	if value != requested {
		return fmt.Errorf("portal autostart state is %t, want %t", value, requested)
	}

	return nil
}

func newHandleToken() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "sentinel_" + hex.EncodeToString(bytes[:]), nil
}
