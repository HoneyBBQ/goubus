package types

import (
	"encoding/json"
	"time"
)

// =============================================================================
// SYSTEM TYPES
// =============================================================================

// SystemInfo holds runtime system information from 'ubus call system info'.
type SystemInfo struct {
	LocalTime int64   `json:"localtime"`
	Uptime    int     `json:"uptime"`
	Load      []int   `json:"load"`
	Memory    Memory  `json:"memory"`
	Root      Storage `json:"root"`
	Tmp       Storage `json:"tmp"`
	Swap      Swap    `json:"swap"`
}

// SystemBoardInfo holds hardware-specific information from 'ubus call system board'.
type SystemBoardInfo struct {
	Kernel    string  `json:"kernel"`
	Hostname  string  `json:"hostname"`
	System    string  `json:"system"`
	Model     string  `json:"model"`
	BoardName string  `json:"board_name"`
	Release   Release `json:"release"`
}

// Release holds release information.
type Release struct {
	Distribution string `json:"distribution"`
	Version      string `json:"version"`
	Revision     string `json:"revision"`
	Target       string `json:"target"`
	Description  string `json:"description"`
	BuildDate    string `json:"builddate"`
}

// Memory holds memory usage statistics.
type Memory struct {
	Total     int `json:"total"`
	Free      int `json:"free"`
	Shared    int `json:"shared"`
	Buffered  int `json:"buffered"`
	Available int `json:"available"`
	Cached    int `json:"cached"`
}

// Storage holds storage usage statistics.
type Storage struct {
	Total int `json:"total"`
	Free  int `json:"free"`
	Used  int `json:"used"`
	Avail int `json:"avail"`
}

// Swap holds swap usage statistics.
type Swap struct {
	Total int `json:"total"`
	Free  int `json:"free"`
}

// =============================================================================
// DHCP TYPES
// =============================================================================

// DHCPLeases represents both IPv4 and IPv6 DHCP leases.
type DHCPLeases struct {
	IPv4Leases []DhcpIPv4Lease `json:"dhcp_leases"`
	IPv6Leases []DhcpIPv6Lease `json:"dhcp6_leases"`
}

// DhcpIPv4Lease represents a single DHCP IPv4 lease.
type DhcpIPv4Lease struct {
	Expires  int    `json:"expires"`
	Hostname string `json:"hostname"`
	Macaddr  string `json:"macaddr"`
	DUID     string `json:"duid"`
	IPAddr   string `json:"ipaddr"`
}

// DhcpIPv6Lease represents a single DHCP IPv6 lease.
type DhcpIPv6Lease struct {
	Expires  int      `json:"expires"`
	Hostname string   `json:"hostname"`
	Macaddr  string   `json:"macaddr"`
	DUID     string   `json:"duid"`
	IPAddr   string   `json:"ip6addr"`
	IPAddrs  []string `json:"ip6addrs"`
}

// AddLeaseRequest represents the parameters for adding a static DHCP lease.
type AddLeaseRequest struct {
	IP        string `json:"ip"`
	Mac       string `json:"mac"`
	DUID      string `json:"duid,omitempty"`
	HostID    string `json:"hostid,omitempty"`
	LeaseTime string `json:"leasetime,omitempty"`
	Name      string `json:"name,omitempty"`
}

// =============================================================================
// NETWORK TYPES
// =============================================================================

// NetworkInterfaceInfo wraps network interface information with its name.
type NetworkInterfaceInfo struct {
	Interface        string `json:"interface"`
	NetworkInterface        // Embedded for direct access to interface data
}

