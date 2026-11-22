package goubus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/honeybbq/goubus/v2/internal/errs"
	"github.com/honeybbq/goubus/v2/internal/logging"
	"github.com/honeybbq/goubus/v2/internal/rpc"
)

const (
	logBodyLimit      = 512
	defaultRPCTimeout = 30 * time.Second
)

const (
	ubusAuthSessionID = "00000000000000000000000000000000"
	jsonRPCVersion    = "2.0"
	jsonRPCMethodCall = "call"
	contentTypeJSON   = "application/json"
	ubusEndpointPath  = "/ubus"
)

// RpcClient calls ubus through the JSON-RPC endpoint and manages its session.
type RpcClient struct {
	logger      *slog.Logger
	httpClient  *http.Client
	host        string
	username    string
	password    string
	endpoint    string
	sessionData rpc.SessionData
	rwMutex     sync.RWMutex
	id          atomic.Uint64
	closed      bool
}

var _ Transport = (*RpcClient)(nil)
var _ Lister = (*RpcClient)(nil)

type rpcOptionFunc func(*RpcClient)

func (f rpcOptionFunc) apply(client *RpcClient) {
	f(client)
}

// RpcOption configures an RpcClient.
type RpcOption interface {
	apply(client *RpcClient)
}

// WithRpcLogger configures the RPC client's logger.
func WithRpcLogger(logger *slog.Logger) RpcOption {
	return rpcOptionFunc(func(client *RpcClient) {
		if logger == nil {
			client.logger = logging.Discard()

			return
		}
		client.logger = logger
	})
}

// WithHTTPClient replaces the HTTP client used by RpcClient.
func WithHTTPClient(client *http.Client) RpcOption {
	return rpcOptionFunc(func(rpcClient *RpcClient) {
		rpcClient.httpClient = client
	})
}

// WithRpcEndpoint sets the path used for JSON-RPC requests.
func WithRpcEndpoint(path string) RpcOption {
	return rpcOptionFunc(func(client *RpcClient) {
		client.endpoint = path
	})
}

// NewRpcClient creates an authenticated RPC client.
func NewRpcClient(ctx context.Context, host, username, password string, opts ...RpcOption) (*RpcClient, error) {
	client := &RpcClient{
		host:       host,
		username:   username,
		password:   password,
		logger:     logging.Discard(),
		httpClient: &http.Client{Timeout: defaultRPCTimeout},
	}

	for _, opt := range opts {
		if opt != nil {
			opt.apply(client)
		}
	}
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: defaultRPCTimeout}
	}

	endpoint, err := normalizeRPCEndpoint(host, client.endpoint)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "RPC endpoint: %v", err)
	}
	client.endpoint = endpoint

	err = client.authenticate(ctx)
	if err != nil {
		return nil, errs.Wrapf(err, "failed to authenticate")
	}

	return client, nil
}

func normalizeRPCEndpoint(host, endpointPath string) (string, error) {
	address := host
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return "", ErrInvalidParameter
	}
	if endpointPath != "" {
		parsed.Path = "/" + strings.TrimLeft(endpointPath, "/")
	} else if parsed.Path == "" {
		parsed.Path = ubusEndpointPath
	}

	return parsed.String(), nil
}

// Call invokes one method using the current ubus session.
func (client *RpcClient) Call(ctx context.Context, service, method string, data any) (Result, error) {
	client.rwMutex.RLock()
	closed := client.closed
	client.rwMutex.RUnlock()
	if closed {
		return nil, ErrClosed
	}

	sessionID, err := client.getValidSessionID(ctx)
	if err != nil {
		return nil, err
	}

	return client.rawCall(ctx, sessionID, service, method, data)
}

// Close destroys the current session and clears its credentials.
func (client *RpcClient) Close() error {
	client.rwMutex.Lock()
	defer client.rwMutex.Unlock()

	if client.closed {
		return nil
	}
	client.closed = true
	sessionID := client.sessionData.UbusRPCSession
	client.sessionData = rpc.SessionData{}
	if sessionID == "" {
		return nil
	}

	_, err := client.rawCall(context.Background(), sessionID, "session", "destroy", nil)

	return err
}

// String formats the client without exposing its password or session ID.
func (client *RpcClient) String() string {
	client.rwMutex.RLock()
	defer client.rwMutex.RUnlock()

	return fmt.Sprintf("RpcClient{host:%q, username:%q, password:%q, session:%q}",
		client.host, client.username, "<redacted>", "<redacted>")
}

// GoString formats the client for Go-syntax formatting without exposing secrets.
func (client *RpcClient) GoString() string {
	return client.String()
}

// LogValue returns structured client attributes with credentials redacted.
func (client *RpcClient) LogValue() slog.Value {
	client.rwMutex.RLock()
	defer client.rwMutex.RUnlock()

	return slog.GroupValue(
		slog.String("host", client.host),
		slog.String("username", client.username),
		slog.String("password", "<redacted>"),
		slog.Group("sessionData",
			slog.String("ubus_rpc_session", "<redacted>"),
			slog.Int("timeout", client.sessionData.Timeout),
			slog.Time("expire_time", client.sessionData.ExpireTime),
		),
	)
}

