package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
)

type AuthChallenge struct {
	ClientID     uint64
	ChallengeURL string
	RequestID    []byte
	Interval     time.Duration
}

type AuthPoll struct {
	NewClientID     uint64
	NewChallengeURL string
	RefreshToken    string
	AccountName     string
	Interval        time.Duration
}

type AuthResult struct {
	AccountName  string
	RefreshToken string
}

type AuthTransport interface {
	Begin(context.Context) (AuthChallenge, error)
	Poll(context.Context, uint64, []byte) (AuthPoll, error)
}

type HTTPAuthTransport struct {
	Client  *http.Client
	BaseURL string
}

func NewHTTPAuthTransport(client *http.Client) *HTTPAuthTransport {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &HTTPAuthTransport{Client: client, BaseURL: steamAuthenticationEndpoint}
}

func (t *HTTPAuthTransport) Begin(ctx context.Context) (AuthChallenge, error) {
	body, err := encodeBeginRequest()
	if err != nil {
		return AuthChallenge{}, err
	}

	response := new(BeginAuthSessionViaQRResponse)
	if err := t.post(ctx, "BeginAuthSessionViaQR", body, response); err != nil {
		return AuthChallenge{}, err
	}

	return AuthChallenge{
		ClientID:     response.GetClientId(),
		ChallengeURL: response.GetChallengeUrl(),
		RequestID:    response.GetRequestId(),
		Interval:     time.Duration(response.GetInterval() * float32(time.Second)),
	}, nil
}

func (t *HTTPAuthTransport) Poll(ctx context.Context, clientID uint64, requestID []byte) (AuthPoll, error) {
	body, err := encodePollRequest(clientID, requestID)
	if err != nil {
		return AuthPoll{}, err
	}

	response := new(PollAuthSessionStatusResponse)
	if err := t.post(ctx, "PollAuthSessionStatus", body, response); err != nil {
		return AuthPoll{}, err
	}

	return AuthPoll{
		NewClientID:     response.GetNewClientId(),
		NewChallengeURL: response.GetNewChallengeUrl(),
		RefreshToken:    response.GetRefreshToken(),
		AccountName:     response.GetAccountName(),
	}, nil
}

func (t *HTTPAuthTransport) post(ctx context.Context, method string, body []byte, response proto.Message) error {
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	if err := writer.WriteField("input_protobuf_encoded", base64.StdEncoding.EncodeToString(body)); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	contentType := writer.FormDataContentType()
	endpoint := strings.TrimRight(t.BaseURL, "/") + "/" + method + "/v1/"
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(form.Bytes()))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", contentType)
		resp, err := t.Client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("steam authentication returned HTTP %d", resp.StatusCode)
			} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("steam authentication returned HTTP %d", resp.StatusCode)
			} else {
				eResultHeader := resp.Header.Get("X-EResult")
				eResult, parseErr := strconv.Atoi(eResultHeader)
				if parseErr != nil {
					return fmt.Errorf("Steam authentication returned invalid X-EResult %q", eResultHeader)
				}
				if eResult != 1 {
					return fmt.Errorf("Steam QR authentication was rejected or expired (EResult %d)", eResult)
				}
				return proto.Unmarshal(data, response)
			}
		}

		if attempt < 2 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
			}
		}
	}
	return lastErr
}

func Authenticate(ctx context.Context, transport AuthTransport, onChallenge func(AuthChallenge)) (AuthResult, error) {
	if transport == nil {
		return AuthResult{}, errors.New("authentication transport is unavailable")
	}

	challenge, err := transport.Begin(ctx)
	if err != nil {
		return AuthResult{}, fmt.Errorf("begin Steam QR authentication: %w", err)
	}

	if challenge.ClientID == 0 || len(challenge.RequestID) == 0 || challenge.ChallengeURL == "" {
		return AuthResult{}, errors.New("Steam returned an incomplete QR authentication challenge")
	}

	if challenge.Interval <= 0 {
		challenge.Interval = 5 * time.Second
	}

	approvalContext, cancelApproval := context.WithTimeout(ctx, qrApprovalTimeout)
	defer cancelApproval()

	if onChallenge != nil {
		onChallenge(challenge)
	}

	for {
		select {
		case <-approvalContext.Done():
			return AuthResult{}, approvalContext.Err()
		case <-time.After(challenge.Interval):
		}
		poll, err := transport.Poll(approvalContext, challenge.ClientID, challenge.RequestID)
		if err != nil {
			return AuthResult{}, fmt.Errorf("poll Steam QR authentication: %w", err)
		}

		if err := approvalContext.Err(); err != nil {
			return AuthResult{}, err
		}

		if poll.NewClientID != 0 {
			challenge.ClientID = poll.NewClientID
		}
		if poll.NewChallengeURL != "" {
			challenge.ChallengeURL = poll.NewChallengeURL
			if onChallenge != nil {
				onChallenge(challenge)
			}
		}
		if poll.Interval > 0 {
			challenge.Interval = poll.Interval
		}
		if poll.RefreshToken != "" {
			return AuthResult{AccountName: poll.AccountName, RefreshToken: poll.RefreshToken}, nil
		}
	}
}

//go:generate protoc --go_out=. --go_opt=paths=source_relative steam_auth.proto

func encodeBeginRequest() ([]byte, error) {
	return proto.Marshal(&BeginAuthSessionViaQRRequest{
		DeviceFriendlyName: proto.String("Sentinel"),
		PlatformType:       EAuthTokenPlatformType_SteamClient.Enum(),
		DeviceDetails: &DeviceDetails{
			DeviceFriendlyName: proto.String("Sentinel"),
			PlatformType:       EAuthTokenPlatformType_SteamClient.Enum(),
			OsType:             proto.Int32(16),
		},
		WebsiteId: proto.String("Client"),
	})
}

func encodePollRequest(clientID uint64, requestID []byte) ([]byte, error) {
	return proto.Marshal(&PollAuthSessionStatusRequest{
		ClientId:  proto.Uint64(clientID),
		RequestId: requestID,
	})
}
