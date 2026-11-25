package rpc

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/internal/errs"
)

const ubusAuthSessionID = "00000000000000000000000000000000"

type sessionMetadata struct {
	ExpireTime     time.Time `json:"-"`
	UbusRPCSession string    `json:"ubus_rpc_session"`
	Timeout        int       `json:"timeout"`
}

func (c *Client) slideSessionExpiry(sessionID string) {
	c.rwMutex.Lock()
	if !c.closed && c.sessionData.UbusRPCSession == sessionID && c.sessionData.Timeout > 0 {
		c.sessionData.ExpireTime = time.Now().Add(time.Duration(c.sessionData.Timeout) * time.Second)
	}
	c.rwMutex.Unlock()
}

func (c *Client) reauthenticateStaleSession(ctx context.Context, staleSessionID string) (string, error) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	if c.closed {
		return "", goubus.ErrClosed
	}
	if c.sessionData.UbusRPCSession != "" && c.sessionData.UbusRPCSession != staleSessionID {
		return c.sessionData.UbusRPCSession, nil
	}

	err := c.authenticateLocked(ctx)
	if err != nil {
		return "", err
	}

	return c.sessionData.UbusRPCSession, nil
}

func (c *Client) getValidSessionID(ctx context.Context) (string, error) {
	c.rwMutex.RLock()
	if c.closed {
		c.rwMutex.RUnlock()

		return "", goubus.ErrClosed
	}
	if c.sessionData.UbusRPCSession != "" && time.Now().Before(c.sessionData.ExpireTime) {
		sessionID := c.sessionData.UbusRPCSession
		c.rwMutex.RUnlock()

		return sessionID, nil
	}
	c.rwMutex.RUnlock()

	err := c.ensureAuthenticated(ctx)
	if err != nil {
		return "", err
	}

	c.rwMutex.RLock()
	if c.closed {
		c.rwMutex.RUnlock()

		return "", goubus.ErrClosed
	}
	sessionID := c.sessionData.UbusRPCSession
	c.rwMutex.RUnlock()

	return sessionID, nil
}

func (c *Client) ensureAuthenticated(ctx context.Context) error {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()
	if c.closed {
		return goubus.ErrClosed
	}
	if c.sessionData.UbusRPCSession != "" && time.Now().Before(c.sessionData.ExpireTime) {
		return nil
	}

	return c.authenticateLocked(ctx)
}

func (c *Client) authenticateLocked(ctx context.Context) error {
	loginData := map[string]string{"username": c.username, "password": c.password}
	result, _, err := c.rawCallWithMode(ctx, ubusAuthSessionID, "session", "login", loginData)
	if err != nil {
		statusErr, ok := errors.AsType[*httpStatusError](err)
		if ok && statusErr.statusCode == http.StatusNotFound {
			return errs.Wrapf(err, "ubus JSON-RPC endpoint not found")
		}

		return errs.Wrapf(err, "error calling auth login")
	}

	var sessionData sessionMetadata
	err = result.Unmarshal(&sessionData)
	if err != nil {
		return errs.Wrapf(err, "failed to parse session data")
	}
	sessionData.ExpireTime = time.Now().Add(time.Duration(sessionData.Timeout) * time.Second)
	c.sessionData = sessionData

	return nil
}
