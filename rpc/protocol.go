package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const (
	logBodyLimit        = 512
	maxRPCResponseBytes = 4 << 20

	rpcSessionFieldKey  = "ubus_rpc_session"
	jsonRPCVersion      = "2.0"
	jsonRPCMethodCall   = "call"
	contentTypeJSON     = "application/json"
	jsonRPCAccessDenied = -32002
)

type jsonRPCError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type jsonRPCEnvelope struct {
	Result  any           `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
}

type httpStatusError struct {
	status     string
	statusCode int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP status %d %s", e.statusCode, e.status)
}

func (e *httpStatusError) Unwrap() error {
	return goubus.ErrConnectionFailed
}

func (c *Client) rawCallWithMode(
	ctx context.Context,
	sessionID, service, method string,
	data any,
) (goubus.Result, bool, error) {
	payload, explicitSession, err := c.callRequest(sessionID, service, method, data)
	if err != nil {
		return nil, false, err
	}

	c.logger.Debug("ubus RPC request",
		slog.Uint64("id", c.id.Load()),
		slog.String("service", service),
		slog.String("method", method),
		slog.String("body", "<redacted>"))
	body, err := c.post(ctx, payload, service == "session")
	if err != nil {
		return nil, explicitSession, err
	}

	res, err := c.parseUbusResponse(body, service, method)

	return res, explicitSession, err
}

func (c *Client) callRequest(sessionID, service, method string, data any) ([]byte, bool, error) {
	rawData, err := marshalCallData(data)
	if err != nil {
		return nil, false, err
	}
	var explicitSession bool
	sessionID, rawData, explicitSession, err = hoistRPCSessionID(sessionID, rawData)
	if err != nil {
		return nil, false, errs.Wrapf(goubus.ErrInvalidParameter, "normalize RPC session parameter: %v", err)
	}

	body, err := c.requestBody(jsonRPCMethodCall, []any{sessionID, service, method, rawData})

	return body, explicitSession, err
}

func (c *Client) requestBody(method string, params any) ([]byte, error) {
	id := c.id.Add(1)
	payload := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      id,
		"method":  method,
		"params":  params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidParameter, "marshal JSON-RPC request: %v", err)
	}

	return body, nil
}

func (c *Client) post(ctx context.Context, payload []byte, redactResponse bool) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrConnectionFailed, "create request: %v", err)
	}
	request.Header.Set("Content-Type", contentTypeJSON)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrConnectionFailed, "HTTP POST: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &httpStatusError{
			statusCode: response.StatusCode,
			status:     http.StatusText(response.StatusCode),
		}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxRPCResponseBytes+1))
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "read response: %v", err)
	}
	if len(body) > maxRPCResponseBytes {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "response exceeds %d bytes", maxRPCResponseBytes)
	}
	responseBody := previewText(body, logBodyLimit)
	if redactResponse {
		responseBody = "<redacted>"
	}
	c.logger.Debug("ubus RPC response",
		slog.String("status", response.Status),
		slog.String("body", responseBody))

	return body, nil
}

func (c *Client) parseUbusResponse(body []byte, service, method string) (goubus.Result, error) {
	var response jsonRPCEnvelope
	err := json.Unmarshal(body, &response)
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "JSON decode error: %v", err)
	}
	if response.Error != nil {
		return nil, &goubus.CallError{
			Err:     mapJSONRPCError(response.Error.Code),
			Service: service,
			Method:  method,
			Message: response.Error.Message,
			Code:    response.Error.Code,
		}
	}

	result, ok := response.Result.([]any)
	if !ok || len(result) == 0 {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "expected ubus result array, got %T", response.Result)
	}
	status, ok := result[0].(float64)
	if !ok || status != float64(int(status)) {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "expected numeric ubus status, got %T", result[0])
	}
	code := int(status)
	if code != goubus.StatusOK {
		return nil, &goubus.CallError{Err: goubus.StatusError(code), Service: service, Method: method, Code: code}
	}
	if len(result) == 1 {
		return goubus.NewJSONResult(nil), nil
	}

	data, err := json.Marshal(result[1])
	if err != nil {
		return nil, errs.Wrapf(goubus.ErrInvalidResponse, "marshal RPC response data: %v", err)
	}

	return goubus.NewJSONResult(data), nil
}

func marshalCallData(data any) (json.RawMessage, error) {
	var rawData json.RawMessage
	switch value := data.(type) {
	case nil:
		return json.RawMessage(`{}`), nil
	case string:
		rawData = json.RawMessage(value)
	case []byte:
		rawData = json.RawMessage(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, errs.Wrapf(goubus.ErrInvalidParameter, "marshal RPC data: %v", err)
		}
		rawData = encoded
	}

	trimmed := bytes.TrimSpace(rawData)
	if !json.Valid(trimmed) {
		return nil, errs.Wrapf(goubus.ErrInvalidParameter, "RPC data is not valid JSON")
	}
	if bytes.Equal(trimmed, []byte("null")) {
		return json.RawMessage(`{}`), nil
	}
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, errs.Wrapf(goubus.ErrInvalidParameter, "RPC data must be a JSON object")
	}

	return trimmed, nil
}

func isRPCAccessDenied(err error) bool {
	callErr, ok := errors.AsType[*goubus.CallError](err)

	return ok && callErr.Code == jsonRPCAccessDenied
}

func normalizeEndpoint(host, endpointPath string) (string, error) {
	address := host
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return "", goubus.ErrInvalidParameter
	}
	if endpointPath != "" {
		parsed.Path = "/" + strings.TrimLeft(endpointPath, "/")
	} else if parsed.Path == "" {
		parsed.Path = DefaultEndpoint
	}

	return parsed.String(), nil
}

func hoistRPCSessionID(sessionID string, rawData json.RawMessage) (string, json.RawMessage, bool, error) {
	if !bytes.Contains(rawData, []byte(rpcSessionFieldKey)) {
		return sessionID, rawData, false, nil
	}

	var fields map[string]json.RawMessage
	err := json.Unmarshal(rawData, &fields)
	if err != nil {
		return "", nil, false, err
	}

	rawSession, hasKey := fields[rpcSessionFieldKey]
	if !hasKey {
		return sessionID, rawData, false, nil
	}

	var explicit string
	var hasExplicit bool
	err = json.Unmarshal(rawSession, &explicit)
	if err == nil && explicit != "" {
		sessionID = explicit
		hasExplicit = true
	}
	delete(fields, rpcSessionFieldKey)

	cleaned, err := json.Marshal(fields)
	if err != nil {
		return "", nil, false, err
	}

	return sessionID, cleaned, hasExplicit, nil
}

func mapJSONRPCError(code int) error {
	switch code {
	case -32000:
		return goubus.ErrNotFound
	case -32002:
		return goubus.ErrPermissionDenied
	case -32601:
		return goubus.ErrMethodNotFound
	case -32602:
		return goubus.ErrInvalidParameter
	case -32600:
		return goubus.ErrInvalidCommand
	default:
		return goubus.ErrUnknown
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