// NetworkInterface represents detailed network interface information.
type NetworkInterface struct {
	Up                   Bool                         `json:"up"`
	Pending              Bool                         `json:"pending"`
	Available            Bool                         `json:"available"`
	Autostart            Bool                         `json:"autostart"`
	Dynamic              Bool                         `json:"dynamic"`
	Uptime               int                          `json:"uptime"`
	L3Device             string                       `json:"l3_device"`
	Proto                string                       `json:"proto"`
	Device               string                       `json:"device"`
	Updated              []string                     `json:"updated"`
	Metric               int                          `json:"metric"`
	DNSServer            []string                     `json:"dns-server"`
	DNSSearch            []string                     `json:"dns-search"`
	DNSMetric            int                          `json:"dns_metric"`
	Delegation           Bool                         `json:"delegation"`
	Delegates            []string                     `json:"delegates"`
	Zone                 string                       `json:"zone"`
	IPv4Address          []NetworkInterfaceAddress    `json:"ipv4-address"`
	IPv6Address          []NetworkInterfaceAddress    `json:"ipv6-address"`
	IPv6AddressGenerated []NetworkInterfaceAddress    `json:"ipv6-address-generated"`
	IPv6Prefix           []NetworkInterfaceIPv6Prefix `json:"ipv6-prefix"`
	IPv6PrefixAssignment []NetworkInterfaceIPv6Prefix `json:"ipv6-prefix-assignment"`
	Route                []NetworkInterfaceRoute      `json:"route"`
	Neighbors            []NetworkInterfaceNeighbor   `json:"neighbors"`
	Inactive             NetworkInterfaceInactive     `json:"inactive"`
	Data                 NetworkInterfaceData         `json:"data"`
}

// NetworkInterfaceAddress represents an IP address assignment.
type NetworkInterfaceAddress struct {
	Address string `json:"address"`
	Mask    int    `json:"mask"`
}

// NetworkInterfaceRoute represents a routing table entry.
type NetworkInterfaceRoute struct {
	Target  string `json:"target"`
	Mask    int    `json:"mask"`
	Nexthop string `json:"nexthop"`
	Metric  int    `json:"metric"`
	Valid   int    `json:"valid"`
	Source  string `json:"source"`
}

// NetworkInterfaceIPv6Prefix represents an IPv6 prefix assignment.
type NetworkInterfaceIPv6Prefix struct {
	Address        string                   `json:"address"`
	Mask           int                      `json:"mask"`
	Preferred      int                      `json:"preferred"`
	Valid          int                      `json:"valid"`
	Class          string                   `json:"class"`
	AssignedLength int                      `json:"assigned-length"`
	LocalAddress   *NetworkInterfaceAddress `json:"local-address,omitempty"`
}

// NetworkInterfaceNeighbor represents a neighbor cache entry.
type NetworkInterfaceNeighbor struct {
	Address string `json:"address"`
	MAC     string `json:"mac"`
	Router  Bool   `json:"router"`
	State   int    `json:"state"`
}

// NetworkInterfaceInactive represents inactive interface configuration.
type NetworkInterfaceInactive struct {
	IPv4Address []NetworkInterfaceAddress `json:"ipv4-address"`
	IPv6Address []NetworkInterfaceAddress `json:"ipv6-address"`
	Route       []NetworkInterfaceRoute   `json:"route"`
}

// NetworkInterfaceData represents additional interface data.
type NetworkInterfaceData struct {
	// This is typically empty or contains protocol-specific data
}

