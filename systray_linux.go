package main

//TODO: Using fyne.io's systray until wails3 implements openMenu for systray
//on Linux

import (
	"fyne.io/systray"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func setupSystray(app *application.App, window *application.WebviewWindow, icon []byte) (startFn, endFn func()) {
	startFn, endFn = systray.RunWithExternalLoop(func() {
		systray.SetIcon(icon)
		systray.SetTooltip("Sentinel")

		showItem := systray.AddMenuItem("Show", "Show Sentinel")
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit", "Exit Sentinel")

		go func() {
			for {
				select {
				case <-showItem.ClickedCh:
					window.Show()
					window.Focus()
				case <-exitItem.ClickedCh:
					systray.Quit()
					app.Quit()
				}
			}
		}()
	}, nil)
	return
}
