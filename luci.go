package goubus

import (
	"time"

	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// LuciManager provides an interface for interacting with the LuCI RPC interface.
type LuciManager struct {
	client *Client
}

// Luci returns a new LuciManager.
func (c *Client) Luci() *LuciManager {
	return &LuciManager{client: c}
}

// GetVersion retrieves the LuCI version information.
func (lm *LuciManager) GetVersion() (*types.LuciVersion, error) {
	return api.GetLuciVersion(lm.client.caller)
}

// GetLocaltime retrieves the current local time from the device.
func (lm *LuciManager) GetLocaltime() (time.Time, error) {
	return api.GetLuciLocaltime(lm.client.caller)
}

// SetLocaltime sets the local time on the device.
func (lm *LuciManager) SetLocaltime(t time.Time) error {
	return api.SetLuciLocaltime(lm.client.caller, t)
}

// GetInitList retrieves the list of init scripts.
func (lm *LuciManager) GetInitList(name string) (map[string]any, error) {
	return api.GetLuciInitList(lm.client.caller, name)
}

// SetInitAction performs an action on an init script.
func (lm *LuciManager) SetInitAction(name, action string) error {
	return api.SetLuciInitAction(lm.client.caller, name, action)
}

// GetTimezones retrieves the list of available timezones.
func (lm *LuciManager) GetTimezones() (map[string]any, error) {
	return api.GetLuciTimezones(lm.client.caller)
}

// =============================================================================
// LUCI-RPC METHODS
// =============================================================================

// GetNetworkDevices retrieves detailed network device information.
// This provides more comprehensive device information than the standard network API.
func (lm *LuciManager) GetNetworkDevices() (map[string]types.LuciNetworkDevice, error) {
	return api.GetLuciRPCNetworkDevices(lm.client.caller)
}

// GetWirelessDevices retrieves detailed wireless device information.
// This includes iwinfo data alongside wireless configuration.
func (lm *LuciManager) GetWirelessDevices() (map[string]types.LuciWirelessDevice, error) {
	return api.GetLuciRPCWirelessDevices(lm.client.caller)
}

// GetHostHints retrieves host hint information.
// This provides MAC to IP/hostname mappings for known devices.
func (lm *LuciManager) GetHostHints() (map[string]types.LuciHostHint, error) {
	return api.GetLuciRPCHostHints(lm.client.caller)
}

// GetDUIDHints retrieves DUID hint information.
// This provides IPv6 DUID mappings.
func (lm *LuciManager) GetDUIDHints() (map[string]any, error) {
	return api.GetLuciRPCDUIDHints(lm.client.caller)
}

// GetBoardJSON retrieves board hardware information.
// This provides detailed hardware capabilities and default configuration.
func (lm *LuciManager) GetBoardJSON() (*types.LuciBoardJSON, error) {
	return api.GetLuciRPCBoardJSON(lm.client.caller)
}

func (lm *LuciManager) GetDHCPLeases() (*types.DHCPLeases, error) {
	return api.GetLuciRPCDHCPLeases(lm.client.caller)
}
