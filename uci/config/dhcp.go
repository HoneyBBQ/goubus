package config

import (
	"github.com/honeybbq/goubus"
)

// DnsmasqConfig represents the configuration for the main dnsmasq service.
// It implements the ConfigModel interface using UCI tags.
type DnsmasqConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields (use value types) - dnsmasq的基础配置都是可选的，没有明确的required字段
	// 但是我们可以认为某些基础字段是必需的，这里都使用指针类型保持一致性

	// DNS Configuration - 所有字段都是可选的
	DomainNeeded *bool   `uci:"domainneeded,omitempty,bool=0/1" default:"false"`     // Required=no: Never forward plain names
	BogusPRIV    *bool   `uci:"boguspriv,omitempty,bool=0/1" default:"true"`         // Required=no: Bogus private reverse lookups
	FilterWin2k  *bool   `uci:"filterwin2k,omitempty,bool=0/1" default:"false"`      // Required=no: Filter Windows DNS requests
	LocalHost    *bool   `uci:"localise_queries,omitempty,bool=0/1" default:"false"` // Required=no: Localise hostname queries
	LocalService *string `uci:"local,omitempty" default:"none"`                      // Required=no: Local service only
	Domain       *string `uci:"domain,omitempty,case=lower" default:"none"`          // Required=no: Local domain suffix
	ExpandHosts  *bool   `uci:"expandhosts,omitempty,bool=0/1" default:"false"`      // Required=no: Expand simple names
	NonNegCache  *bool   `uci:"nonegcache,omitempty,bool=0/1" default:"false"`       // Required=no: Disable negative caching
	AuthorityDNS *bool   `uci:"authoritative,omitempty,bool=0/1" default:"false"`    // Required=no: Authoritative DNS server
	ReadEthers   *bool   `uci:"readethers,omitempty,bool=0/1" default:"false"`       // Required=no: Read /etc/ethers file

	// Resolver Configuration
	ResolvFile  *string `uci:"resolvfile,omitempty" default:"/tmp/resolv.conf.auto"` // Required=no: Path to resolv.conf
	NoResolv    *bool   `uci:"noresolv,omitempty,bool=0/1" default:"false"`          // Required=no: Ignore resolv.conf
	StrictOrder *bool   `uci:"strictorder,omitempty,bool=0/1" default:"false"`       // Required=no: DNS server strict order
	AllServers  *bool   `uci:"allservers,omitempty,bool=0/1" default:"false"`        // Required=no: Query all servers

	// Logging and Debugging
	LogQueries *bool `uci:"logqueries,omitempty,bool=0/1" default:"false"` // Required=no: Log DNS queries
	LogDHCP    *bool `uci:"logdhcp,omitempty,bool=0/1" default:"false"`    // Required=no: Log DHCP transactions

	// Network Configuration
	Port            *int      `uci:"port,omitempty,range=1-65535" default:"53"`                   // Required=no: DNS listening port
	QueryPort       *int      `uci:"queryport,omitempty,range=0-65535" default:"0"`               // Required=no: DNS query source port (0=random)
	Interface       *[]string `uci:"interface,omitempty,join= ,case=lower" default:"none"`        // Required=no: Listening interfaces
	ExceptInterface *[]string `uci:"except_interface,omitempty,join= ,case=lower" default:"none"` // Required=no: Excluded interfaces
	ListenAddress   *[]string `uci:"listen_address,omitempty,join= " default:"none"`              // Required=no: Listen addresses
	BindInterfaces  *bool     `uci:"bind_interfaces,omitempty,bool=0/1" default:"false"`          // Required=no: Bind only to interfaces
	BindDynamic     *bool     `uci:"bind_dynamic,omitempty,bool=0/1" default:"false"`             // Required=no: Bind to dynamic interfaces

	// Performance and Limits
	DnsForwardMax *int `uci:"dnsforwardmax,omitempty,range=1-10000" default:"150"`    // Required=no: Max concurrent DNS queries
	CacheSize     *int `uci:"cachesize,omitempty,range=0-10000" default:"150"`        // Required=no: DNS cache size (0=disable)
	EDnsPacketMax *int `uci:"ednspacketmax,omitempty,range=512-65535" default:"4096"` // Required=no: EDNS packet size

	// TTL Configuration (in seconds)
	LocalTTL    *int `uci:"localttl,omitempty,unit=seconds,range=0-86400" default:"none"`    // Required=no: TTL for local names in seconds
	NegTTL      *int `uci:"negttl,omitempty,unit=seconds,range=0-86400" default:"none"`      // Required=no: Negative caching TTL in seconds
	MaxTTL      *int `uci:"maxttl,omitempty,unit=seconds,range=0-86400" default:"none"`      // Required=no: Maximum TTL in seconds
	MinCacheTTL *int `uci:"mincachettl,omitempty,unit=seconds,range=0-86400" default:"none"` // Required=no: Minimum cache TTL in seconds
	MaxCacheTTL *int `uci:"maxcachettl,omitempty,unit=seconds,range=0-86400" default:"none"` // Required=no: Maximum cache TTL in seconds

	// DHCP Configuration
	DhcpLeasesMax *int    `uci:"dhcpleasemax,omitempty,range=1-65535" default:"150"`                                                         // Required=no: Maximum DHCP leases
	LeaseDuration *string `uci:"leasetime,omitempty" default:"12h"`                                                                          // Required=no: Default lease time (with unit suffix)
	LeasesFile    *string `uci:"leasefile,omitempty" default:"/tmp/dhcp.leases"`                                                             // Required=no: DHCP leases file
	DhcpScript    *string `uci:"dhcpscript,omitempty" default:"none"`                                                                        // Required=no: DHCP event script
	DhcpBoot      *string `uci:"dhcp_boot,omitempty" default:"none"`                                                                         // Required=no: DHCP boot options
	DhcpMatch     *string `uci:"dhcp_match,omitempty,enum=set,tag,vendorclass,userclass,mac,circuitid,remoteid,subscriberid" default:"none"` // Required=no: DHCP client matching

	// Security and Filtering
	DNSSecCheckUnsigned *bool     `uci:"dnsseccheckunsigned,omitempty,bool=0/1" default:"false"`   // Required=no: DNSSEC check unsigned
	DNSSec              *bool     `uci:"dnssec,omitempty,bool=0/1" default:"false"`                // Required=no: Enable DNSSEC
	TrustAnchor         *[]string `uci:"trust_anchor,omitempty,join= " default:"none"`             // Required=no: DNSSEC trust anchors
	RebindProtection    *bool     `uci:"rebind_protection,omitempty,bool=0/1" default:"true"`      // Required=no: DNS rebinding protection
	RebindLocalhost     *bool     `uci:"rebind_localhost,omitempty,bool=0/1" default:"false"`      // Required=no: Allow rebinding to localhost
	RebindDomain        *[]string `uci:"rebind_domain,omitempty,join= ,case=lower" default:"none"` // Required=no: Rebinding domain whitelist
	StopDNSRebind       *bool     `uci:"stop_dns_rebind,omitempty,bool=0/1" default:"true"`        // Required=no: Stop DNS rebinding

	// Advanced Options
	Server         *[]string `uci:"server,omitempty,join= " default:"none"`         // Required=no: Upstream DNS servers
	RevServer      *[]string `uci:"rev_server,omitempty,join= " default:"none"`     // Required=no: Reverse DNS servers
	Address        *[]string `uci:"address,omitempty,join= " default:"none"`        // Required=no: Address entries
	Bogus_nxdomain *[]string `uci:"bogus_nxdomain,omitempty,join= " default:"none"` // Required=no: Bogus NXDOMAIN IPs
	Confdir        *string   `uci:"confdir,omitempty" default:"none"`               // Required=no: Configuration directory
	Addnhosts      *[]string `uci:"addnhosts,omitempty,join= " default:"none"`      // Required=no: Additional hosts files

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// DHCPPoolConfig represents the configuration for a DHCP pool (dhcp section).
// It implements the ConfigModel interface using UCI tags.
type DHCPPoolConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - interface是DHCP池的必需标识
	Interface string `uci:"interface,case=lower" default:"lan"` // Required=yes: Interface name

	// Basic pool configuration - 其他都是可选的
	Start       *int    `uci:"start,omitempty,range=1-254" default:"100"`     // Required=no: Start of IP range
	Limit       *int    `uci:"limit,omitempty,range=1-254" default:"150"`     // Required=no: Number of addresses
	LeaseTme    *string `uci:"leasetime,omitempty" default:"12h"`             // Required=no: Lease time (with unit suffix)
	DynamicDHCP *bool   `uci:"dynamicdhcp,omitempty,bool=0/1" default:"true"` // Required=no: Enable dynamic DHCP
	Force       *bool   `uci:"force,omitempty,bool=0/1" default:"false"`      // Required=no: Force enable
	NetMask     *string `uci:"netmask,omitempty" default:"none"`              // Required=no: Network mask
	Ignore      *bool   `uci:"ignore,omitempty,bool=0/1" default:"false"`     // Required=no: Ignore this pool

	// DHCP Options
	DhcpOption      *[]string `uci:"dhcp_option,omitempty,join= " default:"none"`       // Required=no: DHCP options
	DhcpOptionForce *[]string `uci:"dhcp_option_force,omitempty,join= " default:"none"` // Required=no: Forced DHCP options

	// Domain and hostname settings
	Domain *string `uci:"domain,omitempty,case=lower" default:"none"` // Required=no: Domain name
	Local  *string `uci:"local,omitempty,case=lower" default:"none"`  // Required=no: Local domain

	// IPv4 and IPv6 Configuration
	DhcpV4        *string   `uci:"dhcpv4,omitempty,enum=server,disabled,relay" default:"server"`          // Required=no: DHCPv4 mode
	DhcpV6        *string   `uci:"dhcpv6,omitempty,enum=server,disabled,relay,hybrid" default:"disabled"` // Required=no: DHCPv6 mode
	Ra            *string   `uci:"ra,omitempty,enum=disabled,server,relay,hybrid" default:"disabled"`     // Required=no: Router advertisements
	RaManagement  *int      `uci:"ra_management,omitempty,range=0-3" default:"1"`                         // Required=no: RA management flags (0-3)
	RaDefault     *int      `uci:"ra_default,omitempty,range=0-2" default:"0"`                            // Required=no: RA default route preference
	RaFlags       *[]string `uci:"ra_flags,omitempty,join= " default:"none"`                              // Required=no: RA flags
	RaService     *string   `uci:"ra_service,omitempty,enum=server,relay,hybrid" default:"none"`          // Required=no: RA service
	RaPreference  *string   `uci:"ra_preference,omitempty,enum=low,medium,high" default:"medium"`         // Required=no: RA preference
	RaMaxInterval *int      `uci:"ra_maxinterval,omitempty,unit=seconds,range=4-1800" default:"600"`      // Required=no: RA max interval in seconds
	RaMinInterval *int      `uci:"ra_mininterval,omitempty,unit=seconds,range=3-1350" default:"200"`      // Required=no: RA min interval in seconds
	RaLifetime    *int      `uci:"ra_lifetime,omitempty,unit=seconds,range=0-7200" default:"1800"`        // Required=no: RA lifetime in seconds
	RaAdvRoute    *[]string `uci:"ra_advroute,omitempty,join= " default:"none"`                           // Required=no: RA advertised routes

	// IPv6 Configuration
	RaHopLimit       *int      `uci:"ra_hoplimit,omitempty,range=0-255" default:"none"`                   // Required=no: RA hop limit
	RaMTU            *int      `uci:"ra_mtu,omitempty,range=1280-65535" default:"none"`                   // Required=no: RA MTU
	RaReachableTime  *int      `uci:"ra_reachabletime,omitempty,unit=ms,range=0-3600000" default:"none"`  // Required=no: RA reachable time in ms
	RaRetransmitTime *int      `uci:"ra_retranstime,omitempty,unit=ms,range=0-4294967295" default:"none"` // Required=no: RA retransmit time in ms
	IP6Assign        *int      `uci:"ip6assign,omitempty,range=0-64" default:"none"`                      // Required=no: IPv6 prefix assignment length
	IP6Hint          *string   `uci:"ip6hint,omitempty" default:"none"`                                   // Required=no: IPv6 hint
	IP6Class         *[]string `uci:"ip6class,omitempty,join= ,case=lower" default:"none"`                // Required=no: IPv6 class

	// NDP Configuration
	NdpRelay *string `uci:"ndp,omitempty,enum=disabled,relay,hybrid" default:"disabled"` // Required=no: NDP relay
	NdpProxy *bool   `uci:"ndproxy_routing,omitempty,bool=0/1" default:"none"`           // Required=no: NDP proxy routing
	NdpPing  *bool   `uci:"ndproxy_ping,omitempty,bool=0/1" default:"none"`              // Required=no: NDP proxy ping

	// DNS Configuration
	DnsForward *bool   `uci:"dns,omitempty,bool=0/1" default:"true"`                          // Required=no: DNS forwarding
	DnsService *string `uci:"dns_service,omitempty,enum=dnsmasq,unbound,none" default:"none"` // Required=no: DNS service

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// StaticHostConfig represents the configuration for a static host (host section).
// It implements the ConfigModel interface using UCI tags.
type StaticHostConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 静态主机需要一个标识符，通常是name或mac
	Name string `uci:"name,case=lower" default:"none"` // Required=yes: Host identifier

	// Basic host configuration - 其他字段都是可选的
	MAC      *string `uci:"mac,omitempty,case=lower" default:"none"`      // Required=no: MAC address
	IP       *string `uci:"ip,omitempty" default:"none"`                  // Required=no: IPv4 address
	IPv6     *string `uci:"ip6,omitempty" default:"none"`                 // Required=no: IPv6 address
	LeaseTme *string `uci:"leasetime,omitempty" default:"infinite"`       // Required=no: Lease time (with unit suffix or "infinite")
	DnsEntry *bool   `uci:"dns,omitempty,bool=0/1" default:"false"`       // Required=no: DNS entry
	Hostname *string `uci:"hostname,omitempty,case=lower" default:"none"` // Required=no: Hostname
	Instance *string `uci:"instance,omitempty" default:"none"`            // Required=no: Instance name

	// DHCP Options
	DhcpOption *[]string `uci:"dhcp_option,omitempty,join= " default:"none"` // Required=no: DHCP options

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// DomainConfig represents the configuration for a domain record (domain section).
// It implements the ConfigModel interface using UCI tags.
type DomainConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 域名记录需要名称和IP
	Name string `uci:"name,case=lower" default:"none"` // Required=yes: Domain name
	IP   string `uci:"ip" default:"none"`              // Required=yes: IP address

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// CnameConfig represents the configuration for a CNAME record (cname section).
// It implements the ConfigModel interface using UCI tags.
type CnameConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - CNAME记录需要别名和目标
	Cname  string `uci:"cname,case=lower" default:"none"`  // Required=yes: CNAME alias
	Target string `uci:"target,case=lower" default:"none"` // Required=yes: Target domain

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// MXConfig represents the configuration for an MX record (mx section).
// It implements the ConfigModel interface using UCI tags.
type MXConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - MX记录需要域名和邮件中继
	Domain string `uci:"domain,case=lower" default:"none"` // Required=yes: Domain name
	Relay  string `uci:"relay,case=lower" default:"none"`  // Required=yes: Mail relay

	// Optional fields
	Pref *int `uci:"pref,omitempty,range=0-65535" default:"10"` // Required=no: Preference

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// SRVConfig represents the configuration for an SRV record (srv section).
// It implements the ConfigModel interface using UCI tags.
type SRVConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - SRV记录需要服务、目标和端口
	Srv    string `uci:"srv,case=lower" default:"none"`     // Required=yes: Service
	Target string `uci:"target,case=lower" default:"none"`  // Required=yes: Target server
	Port   int    `uci:"port,range=1-65535" default:"none"` // Required=yes: Port number

	// Optional fields
	Class  *string `uci:"class,omitempty" default:"none"`             // Required=no: Service class
	Weight *int    `uci:"weight,omitempty,range=0-65535" default:"5"` // Required=no: Weight

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// TXTConfig represents the configuration for a TXT record (txt section).
// It implements the ConfigModel interface using UCI tags.
type TXTConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - TXT记录需要名称和值
	Name  string `uci:"name,case=lower" default:"none"` // Required=yes: Record name
	Value string `uci:"value" default:"none"`           // Required=yes: TXT value

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// HostRecordConfig represents the configuration for a host record (hostrecord section).
// It implements the ConfigModel interface using UCI tags.
type HostRecordConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 主机记录需要名称和IP
	Name string `uci:"name,case=lower" default:"none"` // Required=yes: Record name
	IP   string `uci:"ip" default:"none"`              // Required=yes: IP address

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// RelayConfig represents the configuration for DHCP relay (relay section).
// It implements the ConfigModel interface using UCI tags.
type RelayConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - DHCP中继需要ID和服务器地址
	ID         string `uci:"id" default:"none"`          // Required=yes: Relay ID
	ServerAddr string `uci:"server_addr" default:"none"` // Required=yes: Server address

	// Optional fields
	Interface *string `uci:"interface,omitempty,case=lower" default:"none"` // Required=no: Interface
	LocalAddr *string `uci:"local_addr,omitempty" default:"none"`           // Required=no: Local address

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}
