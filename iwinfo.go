package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// IwInfoManager provides an interface for interacting with the 'iwinfo' ubus service,
// which is used to query wireless device runtime information.
type IwInfoManager struct {
	client *Client
}

// IwInfo returns a new IwInfoManager.
func (c *Client) IwInfo() *IwInfoManager {
	return &IwInfoManager{client: c}
}

// Devices returns a list of available wireless runtime interface names (e.g., "wlan0", "phy0-ap0").
// These names are used as input for other iwinfo calls.
// Corresponds to `ubus call iwinfo devices`.
func (im *IwInfoManager) Devices() ([]string, error) {
	return api.GetIwInfoDevices(im.client.caller)
}

// Info retrieves detailed runtime information for a specific wireless interface.
// Corresponds to `ubus call iwinfo info '{"device":"<ifname>"}'`.
func (im *IwInfoManager) Info(device string) (*types.WirelessInfo, error) {
	return api.GetIwInfoInfo(im.client.caller, device)
}

// Scan performs a scan for nearby wireless networks on a given interface.
// Corresponds to `ubus call iwinfo scan '{"device":"<ifname>"}'`.
func (im *IwInfoManager) Scan(device string) ([]types.WirelessScanResult, error) {
	return api.ScanIwInfo(im.client.caller, device)
}

// AssocList retrieves the list of associated stations (clients) for a given interface.
// An optional mac parameter can be provided to filter the results.
// Corresponds to `ubus call iwinfo assoclist '{"device":"<ifname>"}'`.
func (im *IwInfoManager) AssocList(device string) ([]types.WirelessAssoc, error) {
	return api.GetIwInfoAssocList(im.client.caller, device, "") // MAC filter not implemented at this level for simplicity
}

// FreqList retrieves the list of available frequencies/channels for a given interface.
// Corresponds to `ubus call iwinfo freqlist '{"device":"<ifname>"}'`.
func (im *IwInfoManager) FreqList(device string) ([]types.WirelessFreq, error) {
	return api.GetIwInfoFreqList(im.client.caller, device)
}

// TxPowerList retrieves the list of available TX power levels for a given interface.
// Corresponds to `ubus call iwinfo txpowerlist '{"device":"<ifname>"}'`.
func (im *IwInfoManager) TxPowerList(device string) ([]types.WirelessTxPower, error) {
	return api.GetIwInfoTxPowerList(im.client.caller, device)
}

// CountryList retrieves the list of available country codes for a given interface.
// Corresponds to `ubus call iwinfo countrylist '{"device":"<ifname>"}'`.
func (im *IwInfoManager) CountryList(device string) ([]types.WirelessCountry, error) {
	return api.GetIwInfoCountryList(im.client.caller, device)
}

// Survey retrieves the survey results for a given interface.
// Corresponds to `ubus call iwinfo survey '{"device":"<ifname>"}'`.
func (im *IwInfoManager) Survey(device string) ([]types.WirelessSurvey, error) {
	return api.GetIwInfoSurvey(im.client.caller, device)
}

// PhyName retrieves the phy name for a given interface.
// Corresponds to `ubus call iwinfo phyname '{"section":"<section>"}'`.
func (im *IwInfoManager) PhyName(section string) (*string, error) {
	return api.GetIwInfoPhyName(im.client.caller, section)
}
