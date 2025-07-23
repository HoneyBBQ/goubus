package api

import (
	"time"

	"github.com/honeybbq/goubus/errdefs"
	"github.com/honeybbq/goubus/types"
)

// Session service constants
const (
	ServiceSession       = "session"
	SessionMethodLogin   = "login"
	SessionMethodCreate  = "create"
	SessionMethodDestroy = "destroy"
	SessionMethodList    = "list"
	SessionMethodGrant   = "grant"
	SessionMethodRevoke  = "revoke"
	SessionMethodAccess  = "access"
	SessionMethodSet     = "set"
	SessionMethodGet     = "get"
	SessionMethodUnset   = "unset"
)

// Session parameter constants
const (
	SessionParamUsername = "username"
	SessionParamPassword = "password"
	SessionParamSession  = "session"
	SessionParamScope    = "scope"
	SessionParamObjects  = "objects"
	SessionParamAccess   = "access"
	SessionParamKeys     = "keys"
	SessionParamTimeout  = "timeout"
)

// CreateSession creates a new anonymous session.
func CreateSession(caller types.Transport, timeout int) (*types.SessionData, error) {
	params := map[string]any{SessionParamTimeout: timeout}
	resp, err := caller.Call(ServiceSession, SessionMethodCreate, params)
	if err != nil {
		return nil, err
	}
	return parseSessionData(resp)
}

// DestroySession destroys a session.
func DestroySession(caller types.Transport, sessionID string) error {
	_, err := caller.Call(ServiceSession, SessionMethodDestroy, nil)
	return err
}

// ListSessions lists all active sessions.
func ListSessions(caller types.Transport, sessionID string) (map[string]any, error) {
	params := map[string]any{}
	resp, err := caller.Call(ServiceSession, SessionMethodList, params)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := resp.Unmarshal(&result); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal session list response")
	}
	return result, nil
}

// GrantSessionAccess grants access to objects for a session.
func GrantSessionAccess(caller types.Transport, sessionID string, scope string, objects []string) error {
	params := map[string]any{
		SessionParamScope:   scope,
		SessionParamObjects: objects,
	}
	_, err := caller.Call(ServiceSession, SessionMethodGrant, params)
	return err
}

// RevokeSessionAccess revokes access to objects for a session.
func RevokeSessionAccess(caller types.Transport, sessionID string, scope string, objects []string) error {
	params := map[string]any{
		SessionParamScope:   scope,
		SessionParamObjects: objects,
	}
	_, err := caller.Call(ServiceSession, SessionMethodRevoke, params)
	return err
}

// GetSessionAccess gets the access permissions for a session.
func GetSessionAccess(caller types.Transport, sessionID string) (map[string]any, error) {
	params := map[string]any{}
	resp, err := caller.Call(ServiceSession, SessionMethodAccess, params)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := resp.Unmarshal(&result); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal session access response")
	}
	return result, nil
}

// SetSessionData sets session data.
func SetSessionData(caller types.Transport, sessionID string, keys map[string]any) error {
	params := map[string]any{
		SessionParamKeys: keys,
	}
	_, err := caller.Call(ServiceSession, SessionMethodSet, params)
	return err
}

// GetSessionData gets session data.
func GetSessionData(caller types.Transport, sessionID string, keys []string) (map[string]any, error) {
	params := map[string]any{
		SessionParamKeys: keys,
	}
	resp, err := caller.Call(ServiceSession, SessionMethodGet, params)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := resp.Unmarshal(&result); err != nil {
		return nil, errdefs.Wrapf(err, "failed to unmarshal session data response")
	}
	return result, nil
}

// UnsetSessionData unsets session data.
func UnsetSessionData(caller types.Transport, sessionID string, keys []string) error {
	params := map[string]any{
		SessionParamKeys: keys,
	}
	_, err := caller.Call(ServiceSession, SessionMethodUnset, params)
	return err
}

// parseSessionData parses the session data from a response.
func parseSessionData(resp types.Result) (*types.SessionData, error) {
	var sessionData types.SessionData
	if err := resp.Unmarshal(&sessionData); err != nil {
		return nil, err
	}

	// Calculate expire time
	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)

	return &sessionData, nil
}
