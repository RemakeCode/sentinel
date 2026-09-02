//go:build !decky

package contextmenu

import (
	"encoding/json"
	"log/slog"
	"strings"
	"unicode"

	"sentinel/backend"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	GameCardMenuID = "game-card-menu"
)

type gameCardContextData struct {
	AppID string `json:"appId"`
}

func RegisterGameCardMenus(app *application.App) {
	app.ContextMenu.Add(GameCardMenuID, gameCardMenu(app))
}

func gameCardMenu(app *application.App) *application.ContextMenu {
	menu := app.ContextMenu.New()
	refreshGameMenuItem(menu, app)
	return menu
}

func refreshGameMenuItem(menu *application.ContextMenu, app *application.App) {
	menu.Add("Refresh Game").OnClick(func(ctx *application.Context) {
		data, ok := parseGameCardContextData(ctx.ContextMenuData())
		if !ok {
			slog.Warn("Ignoring invalid game context menu data")
			return
		}

		app.Event.Emit(backend.EventRefreshGameRequested, data.AppID)
	})
}

func parseGameCardContextData(value string) (gameCardContextData, bool) {
	var data gameCardContextData
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return gameCardContextData{}, false
	}

	data.AppID = strings.TrimSpace(data.AppID)
	if !validSteamAppID(data.AppID) {
		return gameCardContextData{}, false
	}

	return data, true
}

func validSteamAppID(appID string) bool {
	if appID == "" {
		return false
	}

	for _, value := range appID {
		if !unicode.IsDigit(value) {
			return false
		}
	}

	return true
}
