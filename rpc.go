// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package goubus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/honeybbq/goubus/v2/errdefs"
	"github.com/honeybbq/goubus/v2/internal/logging"
	"github.com/honeybbq/goubus/v2/internal/rpc"
)

const (
	logBodyLimit = 512
)

const (
	ubusAuthSessionID = "00000000000000000000000000000000"
)

const (
	jsonRPCVersion    = "2.0"
	jsonRPCMethodCall = "call"
)

const (
	contentTypeJSON  = "application/json"
	ubusEndpointPath = "/ubus"
)

// RpcClient handles communication with the ubus JSON-RPC endpoint.
// It manages authentication and session state internally.
type RpcClient struct {
	logger      *slog.Logger
	host        string
	username    string
	password    string
	sessionData rpc.SessionData
	id          int
	rwMutex     sync.RWMutex
	closed      bool
}

var _ Transport = (*RpcClient)(nil)

// RpcOption defines a functional option for an RpcClient.
type RpcOption func(*RpcClient)

// WithRpcLogger sets the logger for the RPC client.
func WithRpcLogger(logger *slog.Logger) RpcOption {
	return func(rc *RpcClient) {
		rc.SetLogger(logger)
	}
}

// NewRpcClient creates an authenticated RPC client.
func NewRpcClient(ctx context.Context, host, username, password string, opts ...RpcOption) (*RpcClient, error) {
	client := &RpcClient{
		host:     host,
		username: username,
		password: password,
		id:       1,
		logger:   logging.Discard(),
	}

	for _, opt := range opts {
		opt(client)
	}

	// Perform initial authentication
	err := client.authenticate(ctx)
	if err != nil {
		return nil, errdefs.Wrapf(err, "failed to authenticate")
	}

	return client, nil
}

// SetLogger sets the logger for the RPC client.
func (rc *RpcClient) SetLogger(logger *slog.Logger) {
	if logger == nil {
		rc.logger = logging.Discard()
	} else {
		rc.logger = logger
	}
}

// Call performs a JSON-RPC call with automatic session management.
func (rc *RpcClient) Call(ctx context.Context, service, method string, data any) (Result, error) {
	if rc.closed {
		return nil, errdefs.ErrClosed
	}

	// Get current session ID, re-authenticate if needed
	sessionID, err := rc.getValidSessionID(ctx)
	if err != nil {
		return nil, err
	}

	return rc.rawCall(ctx, sessionID, service, method, data)
}

func (rc *RpcClient) Close() error {
	rc.rwMutex.Lock()
	defer rc.rwMutex.Unlock()

	rc.closed = true
	if rc.sessionData.UbusRPCSession != "" {
		_, err := rc.rawCall(context.Background(), rc.sessionData.UbusRPCSession, "session", "destroy", nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// getValidSessionID returns a valid session ID.
func (rc *RpcClient) getValidSessionID(ctx context.Context) (string, error) {
	rc.rwMutex.RLock()

	// Check if current session is still valid
	if rc.sessionData.UbusRPCSession != "" && time.Now().Before(rc.sessionData.ExpireTime) {
		sessionID := rc.sessionData.UbusRPCSession
		rc.rwMutex.RUnlock()

		return sessionID, nil
	}

	rc.rwMutex.RUnlock()

	// Session expired or doesn't exist, re-authenticate
	err := rc.authenticate(ctx)
	if err != nil {
		return "", err
	}

	rc.rwMutex.RLock()
	sessionID := rc.sessionData.UbusRPCSession
	rc.rwMutex.RUnlock()

	return sessionID, nil
}

// authenticate with the ubus system.
func (rc *RpcClient) authenticate(ctx context.Context) error {
	rc.rwMutex.Lock()
	defer rc.rwMutex.Unlock()

	loginData := map[string]string{
		"username": rc.username,
		"password": rc.password,
	}

	// Use zero session ID for authentication
	resp, err := rc.rawCall(ctx, ubusAuthSessionID, "session", "login", loginData)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return errdefs.Wrapf(err, "ubus or ubus session module not installed")
		}

		return errdefs.Wrapf(err, "error calling auth login")
	}

	var sessionData rpc.SessionData

	err = resp.Unmarshal(&sessionData)
	if err != nil {
		return errdefs.Wrapf(err, "failed to parse session data")
	}

	// Calculate expire time
	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)

	rc.sessionData = sessionData

	return nil
}

// rawCall performs the actual JSON-RPC call without session management.
func (rc *RpcClient) rawCall(ctx context.Context, sessionID, service, method string, data any) (Result, error) {
	requestBody := rc.prepareRequestBody(sessionID, service, method, data)

	rc.logger.Debug("Request",
		slog.Int("id", rc.id),
		slog.String("service", service),
		slog.String("method", method),
		slog.String("body", requestBody))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://"+rc.host+ubusEndpointPath,
		bytes.NewBufferString(requestBody),
	)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "create request: %v", err)
	}

	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "http post error: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "read response: %v", err)
	}

	rc.logger.Debug("Response",
		slog.String("status", resp.Status),
		slog.String("body", previewText(bodyBytes, logBodyLimit)))

	return rc.parseUbusResponse(bodyBytes)
}

func (rc *RpcClient) prepareRequestBody(sessionID, service, method string, data any) string {
	var dataJSON string
	if data == nil {
		dataJSON = "{}"
	} else {
		switch v := data.(type) {
		case string:
			dataJSON = v
		case []byte:
			dataJSON = string(v)
		default:
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				dataJSON = "{}"
			} else {
				dataJSON = string(jsonBytes)
			}
		}
	}

	return fmt.Sprintf(`{
		"jsonrpc": "%s",
		"id": %d,
		"method": "%s",
		"params": [
			"%s",
			"%s",
			"%s",
			%s
		]
	}`,
		jsonRPCVersion,
		rc.id,
		jsonRPCMethodCall,
		sessionID,
		service,
		method,
		dataJSON,
	)
}

func (rc *RpcClient) parseUbusResponse(body []byte) (Result, error) {
	ubusResp := &rpc.UbusResponse{}

	err := json.Unmarshal(body, ubusResp)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "json decode error: %v", err)
	}

	if ubusResp.Error != nil {
		mappedErr := MapUbusCodeToError(ubusResp.Error.Code)

		return nil, errdefs.Wrapf(mappedErr, "json-rpc error: %s", ubusResp.Error.Message)
	}

	result, ok := ubusResp.Result.([]any)
	if !ok {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected array result, got %T", ubusResp.Result)
	}

	return rpcResult(result), nil
}

func previewText(bytes []byte, maxLen int) string {
	if len(bytes) == 0 {
		return ""
	}

	if len(bytes) > maxLen {
		return string(bytes[:maxLen]) + "..."
	}

	return string(bytes)
}

type rpcResult []any

func (r rpcResult) Unmarshal(target any) error {
	return rpc.UbusResult(r).Unmarshal(target, MapUbusCodeToError)
}
