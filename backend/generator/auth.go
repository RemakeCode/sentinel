package generator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

const steamAuthenticationEndpoint = "https://api.steampowered.com/IAuthenticationService/"

var qrApprovalTimeout = 5 * time.Minute // Temporary test timeout for the QR expiry presentation.

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
	body := encodeBeginRequest()
	result, err := t.post(ctx, "BeginAuthSessionViaQR", body, decodeBeginResponse)
	if err != nil {
		return AuthChallenge{}, err
	}

	challenge, ok := result.(AuthChallenge)
	if !ok {
		return AuthChallenge{}, errors.New("invalid Steam begin-auth response")
	}
	return challenge, nil
}

func (t *HTTPAuthTransport) Poll(ctx context.Context, clientID uint64, requestID []byte) (AuthPoll, error) {
	body := encodePollRequest(clientID, requestID)
	result, err := t.post(ctx, "PollAuthSessionStatus", body, decodePollResponse)
	if err != nil {
		return AuthPoll{}, err
	}

	poll, ok := result.(AuthPoll)
	if !ok {
		return AuthPoll{}, errors.New("invalid Steam poll-auth response")
	}
	return poll, nil
}

func (t *HTTPAuthTransport) post(ctx context.Context, method string, body []byte, decode func([]byte) (any, error)) (any, error) {
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	if err := writer.WriteField("input_protobuf_encoded", base64.StdEncoding.EncodeToString(body)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	contentType := writer.FormDataContentType()
	endpoint := strings.TrimRight(t.BaseURL, "/") + "/" + method + "/v1/"
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(form.Bytes()))
		if err != nil {
			return nil, err
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
				return nil, fmt.Errorf("steam authentication returned HTTP %d", resp.StatusCode)
			} else {
				eResultHeader := resp.Header.Get("X-EResult")
				eResult, parseErr := strconv.Atoi(eResultHeader)
				if parseErr != nil {
					return nil, fmt.Errorf("Steam authentication returned invalid X-EResult %q", eResultHeader)
				}
				if eResult != 1 {
					return nil, fmt.Errorf("Steam QR authentication was rejected or expired (EResult %d)", eResult)
				}
				if result, decodeErr := decode(data); decodeErr == nil {
					return result, nil
				}
				return decodeProtoResult(data, method)
			}
		}

		if attempt < 2 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
			}
		}
	}
	return nil, lastErr
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

func encodeBeginRequest() []byte {
	var body []byte
	body = appendString(body, 1, "Sentinel")
	body = appendVarint(body, 2, 1) // SteamClient
	var details []byte
	details = appendString(details, 1, "Sentinel")
	details = appendVarint(details, 2, 1)
	details = appendVarint(details, 3, 16)
	body = appendBytes(body, 3, details)
	body = appendString(body, 4, "Client")
	return body
}

func encodePollRequest(clientID uint64, requestID []byte) []byte {
	var body []byte
	body = appendVarint(body, 1, clientID)
	body = appendBytes(body, 2, requestID)
	return body
}

func appendString(dst []byte, field protowire.Number, value string) []byte {
	return appendBytes(dst, field, []byte(value))
}

func appendBytes(dst []byte, field protowire.Number, value []byte) []byte {
	dst = protowire.AppendTag(dst, field, protowire.BytesType)
	dst = protowire.AppendBytes(dst, value)
	return dst
}

func appendVarint(dst []byte, field protowire.Number, value uint64) []byte {
	dst = protowire.AppendTag(dst, field, protowire.VarintType)
	dst = protowire.AppendVarint(dst, value)
	return dst
}

