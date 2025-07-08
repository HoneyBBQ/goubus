package goubus

import (
	"fmt"
	"slices"
	"strings"
)

// WifiDeviceConfig represents the configuration parameters for a wifi-device
type WifiDeviceConfig struct {
	Type     string   `json:"type,omitempty"`     // Device type: "mac80211", "broadcom", etc.
	Channel  string   `json:"channel,omitempty"`  // Channel number (1-14 for 2.4GHz, 36+ for 5GHz)
	Country  string   `json:"country,omitempty"`  // Country code (US, DE, etc.)
	HTMode   string   `json:"htmode,omitempty"`   // HT mode: "HT20", "HT40", "VHT80", etc.
	TXPower  string   `json:"txpower,omitempty"`  // TX power in dBm
	Path     string   `json:"path,omitempty"`     // Device path
	Disabled string   `json:"disabled,omitempty"` // "0" or "1"
	Legacy   []string `json:"legacy,omitempty"`   // Legacy rates
	Hwmode   string   `json:"hwmode,omitempty"`   // Hardware mode: "11g", "11a", "11n", etc.
}

// WifiIfaceConfig represents the configuration parameters for a wifi-iface
type WifiIfaceConfig struct {
	Device     string `json:"device,omitempty"`     // Parent wifi-device name
	Network    string `json:"network,omitempty"`    // Associated network interface
	Mode       string `json:"mode,omitempty"`       // Mode: "ap", "sta", "adhoc", "monitor"
	SSID       string `json:"ssid,omitempty"`       // Network SSID
	Encryption string `json:"encryption,omitempty"` // Encryption: "none", "wep", "psk", "psk2", "psk-mixed"
	Key        string `json:"key,omitempty"`        // Encryption key/password
	Hidden     string `json:"hidden,omitempty"`     // "0" or "1" - hide SSID
	Isolate    string `json:"isolate,omitempty"`    // "0" or "1" - client isolation
	Disabled   string `json:"disabled,omitempty"`   // "0" or "1"
	BSSID      string `json:"bssid,omitempty"`      // BSSID for station mode
	WPS        string `json:"wps,omitempty"`        // WPS pushbutton: "0" or "1"
	MaxAssoc   string `json:"maxassoc,omitempty"`   // Maximum associated clients
}

// WifiDeviceCreateRequest represents the parameters for creating a new wifi-device
type WifiDeviceCreateRequest struct {
	Type   string           `json:"type"`   // Usually "wifi-device"
	Config WifiDeviceConfig `json:"config"` // Initial configuration
}

// WifiIfaceCreateRequest represents the parameters for creating a new wifi-iface
type WifiIfaceCreateRequest struct {
	Type   string          `json:"type"`   // Usually "wifi-iface"
	Config WifiIfaceConfig `json:"config"` // Initial configuration
}

// Wireless returns a manager for the 'wireless' UCI configuration.
func (c *Client) Wireless() *WirelessManager {
	return &WirelessManager{
		client: c,
	}
}

// WirelessManager provides methods to interact with the wireless configuration.
type WirelessManager struct {
	client *Client
}

// Device selects a specific wifi-device section (a physical radio, e.g., 'radio0') for configuration.
func (wm *WirelessManager) Device(sectionName string) *WifiDeviceManager {
	return &WifiDeviceManager{
		client:  wm.client,
		section: sectionName,
	}
}

// Interface selects a specific wifi-iface section (a virtual AP, e.g., 'default_radio0') for configuration.
func (wm *WirelessManager) Interface(sectionName string) *WifiIfaceManager {
	return &WifiIfaceManager{
		client:  wm.client,
		section: sectionName,
	}
}

// Commit saves all staged changes for the wireless configuration file.
func (wm *WirelessManager) Commit() error {
	req := UbusUciRequestGeneric{
		Config: "wireless",
	}
	return wm.client.uciCommit(wm.client.id, req)
}

// GetAvailableDevices returns a list of available wireless devices
func (wm *WirelessManager) GetAvailableDevices() ([]string, error) {
	devices, err := wm.client.wirelessDevices()
	if err != nil {
		return nil, err
	}
	return devices.Devices, nil
}

// WifiDeviceManager provides methods to configure a specific wifi-device.
type WifiDeviceManager struct {
	client  *Client
	section string
}

