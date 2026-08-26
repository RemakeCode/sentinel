package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestHTTPAuthTransportUsesMultipartAndRequiresEResult(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.NoError(t, request.ParseMultipartForm(1<<20))
		encoded := request.FormValue("input_protobuf_encoded")
		_, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		response := appendVarint(nil, 1, 42)
		response = appendString(response, 2, "https://s.team/q/test")
		response = appendBytes(response, 3, []byte("request"))
		response = protowire.AppendTag(response, 4, protowire.Fixed32Type)
		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], math.Float32bits(5))
		response = append(response, raw[:]...)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"X-Eresult": []string{"1"}},
			Body:       io.NopCloser(bytes.NewReader(response)),
			Request:    request,
		}, nil
	})}
	transport := NewHTTPAuthTransport(client)
	transport.BaseURL = "https://steam.test"
	challenge, err := transport.Begin(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(42), challenge.ClientID)
	require.Equal(t, 5*time.Second, challenge.Interval)
}

type rotatingAuth struct{ polls int }

type failingAuth struct{}

func (failingAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{}, errors.New("transport failed")
}
func (failingAuth) Poll(context.Context, uint64, []byte) (AuthPoll, error) { return AuthPoll{}, nil }

func (a *rotatingAuth) Begin(context.Context) (AuthChallenge, error) {
	return AuthChallenge{ClientID: 1, RequestID: []byte("request"), ChallengeURL: "first", Interval: time.Millisecond}, nil
}

func (a *rotatingAuth) Poll(context.Context, uint64, []byte) (AuthPoll, error) {
	a.polls++
	if a.polls == 1 {
		return AuthPoll{NewClientID: 2, NewChallengeURL: "second", Interval: time.Millisecond}, nil
	}
	return AuthPoll{AccountName: "account", RefreshToken: "secret"}, nil
}

func TestAuthenticateHandlesChallengeRotation(t *testing.T) {
	transport := &rotatingAuth{}
	var challenges []string
	result, err := Authenticate(context.Background(), transport, func(challenge AuthChallenge) {
		challenges = append(challenges, challenge.ChallengeURL)
	})
	require.NoError(t, err)
	require.Equal(t, AuthResult{AccountName: "account", RefreshToken: "secret"}, result)
	require.Equal(t, []string{"first", "second"}, challenges)
}

func TestAuthenticateCancellationAndTransportFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Authenticate(ctx, &rotatingAuth{}, nil)
	require.ErrorIs(t, err, context.Canceled)
	_, err = Authenticate(context.Background(), failingAuth{}, nil)
	require.ErrorContains(t, err, "transport failed")
}

func TestWriteTokenFileUsesGSEMapAndOwnerOnlyPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh_tokens.json")
	require.NoError(t, writeGSETokenFile(path, "account", "secret"))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	var value map[string]string
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &value))
	require.Equal(t, map[string]string{"account": "secret"}, value)
}

func TestSteamQRAuthenticationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	challenge, err := NewHTTPAuthTransport(nil).Begin(ctx)
	require.NoError(t, err)
	require.NotZero(t, challenge.ClientID)
	require.NotEmpty(t, challenge.RequestID)
	require.Contains(t, challenge.ChallengeURL, "s.team/q/")
	require.Positive(t, challenge.Interval)
}
