//go:build !decky

package notifier

import "embed"

//go:embed out/sentinel-custom-notification-*
var embeddedCustomNotificationBinaries embed.FS

//go:embed out/io.github.remakecode.sentinel/*
var embeddedGNOMEExtension embed.FS

func init() {
	customNotificationBinaries = embeddedCustomNotificationBinaries
	gnomeExtensionFiles = embeddedGNOMEExtension
}
