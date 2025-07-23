package api

import (
	"github.com/honeybbq/goubus/types"
)

// System service and method constants
const (
	ServiceSystem      = "system"
	SystemMethodInfo   = "info"
	SystemMethodBoard  = "board"
	SystemMethodReboot = "reboot"
)

// GetSystemInfo retrieves runtime system information using the RPC client.
func GetSystemInfo(caller types.Transport) (*types.SystemInfo, error) {
	response, err := caller.Call(ServiceSystem, SystemMethodInfo, nil)
	if err != nil {
		return nil, err
	}
	systemInfo := &types.SystemInfo{}
	err = response.Unmarshal(systemInfo)
	if err != nil {
		return nil, err
	}
	return systemInfo, nil
}

// GetSystemBoardInfo retrieves hardware-specific board information using the RPC client.
func GetSystemBoardInfo(caller types.Transport) (*types.SystemBoardInfo, error) {
	response, err := caller.Call(ServiceSystem, SystemMethodBoard, nil)
	if err != nil {
		return nil, err
	}
	systemBoardInfo := &types.SystemBoardInfo{}
	err = response.Unmarshal(systemBoardInfo)
	if err != nil {
		return nil, err
	}
	return systemBoardInfo, nil
}

// SystemReboot reboots the system using the RPC client.
func SystemReboot(caller types.Transport) error {
	_, err := caller.Call(ServiceSystem, SystemMethodReboot, nil)
	return err
}