// NetworkDevice represents a network device status.
type NetworkDevice struct {
	External               Bool                      `json:"external"`
	Present                Bool                      `json:"present"`
	Type                   string                    `json:"type"`
	Up                     Bool                      `json:"up"`
	Carrier                Bool                      `json:"carrier"`
	AuthStatus             Bool                      `json:"auth_status"`
	LinkAdvertising        []string                  `json:"link-advertising"`
	LinkPartnerAdvertising []string                  `json:"link-partner-advertising"`
	LinkSupported          []string                  `json:"link-supported"`
	Speed                  string                    `json:"speed"`
	Autoneg                Bool                      `json:"autoneg"`
	FlowControl            *NetworkDeviceFlowControl `json:"flow-control,omitempty"`
	HwTcOffload            Bool                      `json:"hw-tc-offload"`
	Devtype                string                    `json:"devtype"`
	MTU                    int                       `json:"mtu"`
	MTU6                   int                       `json:"mtu6"`
	MacAddr                string                    `json:"macaddr"`
	TxqueueLen             int                       `json:"txqueuelen"`
	IPv6                   Bool                      `json:"ipv6"`
	IP6SegmentRouting      Bool                      `json:"ip6segmentrouting"`
	Promisc                Bool                      `json:"promisc"`
	RPFilter               int                       `json:"rpfilter"`
	Acceptlocal            Bool                      `json:"acceptlocal"`
	Igmpversion            int                       `json:"igmpversion"`
	Mldversion             int                       `json:"mldversion"`
	Neigh4Reachabletime    int                       `json:"neigh4reachabletime"`
	Neigh6Reachabletime    int                       `json:"neigh6reachabletime"`
	Neigh4Gcstaletime      int                       `json:"neigh4gcstaletime"`
	Neigh6Gcstaletime      int                       `json:"neigh6gcstaletime"`
	Neigh4Locktime         int                       `json:"neigh4locktime"`
	Dadtransmits           int                       `json:"dadtransmits"`
	Multicast              Bool                      `json:"multicast"`
	Sendredirects          Bool                      `json:"sendredirects"`
	DropV4Unicastinfwd     Bool                      `json:"drop_v4_unicast_in_l2_multicast"`
	DropV6Unicastinfwd     Bool                      `json:"drop_v6_unicast_in_l2_multicast"`
	DropGratuitousARP      Bool                      `json:"drop_gratuitous_arp"`
	DropUnsolicitudNA      Bool                      `json:"drop_unsolicited_na"`
	ARPAccept              Bool                      `json:"arp_accept"`
	GRO                    Bool                      `json:"gro"`
	Statistics             NetworkDeviceStatistic    `json:"statistics"`
}

// NetworkDeviceStatistic represents network device statistics.
type NetworkDeviceStatistic struct {
	Collisions        int `json:"collisions"`
	RxFrameErrors     int `json:"rx_frame_errors"`
	TxAbortedErrors   int `json:"tx_aborted_errors"`
	TxCarrierErrors   int `json:"tx_carrier_errors"`
	TxCompressed      int `json:"tx_compressed"`
	TxFifoErrors      int `json:"tx_fifo_errors"`
	TxHeartbeatErrors int `json:"tx_heartbeat_errors"`
	TxWindowErrors    int `json:"tx_window_errors"`
	RxCompressed      int `json:"rx_compressed"`
	RxCrcErrors       int `json:"rx_crc_errors"`
	RxFifoErrors      int `json:"rx_fifo_errors"`
	RxLengthErrors    int `json:"rx_length_errors"`
	RxMissedErrors    int `json:"rx_missed_errors"`
	RxOverErrors      int `json:"rx_over_errors"`
	Multicast         int `json:"multicast"`
	RxBytes           int `json:"rx_bytes"`
	RxDropped         int `json:"rx_dropped"`
	RxErrors          int `json:"rx_errors"`
	RxPackets         int `json:"rx_packets"`
	TxBytes           int `json:"tx_bytes"`
	TxDropped         int `json:"tx_dropped"`
	TxErrors          int `json:"tx_errors"`
	TxPackets         int `json:"tx_packets"`
}

// NetworkDeviceFlowControl represents flow control configuration.
type NetworkDeviceFlowControl struct {
	Autoneg                Bool     `json:"autoneg"`
	Supported              []string `json:"supported"`
	LinkAdvertising        []string `json:"link-advertising"`
	LinkPartnerAdvertising []string `json:"link-partner-advertising"`
	Negotiated             []string `json:"negotiated"`
}

// =============================================================================
// WIRELESS TYPES
// =============================================================================

// RadioStatus represents the status of a wireless radio.
type RadioStatus struct {
	Up               Bool             `json:"up"`
	Pending          Bool             `json:"pending"`
	Autostart        Bool             `json:"autostart"`
	Disabled         Bool             `json:"disabled"`
	RetrySetupFailed Bool             `json:"retry_setup_failed"`
	Config           RadioConfig      `json:"config"`
	Interfaces       []RadioInterface `json:"interfaces"`
}

