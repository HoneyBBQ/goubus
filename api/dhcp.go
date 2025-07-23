package api

import (
	"github.com/honeybbq/goubus/types"
)

// DHCP service and method constants
const (
	ServiceDHCP        = "dhcp"
	DHCPMethodAddLease = "add_lease"
)

// DHCP field constants
const (
	DHCPParamIP        = "ip"
	DHCPParamMAC       = "mac"
	DHCPParamDUID      = "duid"
	DHCPParamHostID    = "hostid"
	DHCPParamLeaseTime = "leasetime"
	DHCPParamName      = "name"
)

// AddDHCPLease adds a static DHCP lease.
func AddDHCPLease(caller types.Transport, req types.AddLeaseRequest) error {
	params := map[string]any{
		DHCPParamIP:        req.IP,
		DHCPParamMAC:       req.Mac,
		DHCPParamDUID:      req.DUID,
		DHCPParamHostID:    req.HostID,
		DHCPParamLeaseTime: req.LeaseTime,
		DHCPParamName:      req.Name,
	}
	_, err := caller.Call(ServiceDHCP, DHCPMethodAddLease, params)
	return err
}
