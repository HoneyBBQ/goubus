package goubus

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

// Read reads log entries with the specified parameters.
func (lm *LogManager) Read(lines int, stream bool, oneshot bool) (UbusLog, error) {
	return lm.client.logRead(lines, stream, oneshot)
}

// Write writes a log entry.
func (lm *LogManager) Write(event string) error {
	return lm.client.logWrite(event)
}