// RadioConfig represents the radio configuration from UCI.
// This should correspond to wifi-device section configuration.
type RadioConfig struct {
	Type           string `json:"type"`
	Channel        string `json:"channel"`
	Phy            string `json:"phy"`
	MacAddr        string `json:"macaddr"`
	Disabled       Bool   `json:"disabled"`
	Path           string `json:"path"`
	Channels       []int  `json:"channels"`
	Country        string `json:"country"`
	Hwmode         string `json:"hwmode"`
	Band           string `json:"band"`
	HTMode         string `json:"htmode"`
	ChanBW         int    `json:"chanbw"`
	TXPower        int    `json:"txpower"`
	TXAntenna      int    `json:"txantenna"`
	RXAntenna      int    `json:"rxantenna"`
	Antenna        int    `json:"antenna"`
	Diversity      Bool   `json:"diversity"`
	Distance       int    `json:"distance"`
	Frag           int    `json:"frag"`
	RTS            int    `json:"rts"`
	BeaconInt      int    `json:"beacon_int"`
	BasicRate      int    `json:"basic_rate"`
	SupportedRates int    `json:"supported_rates"`
	RequireMode    string `json:"require_mode"`
	LegacyRates    Bool   `json:"legacy_rates"`
	NoScan         Bool   `json:"noscan"`
	LogLevel       int    `json:"log_level"`
	ShortGI        Bool   `json:"short_gi"`
	Greenfield     Bool   `json:"greenfield"`
	TXQueueLen     int    `json:"txqueuelen"`
	DFS            Bool   `json:"dfs"`
	CountryIE      Bool   `json:"country_ie"`
	CellDensity    int    `json:"cell_density"`
}

// RadioInterface represents a wireless interface attached to a radio.
type RadioInterface struct {
	Section  string               `json:"section"`
	Ifname   string               `json:"ifname"`
	Config   RadioInterfaceConfig `json:"config"`
	Vlans    []any                `json:"vlans"`
	Stations []any                `json:"stations"`
}

// RadioInterfaceConfig represents the configuration of a wireless interface.
// This should correspond to wifi-iface section configuration.
type RadioInterfaceConfig struct {
	Mode       *string  `json:"mode,omitempty"`       // Operation mode (ap, sta, adhoc, etc.)
	SSID       *string  `json:"ssid,omitempty"`       // Network SSID
	Encryption *string  `json:"encryption,omitempty"` // Encryption mode
	Network    []string `json:"network,omitempty"`    // Network interface(s)
	Device     *string  `json:"device,omitempty"`     // Radio device
	Disabled   *Bool    `json:"disabled,omitempty"`   // Interface disabled
	Hidden     *Bool    `json:"hidden,omitempty"`     // Hide SSID
	Key        *string  `json:"key,omitempty"`        // Encryption key
	BSSID      *string  `json:"bssid,omitempty"`      // BSSID
}

// =============================================================================
// IWINFO TYPES
// =============================================================================

// WirelessInfo represents detailed wireless interface information.
type WirelessInfo struct {
	Phy         string               `json:"phy"`
	SSID        string               `json:"ssid"`
	BSSID       string               `json:"bssid"`
	Country     string               `json:"country"`
	Mode        string               `json:"mode"`
	Channel     int                  `json:"channel"`
	CenterChan1 int                  `json:"center_chan1"`
	Frequency   int                  `json:"frequency"`
	TXPower     int                  `json:"txpower"`
	Quality     int                  `json:"quality"`
	QualityMax  int                  `json:"quality_max"`
	Signal      int                  `json:"signal"`
	Noise       int                  `json:"noise"`
	Bitrate     int                  `json:"bitrate"`
	Encryption  WirelessEncryption   `json:"encryption"`
	Htmodes     []string             `json:"htmodes"`
	Hwmodes     []string             `json:"hwmodes"`
	HwmodesText string               `json:"hwmodes_text"`
	Hwmode      string               `json:"hwmode"`
	Htmode      string               `json:"htmode"`
	Hardware    WirelessInfoHardware `json:"hardware"`
}

// WirelessEncryption represents wireless encryption information.
type WirelessEncryption struct {
	Enabled        Bool     `json:"enabled"`
	Wpa            []int    `json:"wpa"`
	Authentication []string `json:"authentication"`
	Ciphers        []string `json:"ciphers"`
}

// WirelessInfoHardware represents wireless hardware information.
type WirelessInfoHardware struct {
	ID   []int  `json:"id"`
	Name string `json:"name"`
}

