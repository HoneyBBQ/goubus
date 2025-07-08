package goubus

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
func (rcm *RCManager) List() (map[string]UbusRcListResponse, error) {
	return rcm.client.rcList()
}

// Init performs an init script action (start, stop, restart, etc.) on the specified service.
func (rcm *RCManager) Init(name, action string) error {
	req := UbusRcInitRequest{
		Name:   name,
		Action: action,
	}
	return rcm.client.rcInit(req)
}
