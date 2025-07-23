package config

import (
	"github.com/honeybbq/goubus"
)

// FirewallDefaultsConfig represents the firewall defaults configuration.
// It implements the ConfigModel interface using UCI tags.
type FirewallDefaultsConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Optional fields (use pointer types) - 防火墙默认配置都是可选的
	// Basic firewall policy
	SynFlood     *bool   `uci:"syn_flood,omitempty,bool=0/1" default:"true"`                // Required=no: SYN flood protection
	Input        *string `uci:"input,omitempty,enum=ACCEPT,REJECT,DROP" default:"ACCEPT"`   // Required=no: Default input policy
	Output       *string `uci:"output,omitempty,enum=ACCEPT,REJECT,DROP" default:"ACCEPT"`  // Required=no: Default output policy
	Forward      *string `uci:"forward,omitempty,enum=ACCEPT,REJECT,DROP" default:"REJECT"` // Required=no: Default forward policy
	DropInvalid  *bool   `uci:"drop_invalid,omitempty,bool=0/1" default:"false"`            // Required=no: Drop invalid packets
	DisableIPv6  *bool   `uci:"disable_ipv6,omitempty,bool=0/1" default:"false"`            // Required=no: Disable IPv6 firewall
	CustomChains *bool   `uci:"custom_chains,omitempty,bool=0/1" default:"true"`            // Required=no: Custom chains

	// Hardware acceleration
	FlowOffloading   *bool `uci:"flow_offloading,omitempty,bool=0/1" default:"false"`    // Required=no: Software flow offloading
	FlowOffloadingHW *bool `uci:"flow_offloading_hw,omitempty,bool=0/1" default:"false"` // Required=no: Hardware flow offloading

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// FirewallZoneConfig represents a firewall zone configuration.
// It implements the ConfigModel interface using UCI tags.
type FirewallZoneConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 防火墙区域需要名称
	Name string `uci:"name,case=lower" default:"none"` // Required=yes: Zone name

	// Optional fields (use pointer types)
	// Basic zone configuration
	Input   *string `uci:"input,omitempty,enum=ACCEPT,REJECT,DROP" default:"ACCEPT"`   // Required=no: Input policy
	Output  *string `uci:"output,omitempty,enum=ACCEPT,REJECT,DROP" default:"ACCEPT"`  // Required=no: Output policy
	Forward *string `uci:"forward,omitempty,enum=ACCEPT,REJECT,DROP" default:"REJECT"` // Required=no: Forward policy

	// NAT and masquerading
	Masq             *bool     `uci:"masq,omitempty,bool=0/1" default:"false"`               // Required=no: Enable masquerading (IPv4)
	Masq6            *bool     `uci:"masq6,omitempty,bool=0/1" default:"false"`              // Required=no: Enable masquerading (IPv6)
	MasqSrc          *[]string `uci:"masq_src,omitempty,join= " default:"none"`              // Required=no: Masquerade source addresses
	MasqDest         *[]string `uci:"masq_dest,omitempty,join= " default:"none"`             // Required=no: Masquerade destination addresses
	MasqAllowInvalid *bool     `uci:"masq_allow_invalid,omitempty,bool=0/1" default:"false"` // Required=no: Allow invalid masquerade packets

	// MTU and MSS
	MSS_clamping *bool `uci:"mtu_fix,omitempty,bool=0/1" default:"false"` // Required=no: MSS clamping

	// Zone interfaces and networks
	Network *[]string `uci:"network,omitempty,join= ,case=lower" default:"none"` // Required=no: Networks in this zone
	Device  *[]string `uci:"device,omitempty,join= ,case=lower" default:"none"`  // Required=no: Devices in this zone
	Subnet  *[]string `uci:"subnet,omitempty,join= " default:"none"`             // Required=no: Subnets in this zone

	// Logging
	LogLimit *string `uci:"log_limit,omitempty" default:"10/minute"` // Required=no: Log limit (iptables format)
	Log      *bool   `uci:"log,omitempty,bool=0/1" default:"false"`  // Required=no: Enable logging

	// Protocol family
	Family *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"any"` // Required=no: IP family

	// Advanced options
	ExtraOpts *[]string `uci:"extra,omitempty,join= " default:"none"`      // Required=no: Extra iptables options
	ExtraSrc  *[]string `uci:"extra_src,omitempty,join= " default:"none"`  // Required=no: Extra source options
	ExtraDest *[]string `uci:"extra_dest,omitempty,join= " default:"none"` // Required=no: Extra destination options

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// FirewallRuleConfig represents a firewall rule configuration.
// It implements the ConfigModel interface using UCI tags.
type FirewallRuleConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 防火墙规则通常需要名称和目标
	Name   string `uci:"name,case=lower" default:"none"`                          // Required=yes: Rule name
	Target string `uci:"target,enum=ACCEPT,REJECT,DROP,MARK,DSCP" default:"DROP"` // Required=yes: Rule target

	// Optional fields (use pointer types)
	// Basic rule configuration
	Family  *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"any"` // Required=no: IP family
	Enabled *bool   `uci:"enabled,omitempty,bool=0/1" default:"true"`         // Required=no: Enable rule

	// Source configuration
	Src     *string   `uci:"src,omitempty,case=lower" default:"none"`            // Required=no: Source zone
	SrcIP   *[]string `uci:"src_ip,omitempty,join= " default:"none"`             // Required=no: Source IP addresses
	SrcMac  *[]string `uci:"src_mac,omitempty,join= ,case=lower" default:"none"` // Required=no: Source MAC addresses
	SrcPort *[]string `uci:"src_port,omitempty,join= " default:"none"`           // Required=no: Source ports

	// Destination configuration
	Dest     *string   `uci:"dest,omitempty,case=lower" default:"none"`  // Required=no: Destination zone
	DestIP   *[]string `uci:"dest_ip,omitempty,join= " default:"none"`   // Required=no: Destination IP addresses
	DestPort *[]string `uci:"dest_port,omitempty,join= " default:"none"` // Required=no: Destination ports

	// Protocol and packet matching
	Proto    *[]string `uci:"proto,omitempty,join= ,enum=tcp,udp,icmp,esp,ah,sctp,all" default:"all"` // Required=no: Protocols
	ICMPType *[]string `uci:"icmp_type,omitempty,join= " default:"any"`                               // Required=no: ICMP types

	// Packet marking and classification
	SetMark  *int    `uci:"set_mark,omitempty,range=0-4294967295" default:"none"`  // Required=no: Set packet mark
	SetXMark *int    `uci:"set_xmark,omitempty,range=0-4294967295" default:"none"` // Required=no: Set extended mark
	SetDSCP  *int    `uci:"set_dscp,omitempty,range=0-63" default:"none"`          // Required=no: Set DSCP value
	MarkMask *string `uci:"mark,omitempty" default:"none"`                         // Required=no: Mark mask

	// Rate limiting
	Limit      *string `uci:"limit,omitempty" default:"none"`                     // Required=no: Rate limit (iptables format)
	LimitBurst *int    `uci:"limit_burst,omitempty,range=1-10000" default:"none"` // Required=no: Rate limit burst

	// Time-based rules
	UTCTime   *bool     `uci:"utc_time,omitempty,bool=0/1" default:"false"`        // Required=no: Use UTC time
	StartDate *string   `uci:"start_date,omitempty" default:"none"`                // Required=no: Start date (YYYY-MM-DD)
	StopDate  *string   `uci:"stop_date,omitempty" default:"none"`                 // Required=no: Stop date (YYYY-MM-DD)
	StartTime *string   `uci:"start_time,omitempty" default:"none"`                // Required=no: Start time (HH:MM)
	StopTime  *string   `uci:"stop_time,omitempty" default:"none"`                 // Required=no: Stop time (HH:MM)
	Weekdays  *[]string `uci:"weekdays,omitempty,join=,case=lower" default:"none"` // Required=no: Weekdays (mon,tue,etc)
	Monthdays *[]string `uci:"monthdays,omitempty,join=," default:"none"`          // Required=no: Month days (1-31)

	// Advanced options
	Extra  *string `uci:"extra,omitempty" default:"none"`                                       // Required=no: Extra iptables options
	Helper *string `uci:"helper,omitempty,enum=ftp,sip,h323,pptp,snmp,tftp,irc" default:"none"` // Required=no: Connection helper

	// Extra fields for additional options not explicitly defined
	ExtraMap map[string]string `uci:",flatten,omitempty"`
}

