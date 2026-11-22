package dhcp

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/honeybbq/goubus/v2"
)

// IPv4Lease represents a DHCPv4 lease.
type IPv4Lease struct {
	// Hostname is the client host name.
	Hostname string `json:"hostname"`
	// Interface is the network interface owning the lease.
	Interface string `json:"-"`
	// IPAddr is the leased IPv4 address.
	IPAddr string `json:"ipaddr"`
	// MACAddr is the client's MAC address.
	MACAddr string `json:"macaddr"`
	// Expires is the remaining lease time in seconds.
	Expires int64 `json:"expires"`
}

// IPv6Lease represents a DHCPv6 lease.
type IPv6Lease struct {
	// Hostname is the client host name.
	Hostname string `json:"hostname"`
	// Interface is the network interface owning the lease.
	Interface string `json:"-"`
	// DUID is the client's DHCP unique identifier.
	DUID string `json:"duid"`
	// IAID is the client's identity association identifier.
	IAID string `json:"iaid"`
	// Address is the primary leased IPv6 address.
	Address string `json:"address,omitempty"`
	// IPAddr contains leased IPv6 addresses.
	IPAddr []string `json:"ip6addr,omitempty"`
	// Assigned is the lease assignment timestamp or duration.
	Assigned uint64 `json:"assigned,omitempty"`
	// Valid is the lease validity duration in seconds.
	Valid uint64 `json:"valid,omitempty"`
	// Length is the lease length or duration reported by odhcpd.
	Length uint64 `json:"length,omitempty"`
}

// UnmarshalJSON accepts the IPv6 address keys used by odhcpd versions.
func (lease *IPv6Lease) UnmarshalJSON(data []byte) error {
	type plain IPv6Lease
	var decoded plain
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	err = json.Unmarshal(data, &fields)
	if err != nil {
		return err
	}
	for _, key := range []string{"ipv6", "ipv6-addr"} {
		if raw, ok := fields[key]; ok {
			var addresses []string
			err := json.Unmarshal(raw, &addresses)
			if err == nil {
				decoded.IPAddr = addresses

				break
			}
			var address string
			err = json.Unmarshal(raw, &address)
			if err != nil {
				return err
			}
			decoded.IPAddr = []string{address}

			break
		}
	}
	if decoded.Address == "" && len(decoded.IPAddr) > 0 {
		decoded.Address = decoded.IPAddr[0]
	}
	*lease = IPv6Lease(decoded)

	return nil
}

// IPv6RA represents an IPv6 router advertisement prefix.
type IPv6RA struct {
	// Interface is the network interface advertising the prefix.
	Interface string `json:"-"`
	// Address is the advertised router address.
	Address string `json:"address,omitempty"`
	// Prefix is the advertised IPv6 prefix.
	Prefix string `json:"prefix,omitempty"`
	// Valid is the prefix validity duration in seconds.
	Valid uint64 `json:"valid,omitempty"`
	// Preferred is the preferred prefix duration in seconds.
	Preferred uint64 `json:"preferred,omitempty"`
	// Length is the prefix length in bits.
	Length uint64 `json:"length,omitempty"`
	// Stale reports whether the advertised prefix is stale.
	Stale goubus.Bool `json:"stale,omitempty"`
}

// UnmarshalJSON interprets a string prefix as an address and a number as a length.
func (ra *IPv6RA) UnmarshalJSON(data []byte) error {
	var decoded struct {
		Interface string          `json:"interface"`
		Address   string          `json:"address"`
		Prefix    json.RawMessage `json:"prefix"`
		Length    uint64          `json:"length"`
		Valid     uint64          `json:"valid"`
		Preferred uint64          `json:"preferred"`
		Stale     goubus.Bool     `json:"stale"`
	}
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}
	prefix, length, err := parsePrefix(decoded.Prefix)
	if err != nil {
		return err
	}
	ra.Interface = decoded.Interface
	ra.Address = decoded.Address
	ra.Prefix = prefix
	ra.Length = decoded.Length
	if length != 0 {
		ra.Length = length
	}
	ra.Valid = decoded.Valid
	ra.Preferred = decoded.Preferred
	ra.Stale = decoded.Stale

	return nil
}

func parsePrefix(raw json.RawMessage) (string, uint64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", 0, nil
	}
	if trimmed[0] == '"' {
		var prefix string
		err := json.Unmarshal(trimmed, &prefix)
		if err != nil {
			return "", 0, err
		}
		length, err := strconv.ParseUint(prefix, 10, 64)
		if err == nil {
			return "", length, nil
		}

		return prefix, 0, nil
	}
	var length uint64
	err := json.Unmarshal(trimmed, &length)

	return "", length, err
}

// AddLeaseRequest contains fields for creating a static DHCP lease.
type AddLeaseRequest struct {
	// IP is the leased IP address.
	IP string
	// MAC is the client's MAC address.
	MAC string
	// DUID is the client's DHCP unique identifier.
	DUID string
	// HostID is the client's host identifier.
	HostID string
	// LeaseTime is the requested lease duration.
	LeaseTime string
	// Name is the lease name.
	Name string
}
