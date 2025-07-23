package config

import (
	"strings"

	"github.com/honeybbq/goubus"
	"github.com/honeybbq/goubus/uci"
)

// NetworkInterfaceConfig represents the configuration parameters for a network interface ('interface' section).
// It implements the ConfigModel interface using UCI tags.
type NetworkInterfaceConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Common network interface fields
	Proto     string   `uci:"proto,omitempty,enum=static,dhcp,dhcpv6,pppoe,ppp,6in4,6rd,6to4,dslite,l2tp,wireguard,3g,qmi,ncm,wwan,gre,vti,vxlan,none"`
	Device    string   `uci:"device,omitempty,case=lower"`
	Type      string   `uci:"type,omitempty,enum=bridge,macvlan,8021q,8021ad,ipip,sit,gre,gretap,ip6gre,ip6gretap,vti,vti6,xfrm,veth,vxlan,bond,team"`
	IfName    []string `uci:"ifname,omitempty,join= ,case=lower"`
	Disabled  *bool    `uci:"disabled,omitempty,bool=0/1" default:"false"`
	Auto      *bool    `uci:"auto,omitempty,bool=0/1" default:"true"`
	ForceLink *bool    `uci:"force_link,omitempty,bool=0/1" default:"false"`
	Metric    *int     `uci:"metric,omitempty,range=0-255" default:"0"`
	MTU       *int     `uci:"mtu,omitempty,range=68-9000" default:"1500"`
	IP4Table  *string  `uci:"ip4table,omitempty,case=lower"`
	IP6Table  *string  `uci:"ip6table,omitempty,case=lower"`
	Delegate  *bool    `uci:"delegate,omitempty,bool=0/1" default:"true"`

	// IPv6 Configuration
	IPv6          *bool    `uci:"ipv6,omitempty,bool=0/1" default:"true"`
	IP6Assign     *int     `uci:"ip6assign,omitempty,range=0-64" default:"64"`
	IP6Addr       []string `uci:"ip6addr,omitempty,join= "`
	IP6GW         *string  `uci:"ip6gw,omitempty"`
	IP6Prefix     []string `uci:"ip6prefix,omitempty,join= "`
	IP6Class      []string `uci:"ip6class,omitempty,join= ,case=lower"`
	IP6Hint       *string  `uci:"ip6hint,omitempty"`
	IP6IfaceID    *string  `uci:"ip6ifaceid,omitempty"`
	IP6Deprecated *bool    `uci:"ip6deprecated,omitempty,bool=0/1" default:"false"`
	SourceFilter  *bool    `uci:"sourcefilter,omitempty,bool=0/1" default:"true"`

	// DNS Configuration
	DNS       []string `uci:"dns,omitempty,join= "`
	DNSMetric *int     `uci:"dns_metric,omitempty,range=0-65535" default:"0"`
	DNSSearch []string `uci:"dns_search,omitempty,join= ,case=lower"`
	PeerDNS   *bool    `uci:"peerdns,omitempty,bool=0/1" default:"true"`

	// DHCP Client Configuration
	Hostname     *string  `uci:"hostname,omitempty,case=lower"`
	ClientID     *string  `uci:"clientid,omitempty"`
	VendorID     *string  `uci:"vendorid,omitempty"`
	ReqOpts      []string `uci:"reqopts,omitempty,join= "`
	SendOpts     []string `uci:"sendopts,omitempty,join= "`
	DefaultRoute *bool    `uci:"defaultroute,omitempty,bool=0/1" default:"true"`
	CustomRoutes []string `uci:"customroutes,omitempty,join= "`
	Broadcast    *bool    `uci:"broadcast,omitempty,bool=0/1" default:"false"`

	// Firewall and Routing
	Zone *string `uci:"zone,omitempty,case=lower"`

	// Protocol-specific configurations (flattened into main config)
	StaticConfig    *StaticConfig    `uci:",flatten,omitempty"`
	DHCPConfig      *DHCPConfig      `uci:",flatten,omitempty"`
	PPPConfig       *PPPConfig       `uci:",flatten,omitempty"`
	BridgeConfig    *BridgeConfig    `uci:",flatten,omitempty"`
	TunnelConfig    *TunnelConfig    `uci:",flatten,omitempty"`
	WireGuardConfig *WireGuardConfig `uci:",flatten,omitempty"`
	MobileConfig    *MobileConfig    `uci:",flatten,omitempty"`
	VirtualConfig   *VirtualConfig   `uci:",flatten,omitempty"`

	// Extra fields for unknown options
	Extra map[string]string `uci:",flatten,omitempty"`
}

