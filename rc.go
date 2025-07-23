package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// RC returns a manager for init script ('rc') operations.
func (c *Client) RC() *RCManager {
	return &RCManager{
		client: c,
	}
}

// RCManager provides methods to interact with init scripts.
type RCManager struct {
	client *Client
}

// List retrieves a list of available init scripts.
func (rcm *RCManager) List() (map[string]types.RcList, error) {
	return api.GetRcList(rcm.client.caller)
}

// Init performs an init script action (start, stop, restart, etc.) on the specified service.
func (rcm *RCManager) Init(name, action string) error {
	req := types.RcInitRequest{
		Name:   name,
		Action: action,
	}
	return api.ControlRcInit(rcm.client.caller, req)
}

// Start starts the specified service.
func (rcm *RCManager) Start(name string) error {
	return rcm.Init(name, "start")
}

// Stop stops the specified service.
func (rcm *RCManager) Stop(name string) error {
	return rcm.Init(name, "stop")
}

// Restart restarts the specified service.
func (rcm *RCManager) Restart(name string) error {
	return rcm.Init(name, "restart")
}

// Reload reloads the specified service.
func (rcm *RCManager) Reload(name string) error {
	return rcm.Init(name, "reload")
}

// Enable enables the specified service to start at boot.
func (rcm *RCManager) Enable(name string) error {
	return rcm.Init(name, "enable")
}

// Disable disables the specified service from starting at boot.
func (rcm *RCManager) Disable(name string) error {
	return rcm.Init(name, "disable")
}