func decodeBeginResponse(data []byte) (any, error) {
	response, err := jsonResponse(data)
	if err != nil {
		return nil, err
	}

	var result struct {
		ClientID     uint64  `json:"client_id"`
		ChallengeURL string  `json:"challenge_url"`
		RequestID    string  `json:"request_id"`
		Interval     float64 `json:"interval"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	requestID, err := decodeRequestID(result.RequestID)
	if err != nil {
		return nil, err
	}
	return AuthChallenge{ClientID: result.ClientID, ChallengeURL: result.ChallengeURL, RequestID: requestID, Interval: secondsDuration(result.Interval)}, nil
}

func decodePollResponse(data []byte) (any, error) {
	response, err := jsonResponse(data)
	if err != nil {
		return nil, err
	}

	var result struct {
		NewClientID     uint64  `json:"new_client_id"`
		NewChallengeURL string  `json:"new_challenge_url"`
		RefreshToken    string  `json:"refresh_token"`
		AccountName     string  `json:"account_name"`
		Interval        float64 `json:"interval"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}

	return AuthPoll{NewClientID: result.NewClientID, NewChallengeURL: result.NewChallengeURL, RefreshToken: result.RefreshToken, AccountName: result.AccountName, Interval: secondsDuration(result.Interval)}, nil
}

func jsonResponse(data []byte) ([]byte, error) {
	var envelope struct {
		Response json.RawMessage `json:"response"`
		EResult  int             `json:"eresult"`
		Error    string          `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil || len(envelope.Response) == 0 {
		return nil, errors.New("response is not JSON")
	}

	if envelope.EResult != 0 && envelope.EResult != 1 {
		return nil, fmt.Errorf("Steam authentication returned EResult %d", envelope.EResult)
	}
	if envelope.Error != "" {
		return nil, errors.New("Steam authentication was rejected")
	}

	return envelope.Response, nil
}

func decodeRequestID(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("Steam returned an empty request ID")
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return []byte(value), nil
}

func secondsDuration(seconds float64) time.Duration {
	if seconds <= 0 || math.IsNaN(seconds) {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func decodeProtoResult(data []byte, method string) (any, error) {
	if method == "BeginAuthSessionViaQR" {
		return decodeBeginProto(data)
	}

	return decodePollProto(data)
}

func decodeBeginProto(data []byte) (AuthChallenge, error) {
	var result AuthChallenge
	if err := consumeProto(data, func(field protowire.Number, typ protowire.Type, value []byte, number uint64, floatValue float32) error {
		switch field {
		case 1:
			result.ClientID = number
		case 2:
			result.ChallengeURL = string(value)
		case 3:
			result.RequestID = append([]byte(nil), value...)
		case 4:
			result.Interval = time.Duration(float64(floatValue) * float64(time.Second))
		}
		return nil
	}); err != nil {
		return AuthChallenge{}, err
	}

	return result, nil
}

func decodePollProto(data []byte) (AuthPoll, error) {
	var result AuthPoll
	if err := consumeProto(data, func(field protowire.Number, typ protowire.Type, value []byte, number uint64, _ float32) error {
		switch field {
		case 1:
			result.NewClientID = number
		case 2:
			result.NewChallengeURL = string(value)
		case 3:
			result.RefreshToken = string(value)
		case 6:
			result.AccountName = string(value)
		}
		return nil
	}); err != nil {
		return AuthPoll{}, err
	}

	return result, nil
}

func consumeProto(data []byte, callback func(protowire.Number, protowire.Type, []byte, uint64, float32) error) error {
	for len(data) > 0 {
		number, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return protowire.ParseError(n)
		}

		data = data[n:]
		var value []byte
		var numberValue uint64
		var floatValue float32
		switch typ {
		case protowire.VarintType:
			numberValue, n = protowire.ConsumeVarint(data)
		case protowire.Fixed32Type:
			var raw uint32
			raw, n = protowire.ConsumeFixed32(data)
			floatValue = math.Float32frombits(raw)
		case protowire.Fixed64Type:
			_, n = protowire.ConsumeFixed64(data)
		case protowire.BytesType:
			value, n = protowire.ConsumeBytes(data)
		default:
			return fmt.Errorf("unsupported protobuf wire type %d", typ)
		}
		if n < 0 {
			return protowire.ParseError(n)
		}

		if err := callback(number, typ, value, numberValue, floatValue); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
