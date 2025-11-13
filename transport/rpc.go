package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// JSON-RPC related constants, moved here as they are internal implementation details.
const (
	jsonRPCVersion    = "2.0"
	jsonRPCMethodCall = "call"
)

// HTTP related constants
const (
	contentTypeJSON  = "application/json"
	ubusEndpointPath = "/ubus"
)

// SessionData holds authentication session information
type SessionData struct {
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
	ExpireTime     time.Time `json:"-"`
}

// RpcClient handles the low-level communication with the ubus JSON-RPC endpoint.
// It manages authentication and session state internally.
type RpcClient struct {
	host     string
	username string
	password string
	id       int
	debug    atomic.Bool

	// Session management
	sessionData SessionData
	rwMutex     sync.RWMutex
	closed      bool
}

var _ types.Transport = (*RpcClient)(nil)

// NewRpcClient creates a new authenticated RPC client.
func NewRpcClient(host, username, password string) (*RpcClient, error) {
	client := &RpcClient{
		host:     host,
		username: username,
		password: password,
		id:       1,
	}

	// Perform initial authentication
	if err := client.authenticate(); err != nil {
		return nil, errdefs.Wrapf(err, "failed to authenticate")
	}

	return client, nil
}

// SetDebug toggles verbose request/response logging.
func (rc *RpcClient) SetDebug(debug bool) {
	rc.debug.Store(debug)
}

func (rc *RpcClient) debugf(format string, args ...any) {
	if rc.debug.Load() {
		fmt.Printf("[rpc] "+format+"\n", args...)
	}
}

// Call performs a JSON-RPC call with automatic session management.
func (rc *RpcClient) Call(service, method string, data any) (types.Result, error) {
	if rc.closed {
		return nil, errdefs.ErrClosed
	}

	// Get current session ID, re-authenticate if needed
	sessionID, err := rc.getValidSessionID()
	if err != nil {
		return nil, err
	}

	return rc.rawCall(sessionID, service, method, data)
}

func (rc *RpcClient) Close() error {
	rc.rwMutex.Lock()
	defer rc.rwMutex.Unlock()

	rc.closed = true
	if rc.sessionData.UbusRPCSession != "" {
		if _, err := rc.rawCall(rc.sessionData.UbusRPCSession, "session", "destroy", nil); err != nil {
			return err
		}
	}
	return nil
}

// getValidSessionID returns a valid session ID, re-authenticating if necessary.
func (rc *RpcClient) getValidSessionID() (string, error) {
	rc.rwMutex.RLock()

	// Check if current session is still valid
	if rc.sessionData.UbusRPCSession != "" && time.Now().Before(rc.sessionData.ExpireTime) {
		sessionID := rc.sessionData.UbusRPCSession
		rc.rwMutex.RUnlock()
		return sessionID, nil
	}

	rc.rwMutex.RUnlock()

	// Session expired or doesn't exist, re-authenticate
	if err := rc.authenticate(); err != nil {
		return "", err
	}

	rc.rwMutex.RLock()
	sessionID := rc.sessionData.UbusRPCSession
	rc.rwMutex.RUnlock()

	return sessionID, nil
}

// authenticate performs authentication with the ubus system.
func (rc *RpcClient) authenticate() error {
	rc.rwMutex.Lock()
	defer rc.rwMutex.Unlock()

	loginData := map[string]string{
		"username": rc.username,
		"password": rc.password,
	}

	// Use zero session ID for authentication
	resp, err := rc.rawCall("00000000000000000000000000000000", "session", "login", loginData)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return errdefs.Wrapf(err, "ubus or ubus session module not installed")
		}
		return errdefs.Wrapf(err, "error calling auth login")
	}

	var sessionData SessionData
	if err := resp.Unmarshal(&sessionData); err != nil {
		return errdefs.Wrapf(err, "failed to parse session data")
	}

	// Calculate expire time
	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)

	rc.sessionData = sessionData
	return nil
}

