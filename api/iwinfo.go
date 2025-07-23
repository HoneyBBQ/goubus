package api

import (
	"github.com/honeybbq/goubus/types"
)

// IwInfo service and method constants
const (
	ServiceIwinfo           = "iwinfo"
	IwInfoMethodInfo        = "info"
	IwInfoMethodScan        = "scan"
	IwInfoMethodDevices     = "devices"
	IwInfoMethodCountryList = "countrylist"
	IwInfoMethodTxPowerList = "txpowerlist"
	IwInfoMethodFreqList    = "freqlist"
	IwInfoMethodAssocList   = "assoclist"
	IwInfoMethodPhyName     = "phyname"
	IwInfoMethodSurvey      = "survey"
)

// IwInfo parameter constants
const (
	IwInfoParamDevice  = "device"
	IwInfoParamMAC     = "mac"
	IwInfoParamSection = "section"
)

// IwinfoDevicesResponse represents the response from iwinfo devices call.
type IwinfoDevicesResponse struct {
	Devices []string `json:"devices"`
}

// GetIwInfoDevices returns a list of available wireless runtime interface names.
func GetIwInfoDevices(caller types.Transport) ([]string, error) {
	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodDevices, nil)
	if err != nil {
		return nil, err
	}
	ubusData := &IwinfoDevicesResponse{}
	if err := resp.Unmarshal(ubusData); err != nil {
		return nil, err
	}
	return ubusData.Devices, nil
}

// GetIwInfoInfo retrieves detailed runtime information for a specific wireless interface.
func GetIwInfoInfo(caller types.Transport, device string) (*types.WirelessInfo, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodInfo, params)
	if err != nil {
		return nil, err
	}
	ubusData := &types.WirelessInfo{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData, nil
}

// WirelessScanResponse represents the response from a wireless scan.
type WirelessScanResponse struct {
	Results []types.WirelessScanResult `json:"results"`
}

// ScanIwInfo performs a scan for nearby wireless networks on a given interface.
func ScanIwInfo(caller types.Transport, device string) ([]types.WirelessScanResult, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodScan, params)
	if err != nil {
		return nil, err
	}
	ubusData := &WirelessScanResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData.Results, nil
}

// WirelessAssocListResponse represents the response from an association list.
type WirelessAssocListResponse struct {
	Results []types.WirelessAssoc `json:"results"`
}

// GetIwInfoAssocList retrieves the list of associated stations for a given interface.
func GetIwInfoAssocList(caller types.Transport, device string, mac string) ([]types.WirelessAssoc, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}
	if mac != "" {
		params[IwInfoParamMAC] = mac
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodAssocList, params)
	if err != nil {
		return nil, err
	}
	ubusData := &WirelessAssocListResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData.Results, nil
}

// WirelessFreqListResponse represents the response from a frequency list.
type WirelessFreqListResponse struct {
	Results []types.WirelessFreq `json:"results"`
}

// GetIwInfoFreqList retrieves the list of available frequencies/channels for a given interface.
func GetIwInfoFreqList(caller types.Transport, device string) ([]types.WirelessFreq, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodFreqList, params)
	if err != nil {
		return nil, err
	}
	ubusData := &WirelessFreqListResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData.Results, nil
}

// WirelessTxPowerListResponse represents the response from a TX power list.
type WirelessTxPowerListResponse struct {
	Results []types.WirelessTxPower `json:"results"`
}

// GetIwInfoTxPowerList retrieves the list of available TX power levels for a given interface.
func GetIwInfoTxPowerList(caller types.Transport, device string) ([]types.WirelessTxPower, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodTxPowerList, params)
	if err != nil {
		return nil, err
	}
	ubusData := &WirelessTxPowerListResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData.Results, nil
}

// GetIwInfoSurvey retrieves the survey list for a given interface.
func GetIwInfoSurvey(caller types.Transport, device string) ([]types.WirelessSurvey, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodSurvey, params)
	if err != nil {
		return nil, err
	}
	ubusData := []types.WirelessSurvey{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData, nil
}

// WirelessPhyNameResponse represents the response from a phyname call.
type WirelessPhyNameResponse struct {
	PhyName string `json:"phyname"`
}

// GetIwInfoPhyName retrieves the phyname for a given section.
func GetIwInfoPhyName(caller types.Transport, section string) (*string, error) {
	params := map[string]any{
		IwInfoParamSection: section,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodPhyName, params)
	if err != nil {
		return nil, err
	}

	ubusData := &WirelessPhyNameResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return &ubusData.PhyName, nil
}

// WirelessCountryListResponse represents the response from a country list.
type WirelessCountryListResponse struct {
	Results []types.WirelessCountry `json:"results"`
}

// GetIwInfoCountryList retrieves the list of available country codes for a given interface.
func GetIwInfoCountryList(caller types.Transport, device string) ([]types.WirelessCountry, error) {
	params := map[string]any{
		IwInfoParamDevice: device,
	}

	resp, err := caller.Call(ServiceIwinfo, IwInfoMethodCountryList, params)
	if err != nil {
		return nil, err
	}
	ubusData := &WirelessCountryListResponse{}
	err = resp.Unmarshal(ubusData)
	if err != nil {
		return nil, err
	}
	return ubusData.Results, nil
}
