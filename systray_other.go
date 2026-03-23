//go:build !linux

package main

import "github.com/wailsapp/wails/v3/pkg/application"

func setupSystray(app *application.App, window *application.WebviewWindow, icon []byte) (startFn, endFn func()) {
	return func() {}, func() {}
}