// FirewallRedirectConfig represents a firewall redirect/port forward configuration.
// It implements the ConfigModel interface using UCI tags.
type FirewallRedirectConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 端口转发需要名称和目标
	Name   string `uci:"name,case=lower" default:"none"`                // Required=yes: Redirect name
	Target string `uci:"target,enum=DNAT,SNAT,REDIRECT" default:"DNAT"` // Required=yes: Redirect target

	// Optional fields (use pointer types)
	// Basic redirect configuration
	Family  *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"any"` // Required=no: IP family
	Enabled *bool   `uci:"enabled,omitempty,bool=0/1" default:"true"`         // Required=no: Enable redirect

	// Source configuration
	Src      *string   `uci:"src,omitempty,case=lower" default:"none"`            // Required=no: Source zone
	SrcIP    *[]string `uci:"src_ip,omitempty,join= " default:"none"`             // Required=no: Source IP addresses
	SrcMac   *[]string `uci:"src_mac,omitempty,join= ,case=lower" default:"none"` // Required=no: Source MAC addresses
	SrcPort  *[]string `uci:"src_port,omitempty,join= " default:"none"`           // Required=no: Source ports
	SrcDPort *[]string `uci:"src_dport,omitempty,join= " default:"none"`          // Required=no: Source destination ports
	SrcDIP   *string   `uci:"src_dip,omitempty" default:"none"`                   // Required=no: Source destination IP

	// Destination configuration
	Dest     *string `uci:"dest,omitempty,case=lower" default:"none"` // Required=no: Destination zone
	DestIP   *string `uci:"dest_ip,omitempty" default:"none"`         // Required=no: Destination IP address
	DestPort *string `uci:"dest_port,omitempty" default:"none"`       // Required=no: Destination port

	// Protocol configuration
	Proto *[]string `uci:"proto,omitempty,join= ,enum=tcp,udp,tcpudp,icmp,esp,ah,sctp,all" default:"tcp udp"` // Required=no: Protocols

	// Reflection configuration (for NAT loopback)
	ReflectionZone *[]string `uci:"reflection_zone,omitempty,join= ,case=lower" default:"none"` // Required=no: Reflection zones
	Reflection     *bool     `uci:"reflection,omitempty,bool=0/1" default:"true"`               // Required=no: Enable reflection
	ReflectionSrc  *string   `uci:"reflection_src,omitempty" default:"none"`                    // Required=no: Reflection source

	// Advanced options
	Helper *string `uci:"helper,omitempty,enum=ftp,sip,h323,pptp,snmp,tftp,irc" default:"none"` // Required=no: Connection helper
	Extra  *string `uci:"extra,omitempty" default:"none"`                                       // Required=no: Extra iptables options

	// Extra fields for additional options not explicitly defined
	ExtraMap map[string]string `uci:",flatten,omitempty"`
}

// FirewallForwardingConfig represents inter-zone forwarding configuration.
// It implements the ConfigModel interface using UCI tags.
type FirewallForwardingConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 转发规则需要源和目标区域
	Src  string `uci:"src,case=lower" default:"none"`  // Required=yes: Source zone
	Dest string `uci:"dest,case=lower" default:"none"` // Required=yes: Destination zone

	// Optional fields (use pointer types)
	// Basic forwarding configuration
	Name    *string `uci:"name,omitempty,case=lower" default:"none"`          // Required=no: Forwarding name
	Family  *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"any"` // Required=no: IP family
	Enabled *bool   `uci:"enabled,omitempty,bool=0/1" default:"true"`         // Required=no: Enable forwarding

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// FirewallIncludeConfig represents firewall include configuration for custom scripts.
// It implements the ConfigModel interface using UCI tags.
type FirewallIncludeConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - 包含配置需要路径
	Path string `uci:"path" default:"none"` // Required=yes: Path to script file

	// Optional fields (use pointer types)
	// Include configuration
	Type   *string `uci:"type,omitempty,enum=script,restore" default:"script"` // Required=no: Include type
	Family *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"any"`   // Required=no: IP family
	Reload *bool   `uci:"reload,omitempty,bool=0/1" default:"false"`           // Required=no: Reload on config changes

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// FirewallNATConfig represents NAT configuration (nat section).
// It implements the ConfigModel interface using UCI tags.
type FirewallNATConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - NAT规则需要目标
	Target string `uci:"target,enum=MASQUERADE,SNAT,DNAT" default:"MASQUERADE"` // Required=yes: NAT target

	// Optional fields (use pointer types)
	// Basic NAT configuration
	Name   *string `uci:"name,omitempty,case=lower" default:"none"`           // Required=no: NAT rule name
	Family *string `uci:"family,omitempty,enum=ipv4,ipv6,any" default:"ipv4"` // Required=no: IP family

	// Source configuration
	Src   *string `uci:"src,omitempty,case=lower" default:"none"` // Required=no: Source zone
	SrcIP *string `uci:"src_ip,omitempty" default:"none"`         // Required=no: Source IP/subnet

	// Destination configuration
	Dest   *string `uci:"dest,omitempty,case=lower" default:"none"` // Required=no: Destination zone
	DestIP *string `uci:"dest_ip,omitempty" default:"none"`         // Required=no: Destination IP

	// Protocol configuration
	Proto *string `uci:"proto,omitempty,enum=tcp,udp,icmp,esp,ah,sctp,all" default:"all"` // Required=no: Protocol

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}