// rawCall performs the actual JSON-RPC call without session management.
func (rc *RpcClient) rawCall(sessionID, service, method string, data any) (types.Result, error) {
	var dataJSON string
	if data == nil {
		dataJSON = "{}"
	} else {
		// This switch is an optimization from the original code for common types.
		switch v := data.(type) {
		case string:
			dataJSON = v
		case []byte:
			dataJSON = string(v)
		default:
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				// Fallback to empty object if marshaling fails
				dataJSON = "{}"
			} else {
				dataJSON = string(jsonBytes)
			}
		}
	}

	// Using fmt.Sprintf for simple JSON templating, as in the original code.
	requestBody := fmt.Sprintf(`{
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
		dataJSON)

	rc.debugf("Request: id=%d service=%s method=%s body=%s", rc.id, service, method, requestBody)

	resp, err := http.Post(
		"http://"+rc.host+ubusEndpointPath,
		contentTypeJSON,
		bytes.NewBuffer([]byte(requestBody)),
	)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrConnectionFailed, "http post error: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "read response: %v", err)
	}
	rc.debugf("Response: status=%s body=%s", resp.Status, previewText(bodyBytes, 512))

	ubusResp := &UbusResponse{}
	if err := json.Unmarshal(bodyBytes, ubusResp); err != nil {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "json decode error: %v", err)
	}

	// Check for JSON-RPC level error (e.g., method not found)
	if ubusResp.Error != nil {
		// Attempt to map the JSON-RPC error code to our internal error types
		mappedErr := mapUbusCodeToError(ubusResp.Error.Code)
		return nil, errdefs.Wrapf(mappedErr, "json-rpc error: %s", ubusResp.Error.Message)
	}

	result, ok := ubusResp.Result.([]any)
	if !ok {
		return nil, errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected array result, got %T", ubusResp.Result)
	}
	// Check for ubus application level error (handled by Unmarshal)
	return UbusResult(result), nil
}

func previewText(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}

// UbusJsonRpcError represents the error structure in a JSON-RPC response.
type UbusJsonRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// UbusResponse represents a response from the ubus RPC interface.
type UbusResponse struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      int               `json:"id"`
	Error   *UbusJsonRpcError `json:"error,omitempty"`
	Result  any               `json:"result,omitempty"`
}

type UbusResult []any

func (r UbusResult) Unmarshal(target any) error {
	if len(r) < 1 {
		return errdefs.ErrInvalidResponse
	}

	// Check the error code (first element)
	code, ok := r[0].(float64)
	if !ok {
		return errdefs.Wrapf(errdefs.ErrInvalidResponse, "expected numeric error code, got %T", r[0])
	}

	// If there's an error code, map it to a typed error
	if code != 0 {
		return mapUbusCodeToError(int(code))
	}

	// If there's only one element and it's 0, it means success but no data
	if len(r) == 1 {
		return errdefs.ErrNoData
	}

	// If there are 2+ elements, the second element contains the data
	if len(r) >= 2 {
		// The actual data is the second element of the result array
		ubusDataByte, err := json.Marshal(r[1])
		if err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidResponse, "failed to marshal response data: %v", err)
		}
		if err := json.Unmarshal(ubusDataByte, target); err != nil {
			return errdefs.Wrapf(errdefs.ErrInvalidResponse, "failed to unmarshal response data: %v", err)
		}
		return nil
	}

	return errdefs.ErrInvalidResponse
}

// mapUbusCodeToError maps a ubus integer code to a typed error.
func mapUbusCodeToError(code int) error {
	switch code {
	case 0:
		return nil
	case 1:
		return errdefs.ErrInvalidCommand
	case 2:
		return errdefs.ErrInvalidParameter
	case 3:
		return errdefs.ErrMethodNotFound
	case 4:
		return errdefs.ErrNotFound
	case 5:
		return errdefs.ErrNoData
	case 6:
		return errdefs.ErrPermissionDenied
	case 7:
		return errdefs.ErrTimeout
	case 8:
		return errdefs.ErrNotSupported
	case 9:
		return errdefs.ErrUnknown
	case 10:
		return errdefs.ErrConnectionFailed
	default:
		return errdefs.Wrapf(errdefs.ErrUnknown, "unknown ubus error code: %d", code)
	}
}
