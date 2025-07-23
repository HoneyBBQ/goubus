package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// Log returns a manager for log operations.
func (c *Client) Log() *LogManager {
	return &LogManager{
		client: c,
	}
}

// LogManager provides methods to interact with the system log.
type LogManager struct {
	client *Client
}

// Read retrieves log entries from the system.
func (lm *LogManager) Read(lines int, stream bool, oneshot bool) (*types.Log, error) {
	return api.ReadLog(lm.client.caller, lines, stream, oneshot)
}

// Write sends a new entry to the system log.
func (lm *LogManager) Write(event string) error {
	return api.WriteLog(lm.client.caller, event)
}
