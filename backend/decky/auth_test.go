package decky

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAuthToken(t *testing.T) {
	first, err := GenerateAuthToken()
	require.NoError(t, err)
	second, err := GenerateAuthToken()
	require.NoError(t, err)

	assert.NotEmpty(t, first)
	assert.NotEqual(t, first, second)
	assert.NotContains(t, first, "\n")
	assert.NotContains(t, first, "=")
}

func TestWriteAuthToken(t *testing.T) {
	var out strings.Builder
	require.NoError(t, WriteAuthToken(&out, "test-token"))

	assert.Equal(t, AuthTokenStdoutPrefix+"test-token\n", out.String())
}