// Get retrieves the static configuration for the wifi-device.
func (wdm *WifiDeviceManager) Get() (*WifiDeviceConfig, error) {
	req := UbusUciGetRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wdm.section,
		},
	}
	resp, err := wdm.client.uciGet(wdm.client.id, req)
	if err != nil {
		return nil, err
	}

	config := &WifiDeviceConfig{}
	if val, ok := resp.Values["type"]; ok {
		config.Type = val
	}
	if val, ok := resp.Values["channel"]; ok {
		config.Channel = val
	}
	if val, ok := resp.Values["country"]; ok {
		config.Country = val
	}
	if val, ok := resp.Values["htmode"]; ok {
		config.HTMode = val
	}
	if val, ok := resp.Values["txpower"]; ok {
		config.TXPower = val
	}
	if val, ok := resp.Values["path"]; ok {
		config.Path = val
	}
	if val, ok := resp.Values["disabled"]; ok {
		config.Disabled = val
	}
	if val, ok := resp.Values["legacy_rates"]; ok {
		config.Legacy = strings.Split(val, " ")
	}
	if val, ok := resp.Values["hwmode"]; ok {
		config.Hwmode = val
	}

	return config, nil
}

// Set applies configuration parameters to the wifi-device section.
func (wdm *WifiDeviceManager) Set(config WifiDeviceConfig) error {
	values := wifiDeviceConfigToMap(config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wdm.section,
			Type:    "wifi-device",
		},
		Values: values,
	}
	return wdm.client.uciSet(wdm.client.id, req)
}

// SetChannel is a helper method to set the channel for the device.
func (wdm *WifiDeviceManager) SetChannel(channel string) error {
	return wdm.Set(WifiDeviceConfig{Channel: channel})
}

// SetCountry is a helper method to set the country code for the device.
func (wdm *WifiDeviceManager) SetCountry(country string) error {
	return wdm.Set(WifiDeviceConfig{Country: country})
}

// SetHTMode is a helper method to set the HT mode for the device.
func (wdm *WifiDeviceManager) SetHTMode(htmode string) error {
	return wdm.Set(WifiDeviceConfig{HTMode: htmode})
}

// AddLegacyRate adds a new legacy rate to the device's legacy rates list.
func (wdm *WifiDeviceManager) AddLegacyRate(rate string) error {
	// Get current configuration
	currentConfig, err := wdm.Get()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Check if legacy rate already exists
	if slices.Contains(currentConfig.Legacy, rate) {
		return nil // Legacy rate already exists, no need to add
	}

	// Add new legacy rate
	currentConfig.Legacy = append(currentConfig.Legacy, rate)

	// Update configuration using Set method
	return wdm.Set(*currentConfig)
}

// DeleteLegacyRate removes a specific legacy rate from the device's legacy rates list.
func (wdm *WifiDeviceManager) DeleteLegacyRate(rate string) error {
	// Get current configuration
	currentConfig, err := wdm.Get()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	// Remove specified rate from legacy rates list
	newLegacyList := make([]string, 0, len(currentConfig.Legacy))
	found := false
	for _, legacyRate := range currentConfig.Legacy {
		if legacyRate != rate {
			newLegacyList = append(newLegacyList, legacyRate)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("legacy rate '%s' not found in the current legacy rates list", rate)
	}

	// Update legacy rates list
	currentConfig.Legacy = newLegacyList

	// Update configuration using Set method
	return wdm.Set(*currentConfig)
}

// Add creates a new wifi-device section with the specified configuration.
func (wdm *WifiDeviceManager) Add(request WifiDeviceCreateRequest) error {
	values := wifiDeviceConfigToMap(request.Config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wdm.section,
			Type:    request.Type,
		},
		Values: values,
	}
	return wdm.client.uciAdd(wdm.client.id, req)
}

// Delete removes the wifi-device section from the configuration.
func (wdm *WifiDeviceManager) Delete() error {
	req := UbusUciRequestGeneric{
		Config:  "wireless",
		Section: wdm.section,
	}
	return wdm.client.uciDelete(wdm.client.id, req)
}

func (wdm *WifiDeviceManager) Info() (UbusWirelessInfoData, error) {
	return wdm.client.wirelessInfo(wdm.section)
}

func (wdm *WifiDeviceManager) Scan() (UbusWirelessScanner, error) {
	return wdm.client.wirelessScanner(wdm.section)
}

func (wdm *WifiDeviceManager) AssocList(mac string) (UbusWirelessAssocList, error) {
	return wdm.client.wirelessAssocList(wdm.section, mac)
}

// WifiIfaceManager provides methods to configure a specific wifi-iface.
type WifiIfaceManager struct {
	client  *Client
	section string
}

