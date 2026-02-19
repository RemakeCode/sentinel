package backend

import (
	"context"
)

//wails:bind
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

// domReady is called after the front-end dom has been loaded
func (app *App) DomReady(ctx context.Context) {
}

// shutdown is called at application termination
func (app *App) shutdown(ctx context.Context) {
}

//
//// GetConfig returns the current configuration
//func (app *App) GetConfig() (*config.CfgFile, error) {
//	return app.config.LoadConfig()
//}
//
//// SaveConfig saves the current configuration to file
//func (app *App) SaveConfig() error {
//	return app.config.SaveConfig()
//}
//

//
//// AddEmulator adds a new emulator to the configuration
//func (app *App) AddEmulator(path string, shouldNotify bool) error {
//
//	emulator := config.Emulator{
//		Path:         path,
//		ShouldNotify: shouldNotify,
//		IsDefault:    false,
//	}
//
//	return app.config.AddEmulator(emulator)
//}
//
//// RemoveEmulator removes an emulator from the configuration by index
//func (app *App) RemoveEmulator(index int) error {
//	if app.config == nil {
//		return nil
//	}
//	return app.config.RemoveEmulator(index)
//}
//
//// ToggleEmulatorNotification toggles the notification setting for an emulator by index
//func (app *App) ToggleEmulatorNotification(index int) error {
//	if app.config == nil {
//		return nil
//	}
//	return app.config.ToggleEmulatorNotification(index, app.ctx)
//}