func (client *RpcClient) getValidSessionID(ctx context.Context) (string, error) {
	client.rwMutex.RLock()
	if client.sessionData.UbusRPCSession != "" && time.Now().Before(client.sessionData.ExpireTime) {
		sessionID := client.sessionData.UbusRPCSession
		client.rwMutex.RUnlock()

		return sessionID, nil
	}
	client.rwMutex.RUnlock()

	err := client.authenticate(ctx)
	if err != nil {
		return "", err
	}

	client.rwMutex.RLock()
	sessionID := client.sessionData.UbusRPCSession
	client.rwMutex.RUnlock()

	return sessionID, nil
}

func (client *RpcClient) authenticate(ctx context.Context) error {
	client.rwMutex.Lock()
	defer client.rwMutex.Unlock()

	loginData := map[string]string{"username": client.username, "password": client.password}
	result, err := client.rawCall(ctx, ubusAuthSessionID, "session", "login", loginData)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return errs.Wrapf(err, "ubus or ubus session module not installed")
		}

		return errs.Wrapf(err, "error calling auth login")
	}

	var sessionData rpc.SessionData
	err = result.Unmarshal(&sessionData)
	if err != nil {
		return errs.Wrapf(err, "failed to parse session data")
	}
	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)
	client.sessionData = sessionData

	return nil
}

func (client *RpcClient) rawCall(ctx context.Context, sessionID, service, method string, data any) (Result, error) {
	payload, err := client.callRequest(sessionID, service, method, data)
	if err != nil {
		return nil, err
	}

	client.logger.Debug("ubus RPC request",
		slog.Uint64("id", client.id.Load()),
		slog.String("service", service),
		slog.String("method", method),
		slog.String("body", "<redacted>"))
	body, err := client.post(ctx, payload, service == "session")
	if err != nil {
		return nil, err
	}

	return client.parseUbusResponse(body, service, method)
}

func (client *RpcClient) callRequest(sessionID, service, method string, data any) ([]byte, error) {
	var rawData json.RawMessage
	switch value := data.(type) {
	case nil:
		rawData = json.RawMessage(`{}`)
	case string:
		rawData = json.RawMessage(value)
	case []byte:
		rawData = json.RawMessage(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, errs.Wrapf(ErrInvalidParameter, "marshal RPC data: %v", err)
		}
		rawData = encoded
	}
	if !json.Valid(rawData) {
		return nil, errs.Wrapf(ErrInvalidParameter, "RPC data is not valid JSON")
	}

	return client.requestBody(jsonRPCMethodCall, []any{sessionID, service, method, rawData})
}

func (client *RpcClient) requestBody(method string, params any) ([]byte, error) {
	id := client.id.Add(1)
	payload := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"method":  method,
		"params":  params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidParameter, "marshal JSON-RPC request: %v", err)
	}

	return body, nil
}

func (client *RpcClient) post(ctx context.Context, payload []byte, redactResponse bool) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, errs.Wrapf(ErrConnectionFailed, "create request: %v", err)
	}
	request.Header.Set("Content-Type", contentTypeJSON)

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, errs.Wrapf(ErrConnectionFailed, "HTTP POST: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidResponse, "read response: %v", err)
	}
	responseBody := previewText(body, logBodyLimit)
	if redactResponse {
		responseBody = "<redacted>"
	}
	client.logger.Debug("ubus RPC response",
		slog.String("status", response.Status),
		slog.String("body", responseBody))

	return body, nil
}

func (client *RpcClient) parseUbusResponse(body []byte, service, method string) (Result, error) {
	var response rpc.UbusResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidResponse, "JSON decode error: %v", err)
	}
	if response.Error != nil {
		return nil, &CallError{
			Err:     mapJSONRPCError(response.Error.Code),
			Service: service,
			Method:  method,
			Message: response.Error.Message,
			Code:    response.Error.Code,
		}
	}

	result, ok := response.Result.([]any)
	if !ok || len(result) == 0 {
		return nil, errs.Wrapf(ErrInvalidResponse, "expected ubus result array, got %T", response.Result)
	}
	status, ok := result[0].(float64)
	if !ok || status != float64(int(status)) {
		return nil, errs.Wrapf(ErrInvalidResponse, "expected numeric ubus status, got %T", result[0])
	}
	code := int(status)
	if code != StatusOK {
		return nil, &CallError{Err: StatusError(code), Service: service, Method: method, Code: code}
	}
	if len(result) == 1 {
		return NewJSONResult(nil), nil
	}

	data, err := json.Marshal(result[1])
	if err != nil {
		return nil, errs.Wrapf(ErrInvalidResponse, "marshal RPC response data: %v", err)
	}

	return NewJSONResult(data), nil
}

func mapJSONRPCError(code int) error {
	switch code {
	case -32000:
		return ErrNotFound
	case -32002:
		return ErrPermissionDenied
	case -32601:
		return ErrMethodNotFound
	case -32602:
		return ErrInvalidParameter
	case -32600:
		return ErrInvalidCommand
	default:
		return ErrUnknown
	}
}

func previewText(data []byte, maxLen int) string {
	if len(data) == 0 {
		return ""
	}
	if len(data) > maxLen {
		return string(data[:maxLen]) + "..."
	}

	return string(data)
}