// WirelessScanResult represents a single scan result.
type WirelessScanResult struct {
	SSID       string             `json:"ssid"`
	BSSID      string             `json:"bssid"`
	Mode       string             `json:"mode"`
	Channel    int                `json:"channel"`
	Signal     int                `json:"signal"`
	Quality    int                `json:"quality"`
	QualityMax int                `json:"quality_max"`
	Encryption WirelessEncryption `json:"encryption"`
}

// WirelessAssoc represents an associated wireless station.
type WirelessAssoc struct {
	Mac      string            `json:"mac"`
	Signal   int               `json:"signal"`
	Noise    int               `json:"noise"`
	Inactive int               `json:"inactive"`
	Rx       WirelessAssocRate `json:"rx"`
	Tx       WirelessAssocRate `json:"tx"`
}

// WirelessAssocRate represents wireless association rate information.
type WirelessAssocRate struct {
	Rate    int  `json:"rate"`
	Mcs     int  `json:"mcs"`
	Is40Mhz Bool `json:"40mhz"`
	ShortGi Bool `json:"short_gi"`
}

// WirelessFreq represents a wireless frequency/channel.
type WirelessFreq struct {
	Channel    int  `json:"channel"`
	Mhz        int  `json:"mhz"`
	Restricted Bool `json:"restricted"`
	Active     Bool `json:"active"`
}

// WirelessTxPower represents a wireless TX power level.
type WirelessTxPower struct {
	Dbm    int  `json:"dbm"`
	Mw     int  `json:"mw"`
	Active Bool `json:"active"`
}

// WirelessCountry represents a wireless country code.
type WirelessCountry struct {
	Code    string `json:"code"`
	Country string `json:"country"`
	ISO3166 string `json:"iso3166"`
	Active  Bool   `json:"active"`
}

// WirelessSurvey represents a channel survey result from `iwinfo survey`.
type WirelessSurvey struct {
	Mhz         int `json:"mhz"`
	Noise       int `json:"noise"`
	ActiveTime  int `json:"active_time"`
	BusyTime    int `json:"busy_time"`
	BusyTimeExt int `json:"busy_time_ext"`
	RxTime      int `json:"rx_time"`
	TxTime      int `json:"tx_time"`
}

// =============================================================================
// SERVICE TYPES
// =============================================================================

// ServiceInfo represents all the information about a single service.
type ServiceInfo struct {
	Instances map[string]ServiceInstance `json:"instances"`
}

// ServiceInstance represents a single running or configured instance of a service.
type ServiceInstance struct {
	Running      Bool              `json:"running"`
	Pid          int               `json:"pid"`
	Command      []string          `json:"command"`
	TermTimeout  int               `json:"term_timeout"`
	ExitCode     int               `json:"exit_code,omitempty"`
	Pidfile      string            `json:"pidfile,omitempty"`
	Respawn      *Respawn          `json:"respawn,omitempty"`
	Jail         *Jail             `json:"jail,omitempty"`
	Mount        map[string]string `json:"mount,omitempty"`
	Limits       *Limits           `json:"limits,omitempty"`
	Data         json.RawMessage   `json:"data,omitempty"`
	NoNewPrivs   Bool              `json:"no_new_privs,omitempty"`
	Capabilities string            `json:"capabilities,omitempty"`
	User         string            `json:"user,omitempty"`
	Group        string            `json:"group,omitempty"`
}

// Respawn holds the configuration for service respawning.
type Respawn struct {
	Threshold int `json:"threshold"`
	Timeout   int `json:"timeout"`
	Retry     int `json:"retry"`
}

// Jail holds the configuration for service sandboxing.
type Jail struct {
	Name      string `json:"name"`
	Procfs    Bool   `json:"procfs"`
	Sysfs     Bool   `json:"sysfs"`
	Ubus      Bool   `json:"ubus"`
	Log       Bool   `json:"log"`
	Ronly     Bool   `json:"ronly"`
	Netns     Bool   `json:"netns"`
	Userns    Bool   `json:"userns"`
	Cgroupsns Bool   `json:"cgroupsns"`
	Console   Bool   `json:"console"`
}

