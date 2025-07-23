package api

import (
	"github.com/honeybbq/goubus/types"
)

// RC service and method constants
const (
	ServiceRC = "rc"

	RCMethodList = "list"
	RCMethodInit = "init"
)

// GetRcList retrieves a list of available init scripts.
func GetRcList(caller types.Transport) (map[string]types.RcList, error) {
	resp, err := caller.Call(ServiceRC, RCMethodList, nil)
	if err != nil {
		return nil, err
	}
	ubusData := make(map[string]types.RcList)
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}

// ControlRcInit controls an init script (start, stop, restart, etc.).
func ControlRcInit(caller types.Transport, request types.RcInitRequest) error {
	_, err := caller.Call(ServiceRC, RCMethodInit, request)
	return err
}
