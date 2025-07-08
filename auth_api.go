package goubus

import "time"

// Auth returns a manager for authentication operations.
func (c *Client) Auth() *AuthManager {
	return &AuthManager{
		client: c,
	}
}

// AuthManager provides methods to manage authentication sessions.
type AuthManager struct {
	client *Client
}

// SessionInfo represents session information exposed to users.
type SessionInfo struct {
	SessionID     string        `json:"session_id"`
	Username      string        `json:"username"`
	Timeout       int           `json:"timeout"`
	Expires       int           `json:"expires"`
	TimeRemaining time.Duration `json:"time_remaining"`
	IsValid       bool          `json:"is_valid"`
}

// GetSessionInfo retrieves information about the current session.
func (am *AuthManager) GetSessionInfo() (*SessionInfo, error) {
	// For now, we get basic session info from AuthData
	return &SessionInfo{
		SessionID:     am.client.AuthData.UbusRPCSession,
		Username:      am.client.Username,
		TimeRemaining: am.client.AuthGetTimeUntilExpiry(),
	}, nil
}

// IsSessionValid checks if the current session is still valid.
func (am *AuthManager) IsSessionValid() bool {
	return am.client.AuthIsSessionValid()
}

// GetTimeUntilExpiry returns the remaining time until the session expires.
func (am *AuthManager) GetTimeUntilExpiry() time.Duration {
	return am.client.AuthGetTimeUntilExpiry()
}

// Refresh refreshes the current session to extend its lifetime.
func (am *AuthManager) Refresh() error {
	return am.client.AuthRefresh()
}

// Logout logs out the current session.
func (am *AuthManager) Logout() error {
	return am.client.AuthLogout()
}

// Login performs a manual login (usually not needed as it's done automatically).
func (am *AuthManager) Login() error {
	_, err := am.client.AuthLogin()
	return err
}