// Limits represents resource limits for a service instance.
type Limits struct {
	NoFile string `json:"nofile,omitempty"`
	Core   string `json:"core,omitempty"`
}

// =============================================================================
// RC (INIT SCRIPT) TYPES
// =============================================================================

// RcList represents the response from listing init scripts.
type RcList struct {
	Start   int  `json:"start"`
	Stop    int  `json:"stop,omitempty"`
	Running Bool `json:"running"`
	Enabled Bool `json:"enabled"`
}

// RcInitRequest represents parameters for controlling init scripts.
type RcInitRequest struct {
	Name   string `json:"name"`
	Action string `json:"action"`
}

// =============================================================================
// UCI TYPES
// =============================================================================

// UciGetResponse holds the response from a uci.get call.
type UciGetResponse struct {
	Value  string         `json:"value"`
	Values map[string]any `json:"values"`
}

// UciConfigsResponse holds the response from a uci.configs call.
type UciConfigsResponse struct {
	Configs []string `json:"configs"`
}

// UciStateResponse holds the response from a uci.state call.
type UciStateResponse struct {
	Value  string            `json:"value"`
	Values map[string]string `json:"values"`
}

// UciChangesResponse holds the response from a uci.changes call.
type UciChangesResponse struct {
	Changes map[string]any `json:"changes"`
}

// =============================================================================
// FILE TYPES
// =============================================================================

// FileList represents a directory listing response.
type FileList struct {
	Entries []FileListData `json:"entries"`
}

// FileListData represents a single file or directory entry.
type FileListData struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Size  int    `json:"size"`
	Mode  int    `json:"mode"`
	Atime int64  `json:"atime"`
	Mtime int64  `json:"mtime"`
	Ctime int64  `json:"ctime"`
	Inode int    `json:"inode"`
	UID   int    `json:"uid"`
	GID   int    `json:"gid"`
}

// FileStat represents file statistics.
type FileStat struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Size  int    `json:"size"`
	Mode  int    `json:"mode"`
	Atime int    `json:"atime"`
	Mtime int    `json:"mtime"`
	Ctime int    `json:"ctime"`
	Inode int    `json:"inode"`
	Uid   int    `json:"uid"`
	Gid   int    `json:"gid"`
}

// FileRead represents the response from reading a file.
type FileRead struct {
	Data string `json:"data"`
}

// FileExec represents the response from executing a command.
type FileExec struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

// =============================================================================
// LOG TYPES
// =============================================================================

// Log represents system log entries.
type Log struct {
	Log []LogData `json:"log"`
}

// LogData represents a single log entry.
type LogData struct {
	Text     string `json:"text"`
	ID       int    `json:"id"`
	Priority int    `json:"priority"`
	Source   int    `json:"source"`
	Time     int    `json:"time"`
}

// =============================================================================
// EVENT TYPES
// =============================================================================

// EventHandler is a callback function type for handling ubus events.
// The first argument is the event type (e.g., "network.interface"),
// the second is the event data.
type EventHandler func(eventType string, data map[string]any)

// EventSubscription holds the state for an active event listener.
type EventSubscription struct {
	IsStopped bool
}

// =============================================================================
// SESSION TYPES
// =============================================================================

// SessionData represents the data returned by a session list or create call.
type SessionData struct {
	UbusRPCSession string         `json:"ubus_rpc_session"`
	Timeout        int            `json:"timeout"`
	Expires        int            `json:"expires"`
	ACLs           SessionACLs    `json:"acls"`
	Data           map[string]any `json:"data"`
	ExpireTime     time.Time      `json:"-"` // This field is calculated locally
}

// SessionACLs represents the ACL from user on Authentication.
type SessionACLs struct {
	AccessGroup map[string][]string `json:"access-group"`
	Ubus        map[string][]string
	Uci         map[string][]string
}

// =============================================================================
// LUCI TYPES
// =============================================================================

// LuciVersion represents the version information of LuCI.
type LuciVersion struct {
	Revision string `json:"revision"`
	Branch   string `json:"branch"`
}

// =============================================================================
// LUCI-RPC TYPES
// =============================================================================