// Get retrieves the static configuration for the wifi-iface.
func (wim *WifiIfaceManager) Get() (*WifiIfaceConfig, error) {
	req := UbusUciGetRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wim.section,
		},
	}
	resp, err := wim.client.uciGet(wim.client.id, req)
	if err != nil {
		return nil, err
	}

	config := &WifiIfaceConfig{}
	if val, ok := resp.Values["device"]; ok {
		config.Device = val
	}
	if val, ok := resp.Values["network"]; ok {
		config.Network = val
	}
	if val, ok := resp.Values["mode"]; ok {
		config.Mode = val
	}
	if val, ok := resp.Values["ssid"]; ok {
		config.SSID = val
	}
	if val, ok := resp.Values["encryption"]; ok {
		config.Encryption = val
	}
	if val, ok := resp.Values["key"]; ok {
		config.Key = val
	}
	if val, ok := resp.Values["hidden"]; ok {
		config.Hidden = val
	}
	if val, ok := resp.Values["isolate"]; ok {
		config.Isolate = val
	}
	if val, ok := resp.Values["disabled"]; ok {
		config.Disabled = val
	}
	if val, ok := resp.Values["bssid"]; ok {
		config.BSSID = val
	}
	if val, ok := resp.Values["wps_pushbutton"]; ok {
		config.WPS = val
	}
	if val, ok := resp.Values["maxassoc"]; ok {
		config.MaxAssoc = val
	}

	return config, nil
}

// Set applies configuration parameters to the wifi-iface section.
func (wim *WifiIfaceManager) Set(config WifiIfaceConfig) error {
	values := wifiIfaceConfigToMap(config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wim.section,
			Type:    "wifi-iface",
		},
		Values: values,
	}
	return wim.client.uciSet(wim.client.id, req)
}

// SetSSID is a helper method to set the SSID for the interface.
func (wim *WifiIfaceManager) SetSSID(ssid string) error {
	return wim.Set(WifiIfaceConfig{SSID: ssid})
}

// SetEncryption is a helper method to set the encryption and key for the interface.
func (wim *WifiIfaceManager) SetEncryption(encryption, key string) error {
	return wim.Set(WifiIfaceConfig{Encryption: encryption, Key: key})
}

// SetWPA2 is a helper method to configure WPA2 encryption.
func (wim *WifiIfaceManager) SetWPA2(key string) error {
	return wim.Set(WifiIfaceConfig{Encryption: "psk2", Key: key})
}

// SetOpen is a helper method to configure open (no encryption) access.
func (wim *WifiIfaceManager) SetOpen() error {
	return wim.Set(WifiIfaceConfig{Encryption: "none"})
}

// Add creates a new wifi-iface section with the specified configuration.
func (wim *WifiIfaceManager) Add(request WifiIfaceCreateRequest) error {
	values := wifiIfaceConfigToMap(request.Config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "wireless",
			Section: wim.section,
			Type:    request.Type,
		},
		Values: values,
	}
	return wim.client.uciAdd(wim.client.id, req)
}

// Delete removes the wifi-iface section from the configuration.
func (wim *WifiIfaceManager) Delete() error {
	req := UbusUciRequestGeneric{
		Config:  "wireless",
		Section: wim.section,
	}
	return wim.client.uciDelete(wim.client.id, req)
}

// Helper functions to convert config structs to maps
func wifiDeviceConfigToMap(config WifiDeviceConfig) map[string]string {
	result := make(map[string]string)
	if config.Type != "" {
		result["type"] = config.Type
	}
	if config.Channel != "" {
		result["channel"] = config.Channel
	}
	if config.Country != "" {
		result["country"] = config.Country
	}
	if config.HTMode != "" {
		result["htmode"] = config.HTMode
	}
	if config.TXPower != "" {
		result["txpower"] = config.TXPower
	}
	if config.Path != "" {
		result["path"] = config.Path
	}
	if config.Disabled != "" {
		result["disabled"] = config.Disabled
	}
	if len(config.Legacy) > 0 {
		result["legacy_rates"] = strings.Join(config.Legacy, " ")
	}
	if config.Hwmode != "" {
		result["hwmode"] = config.Hwmode
	}
	return result
}

func wifiIfaceConfigToMap(config WifiIfaceConfig) map[string]string {
	result := make(map[string]string)
	if config.Device != "" {
		result["device"] = config.Device
	}
	if config.Network != "" {
		result["network"] = config.Network
	}
	if config.Mode != "" {
		result["mode"] = config.Mode
	}
	if config.SSID != "" {
		result["ssid"] = config.SSID
	}
	if config.Encryption != "" {
		result["encryption"] = config.Encryption
	}
	if config.Key != "" {
		result["key"] = config.Key
	}
	if config.Hidden != "" {
		result["hidden"] = config.Hidden
	}
	if config.Isolate != "" {
		result["isolate"] = config.Isolate
	}
	if config.Disabled != "" {
		result["disabled"] = config.Disabled
	}
	if config.BSSID != "" {
		result["bssid"] = config.BSSID
	}
	if config.WPS != "" {
		result["wps_pushbutton"] = config.WPS
	}
	if config.MaxAssoc != "" {
		result["maxassoc"] = config.MaxAssoc
	}
	return result
}
