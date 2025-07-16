package goubus

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ubusAuthACLs represents the ACL from user on Authentication
type ubusAuthACLs struct {
	AccessGroup map[string][]string `json:"access-group"`
	Ubus        map[string][]string
	Uci         map[string][]string
}

// ubusAuth is the single internal struct for ubus authentication data.
type ubusAuth struct {
	UbusRPCSession string       `json:"ubus_rpc_session"`
	Timeout        int          `json:"timeout"`
	Expires        int          `json:"expires"`
	ACLs           ubusAuthACLs `json:"acls"`
	ExpireTime     time.Time
}

// AuthLogin Call JSON-RPC method to Router Authentication
func (u *Client) AuthLogin() (UbusResponse, error) {
	loginData := map[string]string{
		JSONFieldUsername: u.Username,
		JSONFieldPassword: u.Password,
	}
	jsonStr := u.buildUbusCallWithSession(NullSessionID, ServiceSession, MethodLogin, loginData)
	call, err := u.Call(jsonStr)
	if err != nil {
		if strings.Contains(err.Error(), "404") { // More robust check
			return UbusResponse{}, ErrUbusModuleNotInstalled
		}
		return UbusResponse{}, WrapError(err, ErrorCodeAuthenticationFailed, "error calling auth login")
	}

	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusResponse{}, ErrDataParsingError
	}

	var authData ubusAuth
	json.Unmarshal(ubusDataByte, &authData)

	// Set the absolute expiration time
	authData.ExpireTime = time.Now().Add(time.Second * time.Duration(authData.Expires))

	u.AuthData = authData
	return call, nil
}

// LoginCheck checks if the ubus session is still valid
func (u *Client) LoginCheck() error {
	// check if ubus session has expired
	if time.Now().After(u.AuthData.ExpireTime) {
		_, err := u.AuthLogin()
		if err != nil {
			return ErrSessionExpired
		}
	}
	return nil
}

// AuthLogout logs out the current session
func (u *Client) AuthLogout() error {
	if u.AuthData.UbusRPCSession == "" {
		return ErrNoActiveSession
	}

	jsonStr := u.buildUbusCall(ServiceSession, MethodDestroy, nil)
	_, err := u.Call(jsonStr)
	if err != nil {
		return fmt.Errorf("error calling auth logout: %w", err)
	}

	// Clear the session data
	u.AuthData = ubusAuth{}
	return nil
}

// AuthRefresh refreshes the current session to extend its lifetime
func (u *Client) AuthRefresh() error {
	if u.AuthData.UbusRPCSession == "" {
		return ErrNoActiveSession
	}

	jsonStr := u.buildUbusCall(ServiceSession, MethodAccess, nil)
	call, err := u.Call(jsonStr)
	if err != nil {
		return WrapError(err, ErrorCodeAuthenticationFailed, "error calling auth refresh")
	}

	// Update session data with new expiration time
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return ErrDataParsingError
	}

	var authData ubusAuth
	json.Unmarshal(ubusDataByte, &authData)

	// Keep the existing session ID but update other fields
	authData.UbusRPCSession = u.AuthData.UbusRPCSession
	authData.ExpireTime = time.Now().Add(time.Second * time.Duration(authData.Expires))

	u.AuthData = authData
	return nil
}

// AuthGetSessionInfo retrieves information about the current session
func (u *Client) AuthGetSessionInfo() (*ubusAuth, error) {
	if u.AuthData.UbusRPCSession == "" {
		return nil, ErrNoActiveSession
	}

	err := u.LoginCheck() // Ensure session is still valid
	if err != nil {
		return nil, err
	}

	// Return a copy of the current auth data
	sessionInfo := u.AuthData
	return &sessionInfo, nil
}

// AuthIsSessionValid checks if the current session is still valid without refreshing
func (u *Client) AuthIsSessionValid() bool {
	if u.AuthData.UbusRPCSession == "" {
		return false
	}
	return time.Now().Before(u.AuthData.ExpireTime)
}

// AuthGetTimeUntilExpiry returns the time remaining until session expiry
func (u *Client) AuthGetTimeUntilExpiry() time.Duration {
	if u.AuthData.UbusRPCSession == "" {
		return 0
	}
	remaining := time.Until(u.AuthData.ExpireTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}