// NetworkDeviceConfig represents the configuration for a network 'device' section.
type NetworkDeviceConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Name              string            `uci:"name,omitempty,case=lower"`
	Type              string            `uci:"type,omitempty,enum=bridge,macvlan,8021q,8021ad,ipip,sit,gre,gretap,ip6gre,ip6gretap,vti,vti6,xfrm,veth,vxlan,bond,team"`
	MACAddr           *string           `uci:"macaddr,omitempty,case=lower"`
	Ports             []string          `uci:"ports,omitempty,join= ,case=lower"`
	TXQueueLen        *int              `uci:"txqueuelen,omitempty,range=1-10000" default:"1000"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// --- Sub-structs for NetworkInterfaceConfig ---

type StaticConfig struct {
	IPAddr    []string `uci:"ipaddr,omitempty,join= "`
	Netmask   *string  `uci:"netmask,omitempty"`
	Gateway   *string  `uci:"gateway,omitempty"`
	Broadcast *string  `uci:"broadcast,omitempty"`
	DNS       []string `uci:"dns,omitempty,join= "`
}

type DHCPConfig struct {
	Hostname     *string `uci:"hostname,omitempty,case=lower"`
	ClientID     *string `uci:"clientid,omitempty"`
	VendorClass  *string `uci:"vendorclass,omitempty"`
	DefaultRoute *bool   `uci:"defaultroute,omitempty,bool=0/1" default:"true"`
}

type PPPConfig struct {
	Username  *string `uci:"username,omitempty"`
	Password  *string `uci:"password,omitempty"`
	Service   *string `uci:"service,omitempty"`
	Server    *string `uci:"server,omitempty"`
	Keepalive *int    `uci:"keepalive,omitempty,unit=seconds,range=0-3600" default:"60"`
	Demand    *bool   `uci:"demand,omitempty,bool=0/1" default:"false"`
	IdleTime  *int    `uci:"idletime,omitempty,unit=seconds,range=0-7200" default:"300"`
}

type BridgeConfig struct {
	STP              *bool `uci:"stp,omitempty,bool=0/1" default:"false"`
	ForwardDelay     *int  `uci:"forward_delay,omitempty,unit=seconds,range=2-30" default:"15"`
	HelloTime        *int  `uci:"hello_time,omitempty,unit=seconds,range=1-10" default:"2"`
	Priority         *int  `uci:"priority,omitempty,range=0-65535" default:"32768"`
	AgeingTime       *int  `uci:"ageing_time,omitempty,unit=seconds,range=10-1000000" default:"300"`
	IGMPSnooping     *bool `uci:"igmp_snooping,omitempty,bool=0/1" default:"true"`
	MulticastQuerier *bool `uci:"multicast_querier,omitempty,bool=0/1" default:"false"`
}

// --- Tunneling Protocol Configurations ---

// TunnelConfig handles various tunneling protocols (6in4, 6rd, 6to4, dslite, l2tp)
type TunnelConfig struct {
	// 6in4 and tunneling common options
	PeerAddr     *string `uci:"peeraddr,omitempty"`                             // Remote tunnel endpoint
	IPAddr       *string `uci:"ipaddr,omitempty"`                               // Local IPv4 endpoint
	IP6Addr      *string `uci:"ip6addr,omitempty"`                              // IPv6 tunnel address
	IP6Prefix    *string `uci:"ip6prefix,omitempty"`                            // IPv6 prefix for routing
	TunLink      *string `uci:"tunlink,omitempty,case=lower"`                   // Base interface
	DefaultRoute *bool   `uci:"defaultroute,omitempty,bool=0/1" default:"true"` // Create default route
	TTL          *int    `uci:"ttl,omitempty,range=1-255" default:"64"`         // TTL for tunnel
	TOS          *int    `uci:"tos,omitempty,range=0-255" default:"0"`          // Type of service

	// 6rd specific
	IP6PrefixLen *int `uci:"ip6prefixlen,omitempty,range=0-128" default:"32"` // 6rd IPv6 prefix length
	IP4PrefixLen *int `uci:"ip4prefixlen,omitempty,range=0-32" default:"0"`   // 6rd IPv4 prefix length

	// HE.net specific for 6in4
	TunnelID  *string `uci:"tunnelid,omitempty"`  // HE.net tunnel ID
	Username  *string `uci:"username,omitempty"`  // HE.net username
	Password  *string `uci:"password,omitempty"`  // HE.net password
	UpdateKey *string `uci:"updatekey,omitempty"` // HE.net update key

	// L2TP specific
	Server          *string `uci:"server,omitempty"`                                                   // L2TP server
	CheckupInterval *int    `uci:"checkup_interval,omitempty,unit=seconds,range=10-3600" default:"30"` // L2TP checkup interval
	PPPDOptions     *string `uci:"pppd_options,omitempty"`                                             // Additional pppd options
}

// WireGuardConfig handles WireGuard VPN configuration
type WireGuardConfig struct {
	PrivateKey          *string  `uci:"private_key,omitempty"`                                                 // WireGuard private key
	PublicKey           *string  `uci:"public_key,omitempty"`                                                  // WireGuard public key
	ListenPort          *int     `uci:"listen_port,omitempty,range=1-65535" default:"51820"`                   // Listen port
	Addresses           []string `uci:"addresses,omitempty,join= "`                                            // Interface addresses
	PreSharedKey        *string  `uci:"preshared_key,omitempty"`                                               // Pre-shared key
	Endpoint            *string  `uci:"endpoint,omitempty"`                                                    // Peer endpoint
	AllowedIPs          []string `uci:"allowed_ips,omitempty,join=,"`                                          // Allowed IP ranges
	Route               []string `uci:"route,omitempty,join= "`                                                // Routes to add
	PersistentKeepalive *int     `uci:"persistent_keepalive,omitempty,unit=seconds,range=0-65535" default:"0"` // Keepalive interval
}

// MobileConfig handles mobile network protocols (3G, QMI, NCM, WWAN)
type MobileConfig struct {
	// Common mobile options
	Device   *string `uci:"device,omitempty"`                       // Device path
	APN      *string `uci:"apn,omitempty,case=lower"`               // Access Point Name
	PINCode  *string `uci:"pincode,omitempty"`                      // SIM PIN
	Username *string `uci:"username,omitempty"`                     // Username for auth
	Password *string `uci:"password,omitempty"`                     // Password for auth
	Auth     *string `uci:"auth,omitempty,enum=none,pap,chap,both"` // Authentication type
	Mode     *string `uci:"mode,omitempty,enum=lte,umts,gsm,auto"`  // Connection mode
	PlmnID   *string `uci:"plmnid,omitempty"`                       // PLMN ID

	// QMI specific
	PDPType *string `uci:"pdptype,omitempty,enum=ip,ipv6,ipv4v6" default:"ip"` // PDP context type
	Profile *int    `uci:"profile,omitempty,range=1-16" default:"1"`           // Connection profile

	// 3G/WWAN specific
	Service    *string `uci:"service,omitempty,enum=gprs,edge,umts,hsdpa,hsupa,hspa,lte"` // Service type
	InitString *string `uci:"initstring,omitempty"`                                       // Modem init string
	Delay      *int    `uci:"delay,omitempty,unit=seconds,range=0-60" default:"10"`       // Connection delay
}

// VirtualConfig handles virtualization protocols (GRE, VTI, VXLAN)
type VirtualConfig struct {
	// GRE options
	RemoteIP *string `uci:"remote,omitempty"`                              // Remote endpoint IP
	LocalIP  *string `uci:"local,omitempty"`                               // Local endpoint IP
	Key      *int    `uci:"key,omitempty,range=0-4294967295" default:"0"`  // GRE key
	ICSum    *bool   `uci:"icsum,omitempty,bool=0/1" default:"false"`      // Input checksum
	OCSum    *bool   `uci:"ocsum,omitempty,bool=0/1" default:"false"`      // Output checksum
	IKey     *int    `uci:"ikey,omitempty,range=0-4294967295" default:"0"` // Input key
	OKey     *int    `uci:"okey,omitempty,range=0-4294967295" default:"0"` // Output key

	// VXLAN options
	VNI   *int    `uci:"vni,omitempty,range=1-16777215" default:"1"`  // VXLAN Network Identifier
	Port  *int    `uci:"port,omitempty,range=1-65535" default:"4789"` // VXLAN port
	Group *string `uci:"group,omitempty"`                             // Multicast group
	Local *string `uci:"local,omitempty"`                             // Local interface

	// VTI options
	IIf    *string `uci:"iif,omitempty,case=lower"` // Input interface
	OIf    *string `uci:"oif,omitempty,case=lower"` // Output interface
	TunSrc *string `uci:"tunsrc,omitempty"`         // Tunnel source
	TunDst *string `uci:"tundst,omitempty"`         // Tunnel destination
}

// Custom UCI serialization for NetworkInterfaceConfig to handle protocol-specific logic
func (c *NetworkInterfaceConfig) ToUCI() (map[string]string, error) {
	// Create protocol-specific config based on proto field
	c.createProtocolConfig()

	// Use default UCI marshaling
	return uci.Marshal(c)
}

func (c *NetworkInterfaceConfig) FromUCI(data map[string]string) error {
	// First, parse metadata
	c.BaseConfig.FromUCI(data)

	// Clean data (remove metadata)
	cleanData := make(map[string]string)
	for k, v := range data {
		if !strings.HasPrefix(k, ".") {
			cleanData[k] = v
		}
	}

	// Extract proto field first to determine which config to create
	if proto, exists := cleanData["proto"]; exists {
		c.Proto = proto
	}

	// Create protocol-specific config based on proto field
	// This ensures pointer fields are initialized for flatten tags
	c.createProtocolConfig()

	// Unmarshal into struct
	if err := uci.Unmarshal(cleanData, c); err != nil {
		return err
	}

	return nil
}

// createProtocolConfig creates the appropriate protocol configuration based on proto field
func (c *NetworkInterfaceConfig) createProtocolConfig() {
	switch c.Proto {
	case "static":
		if c.StaticConfig == nil {
			c.StaticConfig = &StaticConfig{}
		}
	case "dhcp", "dhcpv6":
		if c.DHCPConfig == nil {
			c.DHCPConfig = &DHCPConfig{}
		}
	case "pppoe", "ppp":
		if c.PPPConfig == nil {
			c.PPPConfig = &PPPConfig{}
		}
	case "6in4", "6rd", "6to4", "dslite", "l2tp":
		if c.TunnelConfig == nil {
			c.TunnelConfig = &TunnelConfig{}
		}
	case "wireguard":
		if c.WireGuardConfig == nil {
			c.WireGuardConfig = &WireGuardConfig{}
		}
	case "3g", "qmi", "ncm", "wwan":
		if c.MobileConfig == nil {
			c.MobileConfig = &MobileConfig{}
		}
	case "gre", "vti", "vxlan":
		if c.VirtualConfig == nil {
			c.VirtualConfig = &VirtualConfig{}
		}
	}
}