// LuciNetworkDevice represents detailed network device information from luci-rpc.
type LuciNetworkDevice struct {
	Name     string                 `json:"name"`
	Wireless Bool                   `json:"wireless"`
	Up       Bool                   `json:"up"`
	MTU      int                    `json:"mtu"`
	QLen     int                    `json:"qlen"`
	DevType  string                 `json:"devtype"`
	IPAddrs  []LuciIPAddress        `json:"ipaddrs"`
	IP6Addrs []LuciIPAddress        `json:"ip6addrs"`
	MAC      string                 `json:"mac,omitempty"`
	Type     int                    `json:"type,omitempty"`
	IfIndex  int                    `json:"ifindex,omitempty"`
	Master   string                 `json:"master,omitempty"`
	Bridge   Bool                   `json:"bridge,omitempty"`
	Ports    []string               `json:"ports,omitempty"`
	ID       string                 `json:"id,omitempty"`
	STP      Bool                   `json:"stp,omitempty"`
	Stats    LuciNetworkDeviceStats `json:"stats"`
	Flags    LuciNetworkDeviceFlags `json:"flags"`
	Link     LuciNetworkDeviceLink  `json:"link"`
}

// LuciIPAddress represents an IP address with netmask and optional broadcast.
type LuciIPAddress struct {
	Address   string `json:"address"`
	Netmask   string `json:"netmask"`
	Broadcast string `json:"broadcast,omitempty"`
}

// LuciNetworkDeviceStats represents network device statistics.
type LuciNetworkDeviceStats struct {
	RxBytes    int `json:"rx_bytes"`
	TxBytes    int `json:"tx_bytes"`
	TxErrors   int `json:"tx_errors"`
	RxErrors   int `json:"rx_errors"`
	TxPackets  int `json:"tx_packets"`
	RxPackets  int `json:"rx_packets"`
	Multicast  int `json:"multicast"`
	Collisions int `json:"collisions"`
	RxDropped  int `json:"rx_dropped"`
	TxDropped  int `json:"tx_dropped"`
}

// LuciNetworkDeviceFlags represents network device flags.
type LuciNetworkDeviceFlags struct {
	Up           Bool `json:"up"`
	Broadcast    Bool `json:"broadcast"`
	Promisc      Bool `json:"promisc"`
	Loopback     Bool `json:"loopback"`
	NoARP        Bool `json:"noarp"`
	Multicast    Bool `json:"multicast"`
	PointToPoint Bool `json:"pointtopoint"`
}

// LuciNetworkDeviceLink represents network device link information.
type LuciNetworkDeviceLink struct {
	Speed     int    `json:"speed,omitempty"`
	Duplex    string `json:"duplex,omitempty"`
	Carrier   Bool   `json:"carrier"`
	Changes   int    `json:"changes"`
	UpCount   int    `json:"up_count"`
	DownCount int    `json:"down_count"`
}

// LuciWirelessDevice represents wireless device information with enhanced details.
type LuciWirelessDevice struct {
	RadioStatus              // Embedded for compatibility
	IWInfo      WirelessInfo `json:"iwinfo"`
}

// LuciHostHint represents host hint information.
type LuciHostHint struct {
	IPAddrs  []string `json:"ipaddrs"`
	IP6Addrs []string `json:"ip6addrs"`
	Name     string   `json:"name,omitempty"`
}

// LuciBoardJSON represents board hardware information.
type LuciBoardJSON struct {
	Model   LuciBoardModel           `json:"model"`
	Network LuciBoardNetwork         `json:"network"`
	WLAN    map[string]LuciBoardWLAN `json:"wlan"`
}

// LuciBoardModel represents board model information.
type LuciBoardModel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LuciBoardNetwork represents board network configuration.
type LuciBoardNetwork struct {
	WAN LuciBoardInterface `json:"wan"`
	LAN LuciBoardInterface `json:"lan"`
}

// LuciBoardInterface represents a board network interface.
type LuciBoardInterface struct {
	Device   string `json:"device"`
	Protocol string `json:"protocol"`
	IPAddr   string `json:"ipaddr,omitempty"`
	MacAddr  string `json:"macaddr"`
}

