package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// SessionManager provides an interface for managing ubus sessions.
type SessionManager struct {
	client *Client
}

// Session returns a new SessionManager.
func (c *Client) Session() *SessionManager {
	return &SessionManager{client: c}
}

// Create creates a new anonymous session with a specified timeout.
func (sm *SessionManager) Create(timeout int) (*types.SessionData, error) {
	return api.CreateSession(sm.client.caller, timeout)
}

// Destroy terminates a given session.
func (sm *SessionManager) Destroy(sessionID string) error {
	return api.DestroySession(sm.client.caller, sessionID)
}

// List lists all active sessions.
func (sm *SessionManager) List(sessionID string) (map[string]any, error) {
	return api.ListSessions(sm.client.caller, sessionID)
}

// Grant grants ACLs to a session.
func (sm *SessionManager) Grant(sessionID string, scope string, objects []string) error {
	return api.GrantSessionAccess(sm.client.caller, sessionID, scope, objects)
}

// Revoke revokes ACLs from a session.
func (sm *SessionManager) Revoke(sessionID string, scope string, objects []string) error {
	return api.RevokeSessionAccess(sm.client.caller, sessionID, scope, objects)
}

// Access checks access for a session.
func (sm *SessionManager) Access(sessionID string) (map[string]any, error) {
	return api.GetSessionAccess(sm.client.caller, sessionID)
}

// Set sets session data.
func (sm *SessionManager) Set(sessionID string, values map[string]any) error {
	return api.SetSessionData(sm.client.caller, sessionID, values)
}

// Get gets session data.
func (sm *SessionManager) Get(sessionID string, keys []string) (map[string]any, error) {
	return api.GetSessionData(sm.client.caller, sessionID, keys)
}

// Unset unsets session data.
func (sm *SessionManager) Unset(sessionID string, keys []string) error {
	return api.UnsetSessionData(sm.client.caller, sessionID, keys)
}
