// Copyright (c) 2026 honeybbq
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package dhcp

// Leases represents DHCP leases.
type Leases struct {
	IPv4Leases []IPv4Lease `json:"dhcp_leases"`
	IPv6Leases []IPv6Lease `json:"dhcp6_leases"`
}

// IPv4Lease represents an IPv4 lease.
type IPv4Lease struct {
	Hostname string `json:"hostname"`
	IPAddr   string `json:"ipaddr"`
	MACAddr  string `json:"macaddr"`
	Expires  int64  `json:"expires"`
}

// IPv6Lease represents an IPv6 lease.
type IPv6Lease struct {
	Hostname string   `json:"hostname"`
	DUID     string   `json:"duid"`
	IPAddr   []string `json:"ip6addr"`
	Expires  int64    `json:"expires"`
}

// IPv6RA represents an IPv6 Router Advertisement entry.
type IPv6RA struct {
	Hostname string   `json:"hostname"`
	DUID     string   `json:"duid"`
	IPAddr   []string `json:"ip6addr"`
	Expires  int64    `json:"expires"`
}

// AddLeaseRequest represents parameters for adding a lease.
type AddLeaseRequest struct {
	IP        string
	Mac       string
	DUID      string
	HostID    string
	LeaseTime string
	Name      string
}
