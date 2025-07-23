package api

import (
	"github.com/honeybbq/goubus/types"
)

// Service service and method constants
const (
	ServiceService      = "service"
	ServiceMethodList   = "list"
	ServiceMethodSet    = "set"
	ServiceMethodDelete = "delete"
	ServiceMethodSignal = "signal"
	ServiceMethodEvent  = "event"
	ServiceMethodState  = "state"
)

// Service parameter constants
const (
	ServiceParamName      = "name"
	ServiceParamScript    = "script"
	ServiceParamInstances = "instances"
	ServiceParamTriggers  = "triggers"
	ServiceParamAutostart = "autostart"
	ServiceParamInstance  = "instance"
	ServiceParamSignal    = "signal"
	ServiceParamType      = "type"
	ServiceParamData      = "data"
)

// ServiceListResponse is a map where the key is the service name.
type ServiceListResponse map[string]types.ServiceInfo

// GetServiceList retrieves the status of all services or a specific one.
func GetServiceList(caller types.Transport, name string, verbose bool) (map[string]types.ServiceInfo, error) {
	params := make(map[string]any)
	if name != "" {
		params["name"] = name
	}
	if verbose {
		params["verbose"] = true
	}

	resp, err := caller.Call(ServiceService, ServiceMethodList, params)
	if err != nil {
		return nil, err
	}

	ubusData := make(map[string]types.ServiceInfo)
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}

	return ubusData, nil
}

// SetService sets or adds a service.
func SetService(caller types.Transport, name, script string, instances map[string]any, triggers []any, autostart bool, data map[string]any) error {
	params := map[string]any{
		ServiceParamName:      name,
		ServiceParamScript:    script,
		ServiceParamInstances: instances,
		ServiceParamTriggers:  triggers,
		ServiceParamAutostart: autostart,
		ServiceParamData:      data,
	}
	_, err := caller.Call(ServiceService, ServiceMethodSet, params)
	return err
}

// DeleteService deletes a service instance.
func DeleteService(caller types.Transport, name, instance string) error {
	params := map[string]any{
		"name":               name,
		ServiceParamInstance: instance,
	}
	_, err := caller.Call(ServiceService, ServiceMethodDelete, params)
	return err
}

// SignalService sends a signal to a service instance.
func SignalService(caller types.Transport, name, instance string, signal int) error {
	params := map[string]any{
		ServiceParamName:     name,
		ServiceParamInstance: instance,
		ServiceParamSignal:   signal,
	}
	_, err := caller.Call(ServiceService, ServiceMethodSignal, params)
	return err
}

// ServiceEvent sends an event.
func ServiceEvent(caller types.Transport, eventType string, data map[string]any) error {
	params := map[string]any{
		ServiceParamType: eventType,
		ServiceParamData: data,
	}
	_, err := caller.Call(ServiceService, ServiceMethodEvent, params)
	return err
}

// GetServiceState gets the state of a service.
func GetServiceState(caller types.Transport, name string) (map[string]any, error) {
	params := map[string]any{ServiceParamName: name}
	resp, err := caller.Call(ServiceService, ServiceMethodState, params)
	if err != nil {
		return nil, err
	}
	var ubusData map[string]any
	if err := resp.Unmarshal(&ubusData); err != nil {
		return nil, err
	}
	return ubusData, nil
}
