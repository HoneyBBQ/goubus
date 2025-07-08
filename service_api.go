package goubus

// Service returns a manager for a specific system service.
func (c *Client) Service(name string) *ServiceManager {
	return &ServiceManager{
		client: c,
		name:   name,
	}
}

// ServiceManager provides methods to interact with a system service (e.g., 'firewall', 'network').
type ServiceManager struct {
	client *Client
	name   string
}

// Start starts the service.
func (sm *ServiceManager) Start() error {
	return sm.client.startService(sm.name)
}

// Stop stops the service.
func (sm *ServiceManager) Stop() error {
	return sm.client.stopService(sm.name)
}

// Restart restarts the service.
func (sm *ServiceManager) Restart() error {
	return sm.client.restartService(sm.name)
}

// Status retrieves the current status of the service.
func (sm *ServiceManager) Status() (ServiceListResponse, error) {
	return sm.client.getServiceList(ServiceListRequest{Name: sm.name})
}
