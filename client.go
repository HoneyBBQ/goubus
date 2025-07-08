package goubus

import "fmt"

// Client is the main struct for ubus connection and API access
type Client struct {
	Host     string
	Username string
	Password string
	AuthData ubusAuth
	id       int
}

// NewClient creates a new ubus connection
func NewClient(host string, username string, password string) (*Client, error) {
	u := &Client{
		Host:     host,
		Username: username,
		Password: password,
		id:       1, // Initialize id
	}
	_, err := u.AuthLogin()
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	return u, nil
}
