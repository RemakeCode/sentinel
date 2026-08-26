//go:build !decky

package contextmenu

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGameCardContextData(t *testing.T) {
	data, ok := parseGameCardContextData(`{"appId":" 620 ","gameName":"Portal 2"}`)
	require.True(t, ok)
	require.Equal(t, "620", data.AppID)
	require.Equal(t, "Portal 2", data.GameName)
}

func TestParseGameCardContextDataRejectsInvalidInput(t *testing.T) {
	for _, value := range []string{"", `{"appId":"not-an-app-id"}`, `{"gameName":"Portal 2"}`} {
		_, ok := parseGameCardContextData(value)
		require.False(t, ok, value)
	}
}
