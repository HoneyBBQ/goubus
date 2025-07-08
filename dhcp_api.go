package goubus

// StaticLeaseConfig represents the configuration parameters for a DHCP static lease
type StaticLeaseConfig struct {
	Name   string `json:"name,omitempty"`   // Hostname
	MAC    string `json:"mac,omitempty"`    // MAC address
	IP     string `json:"ip,omitempty"`     // IPv4 address
	HostID string `json:"hostid,omitempty"` // IPv6 host ID
	DUID   string `json:"duid,omitempty"`   // DHCPv6 Unique Identifier
}

// StaticLeaseCreateRequest represents the parameters for creating a new static lease
type StaticLeaseCreateRequest struct {
	Type   string            `json:"type"`   // Usually "host"
	Config StaticLeaseConfig `json:"config"` // Initial configuration
}

// DHCP returns a manager for the 'dhcp' UCI configuration.
func (c *Client) DHCP() *DHCPManager {
	return &DHCPManager{
		client: c,
	}
}

// DHCPManager provides methods to interact with the dhcp configuration.
type DHCPManager struct {
	client *Client
}

// StaticLease selects a specific host section (static lease) for configuration.
func (dm *DHCPManager) StaticLease(sectionName string) *StaticLeaseManager {
	return &StaticLeaseManager{
		client:  dm.client,
		section: sectionName,
	}
}

// Commit saves all staged changes for the dhcp configuration file.
func (dm *DHCPManager) Commit() error {
	req := UbusUciRequestGeneric{
		Config: "dhcp",
	}
	return dm.client.uciCommit(dm.client.id, req)
}

// Leases retrieves both IPv4 and IPv6 DHCP leases in a single optimized call.
func (dm *DHCPManager) Leases() (UbusDhcpLeases, error) {
	return dm.client.dhcpLeases()
}

// StaticLeaseManager provides methods to configure a specific static lease.
type StaticLeaseManager struct {
	client  *Client
	section string
}

// Get retrieves the static lease configuration.
func (slm *StaticLeaseManager) Get() (*StaticLeaseConfig, error) {
	req := UbusUciGetRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "dhcp",
			Section: slm.section,
		},
	}
	resp, err := slm.client.uciGet(slm.client.id, req)
	if err != nil {
		return nil, err
	}

	config := &StaticLeaseConfig{}
	if val, ok := resp.Values["name"]; ok {
		config.Name = val
	}
	if val, ok := resp.Values["mac"]; ok {
		config.MAC = val
	}
	if val, ok := resp.Values["ip"]; ok {
		config.IP = val
	}
	if val, ok := resp.Values["hostid"]; ok {
		config.HostID = val
	}
	if val, ok := resp.Values["duid"]; ok {
		config.DUID = val
	}

	return config, nil
}

// Set applies configuration parameters to the static lease section.
func (slm *StaticLeaseManager) Set(config StaticLeaseConfig) error {
	values := staticLeaseConfigToMap(config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "dhcp",
			Section: slm.section,
		},
		Values: values,
	}
	return slm.client.uciSet(slm.client.id, req)
}

// Add creates a new static lease section with the specified configuration.
func (slm *StaticLeaseManager) Add(request StaticLeaseCreateRequest) error {
	values := staticLeaseConfigToMap(request.Config)
	req := UbusUciRequest{
		UbusUciRequestGeneric: UbusUciRequestGeneric{
			Config:  "dhcp",
			Section: slm.section,
			Type:    request.Type,
		},
		Values: values,
	}
	return slm.client.uciAdd(slm.client.id, req)
}

// Delete removes the static lease section from the configuration.
func (slm *StaticLeaseManager) Delete() error {
	req := UbusUciRequestGeneric{
		Config:  "dhcp",
		Section: slm.section,
	}
	return slm.client.uciDelete(slm.client.id, req)
}

// staticLeaseConfigToMap converts a StaticLeaseConfig to map[string]string for UCI operations
func staticLeaseConfigToMap(config StaticLeaseConfig) map[string]string {
	result := make(map[string]string)
	if config.Name != "" {
		result["name"] = config.Name
	}
	if config.MAC != "" {
		result["mac"] = config.MAC
	}
	if config.IP != "" {
		result["ip"] = config.IP
	}
	if config.HostID != "" {
		result["hostid"] = config.HostID
	}
	if config.DUID != "" {
		result["duid"] = config.DUID
	}
	return result
}
