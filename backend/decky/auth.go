package decky

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const (
	AuthTokenStdoutPrefix = "SENTINEL_DECKY_AUTH_TOKEN="
	AuthHeader            = "Authorization"
	AuthQueryParam        = "decky_auth_token"
)

func GenerateAuthToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func WriteAuthToken(w io.Writer, token string) error {
	_, err := fmt.Fprintf(w, "%s%s\n", AuthTokenStdoutPrefix, token)
	return err
}