// LuciBoardWLAN represents board wireless information.
type LuciBoardWLAN struct {
	Path string            `json:"path"`
	Info LuciBoardWLANInfo `json:"info"`
}

// LuciBoardWLANInfo represents detailed wireless capability information.
type LuciBoardWLANInfo struct {
	AntennaRx int                          `json:"antenna_rx"`
	AntennaTx int                          `json:"antenna_tx"`
	Bands     map[string]LuciBoardWLANBand `json:"bands"`
	Radios    []any                        `json:"radios"`
}

// LuciBoardWLANBand represents wireless band capabilities.
type LuciBoardWLANBand struct {
	HT             Bool     `json:"ht,omitempty"`
	VHT            Bool     `json:"vht,omitempty"`
	HE             Bool     `json:"he,omitempty"`
	MaxWidth       int      `json:"max_width"`
	Modes          []string `json:"modes"`
	DefaultChannel int      `json:"default_channel,omitempty"`
}

// =============================================================================
// UCI TYPES
// =============================================================================

// UbusUciRequestGeneric represents the basic UCI request structure.
type UbusUciRequestGeneric struct {
	Config  string `json:"config"`
	Section string `json:"section,omitempty"`
	Option  string `json:"option,omitempty"`
	Type    string `json:"type,omitempty"`
	Match   string `json:"match,omitempty"`
	Name    string `json:"name,omitempty"`
}

// UbusUciRequest represents a UCI request with values.
type UbusUciRequest struct {
	UbusUciRequestGeneric
	Values map[string]string `json:"values,omitempty"`
}

// UbusUciGetRequest represents a UCI get request.
type UbusUciGetRequest struct {
	UbusUciRequestGeneric
}

// UbusUciGetResponse represents a UCI get response.
type UbusUciGetResponse struct {
	Value  string         `json:"value"`
	Values map[string]any `json:"values"`
}

// UbusUciConfigsRequest represents a UCI configs request.
type UbusUciConfigsRequest struct{}

// UbusUciConfigsResponse represents a UCI configs response.
type UbusUciConfigsResponse struct {
	Configs []string `json:"configs"`
}

// UbusUciStateRequest represents a UCI state request.
type UbusUciStateRequest struct {
	UbusUciRequestGeneric
}

// UbusUciStateResponse represents a UCI state response.
type UbusUciStateResponse struct {
	Value  string            `json:"value"`
	Values map[string]string `json:"values"`
}

// UbusUciRenameRequest represents a UCI rename request.
type UbusUciRenameRequest struct {
	Config  string `json:"config"`
	Section string `json:"section,omitempty"`
	Option  string `json:"option,omitempty"`
	Name    string `json:"name"`
}

// UbusUciOrderRequest represents a UCI order request.
type UbusUciOrderRequest struct {
	Config   string   `json:"config"`
	Sections []string `json:"sections"`
}

// UbusUciChangesRequest represents a UCI changes request.
type UbusUciChangesRequest struct {
	Config string `json:"config"`
}

// UbusUciChangesResponse represents a UCI changes response.
type UbusUciChangesResponse struct {
	Changes map[string]any `json:"changes"`
}

// UbusUciRevertRequest represents a UCI revert request.
type UbusUciRevertRequest struct {
	Config string `json:"config"`
}

// UbusUciApplyRequest represents a UCI apply request.
type UbusUciApplyRequest struct {
	Rollback Bool `json:"rollback,omitempty"`
	Timeout  int  `json:"timeout,omitempty"`
}

// UciMetadata holds the read-only metadata associated with a UCI section.
type UciMetadata struct {
	Anonymous Bool   `json:".anonymous"`
	Type      string `json:".type"`
	Name      string `json:".name"`
	// Index represents the position of this section in the UCI configuration.
	// IMPORTANT: This field is only available when querying entire UCI configs
	// (e.g., getting all sections of a package). When querying individual sections,
	// this field will be nil because OpenWrt's UCI system doesn't include the
	// .index field in single-section responses.
	//
	// Usage:
	//   if meta.Index != nil {
	//       fmt.Printf("Section index: %d", *meta.Index)
	//   } else {
	//       fmt.Println("Index not available (single section query)")
	//   }
	Index *int `json:".index"`
}
