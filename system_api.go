package goubus

// System returns a manager for system-related operations.
func (c *Client) System() *SystemManager {
	return &SystemManager{
		client: c,
	}
}

// SystemManager provides methods to interact with system information.
type SystemManager struct {
	client *Client
}

// Info retrieves runtime system information (uptime, memory, load, etc.).
func (sm *SystemManager) Info() (*SystemInfo, error) {
	return sm.client.systemGetInfo()
}

// Board retrieves hardware-specific board information.
func (sm *SystemManager) Board() (*SystemBoardInfo, error) {
	return sm.client.systemGetBoardInfo()
}

// Reboot reboots the system.
func (sm *SystemManager) Reboot() error {
	return sm.client.systemReboot()
}
