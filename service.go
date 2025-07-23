package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// ServiceManager provides an interface for managing system services.
type ServiceManager struct {
	client *Client
}

// Service returns a new ServiceManager.
func (c *Client) Service() *ServiceManager {
	return &ServiceManager{client: c}
}

// List retrieves the status of all services or a specific one.
func (sm *ServiceManager) List(name string, verbose bool) (map[string]types.ServiceInfo, error) {
	return api.GetServiceList(sm.client.caller, name, verbose)
}

// Set sets or adds a service configuration.
func (sm *ServiceManager) Set(name, script string, instances map[string]any, triggers []any, autostart bool, data map[string]any) error {
	return api.SetService(sm.client.caller, name, script, instances, triggers, autostart, data)
}

// Delete deletes a service instance.
func (sm *ServiceManager) Delete(name, instance string) error {
	return api.DeleteService(sm.client.caller, name, instance)
}

// Signal sends a signal to a service instance.
func (sm *ServiceManager) Signal(name, instance string, signal int) error {
	return api.SignalService(sm.client.caller, name, instance, signal)
}

// Event sends a custom event.
func (sm *ServiceManager) Event(eventType string, data map[string]any) error {
	return api.ServiceEvent(sm.client.caller, eventType, data)
}

// State retrieves the state of a service.
func (sm *ServiceManager) State(name string) (map[string]any, error) {
	return api.GetServiceState(sm.client.caller, name)
}
