package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// SystemManager provides methods to interact with system information.
type SystemManager struct {
	client *Client
}

// System returns a manager for system-related operations.
// Corresponds to 'ubus call system ...'.
func (c *Client) System() *SystemManager {
	return &SystemManager{
		client: c,
	}
}

// Info retrieves runtime system information (uptime, memory, load, etc.).
// Corresponds to 'ubus call system info'.
func (sm *SystemManager) Info() (*types.SystemInfo, error) {
	return api.GetSystemInfo(sm.client.caller)
}

// Board retrieves hardware-specific board information.
// Corresponds to 'ubus call system board'.
func (sm *SystemManager) Board() (*types.SystemBoardInfo, error) {
	return api.GetSystemBoardInfo(sm.client.caller)
}

// Reboot reboots the system.
// Corresponds to 'ubus call system reboot'.
func (sm *SystemManager) Reboot() error {
	return api.SystemReboot(sm.client.caller)
}
