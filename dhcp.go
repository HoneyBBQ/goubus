package goubus

import (
	"github.com/honeybbq/goubus/api"
	"github.com/honeybbq/goubus/types"
)

// DHCPManager provides an interface for interacting with the DHCP server.
type DHCPManager struct {
	client *Client
}

// DHCP returns a new DHCPManager.
func (c *Client) DHCP() *DHCPManager {
	return &DHCPManager{client: c}
}

// AddLease adds a static DHCP lease.
// Note: This method seems to use a non-UCI ubus call 'dhcp.add_lease'.
// We will keep it here as it's not a direct UCI modification.
func (dm *DHCPManager) AddLease(req types.AddLeaseRequest) error {
	return api.AddDHCPLease(dm.client.caller, req)
}
