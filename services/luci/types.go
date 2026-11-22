package luci

import (
	"encoding/json"

	"github.com/honeybbq/goubus/v2"
)

// Version contains LuCI version metadata.
type Version struct {
	// Revision is the LuCI revision identifier.
	Revision string `json:"revision"`
	// Branch is the LuCI source branch.
	Branch string `json:"branch"`
}

// Feature represents a system feature and its optional subfeatures.
type Feature struct {
	// Subfeatures contains individual feature flags.
	Subfeatures map[string]goubus.Bool `json:"-"`
	// Enabled reports the feature's overall availability.
	Enabled goubus.Bool `json:"-"`
}

// UnmarshalJSON accepts feature flags represented by a scalar or an object.
func (feature *Feature) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '{' {
		var subfeatures map[string]goubus.Bool
		err := json.Unmarshal(data, &subfeatures)
		if err != nil {
			return err
		}
		feature.Subfeatures = subfeatures
		feature.Enabled = true

		return nil
	}
	feature.Subfeatures = nil

	return json.Unmarshal(data, &feature.Enabled)
}

// LED describes a system LED and its supported triggers.
type LED struct {
	// Name is the system LED name.
	Name string `json:"name,omitempty"`
	// ActiveTrigger is the currently selected LED trigger.
	ActiveTrigger string `json:"active_trigger"`
	// Triggers lists triggers supported by the LED.
	Triggers []string `json:"triggers"`
	// Brightness is the current LED brightness.
	Brightness int `json:"brightness"`
	// MaxBrightness is the maximum LED brightness.
	MaxBrightness int `json:"max_brightness"`
}

// USBInfo contains connected USB devices and ports.
type USBInfo struct {
	// Devices lists detected USB devices.
	Devices []USBDevice `json:"devices"`
	// Ports lists available USB ports.
	Ports []USBPort `json:"ports"`
}

// USBDevice describes a detected USB device.
type USBDevice struct {
	// Bus is the USB bus number.
	Bus string `json:"bus"`
	// Device is the device number.
	Device string `json:"device"`
	// Vendor is the vendor identifier.
	Vendor string `json:"vendor"`
	// Product is the product identifier.
	Product string `json:"product"`
	// Class is the USB device class.
	Class string `json:"class"`
}

// USBPort describes a USB port.
type USBPort struct {
	// Device contains the device attached to this port, if any.
	Device any `json:"device"`
	// Port is the USB port name.
	Port string `json:"port"`
}

// BlockDevice describes a block device.
type BlockDevice struct {
	// Name is the device name.
	Name string `json:"name"`
	// Dev is the device path.
	Dev string `json:"dev"`
	// Model is the hardware model.
	Model string `json:"model,omitempty"`
	// Vendor is the hardware vendor.
	Vendor string `json:"vendor,omitempty"`
	// Size is the device size in bytes.
	Size uint64 `json:"size"`
	// Major is the device major number.
	Major int `json:"major"`
	// Minor is the device minor number.
	Minor int `json:"minor"`
}

// MountPoint describes a mounted filesystem.
type MountPoint struct {
	// Device is the mounted source device.
	Device string `json:"device"`
	// Mount is the mount path.
	Mount string `json:"mount"`
	// Size is the filesystem size in bytes.
	Size uint64 `json:"size"`
	// Avail is the available filesystem space in bytes.
	Avail uint64 `json:"avail"`
	// Free is the free filesystem space in bytes.
	Free uint64 `json:"free"`
}

// Process describes a running system process.
type Process struct {
	// User is the process owner.
	User string `json:"USER"`
	// Stat is the process state.
	Stat string `json:"STAT"`
	// VSZ is the virtual memory size.
	VSZ string `json:"VSZ"`
	// CPU is the CPU utilization percentage.
	CPU string `json:"%CPU"`
	// Mem is the memory utilization percentage.
	Mem string `json:"%MEM"`
	// Command is the process command line.
	Command string `json:"COMMAND"`
	// PID is the process identifier.
	PID goubus.StringInt `json:"PID"`
	// PPID is the parent process identifier.
	PPID goubus.StringInt `json:"PPID"`
}

// RealtimeStats contains interface traffic counters.
type RealtimeStats struct {
	// RxBytes is the received byte count.
	RxBytes uint64 `json:"rx_bytes"`
	// TxBytes is the transmitted byte count.
	TxBytes uint64 `json:"tx_bytes"`
	// RxPackets is the received packet count.
	RxPackets uint64 `json:"rx_packets"`
	// TxPackets is the transmitted packet count.
	TxPackets uint64 `json:"tx_packets"`
}

// NetworkDevice contains a detailed network device status.
type NetworkDevice struct {
	// MAC is the device hardware address.
	MAC string `json:"mac,omitempty"`
	// DevType is the device type string.
	DevType string `json:"devtype"`
	// Name is the device name.
	Name string `json:"name"`
	// ID is the device identifier.
	ID string `json:"id,omitempty"`
	// Master is the master device name.
	Master string `json:"master,omitempty"`
	// IPAddrs contains IPv4 address assignments.
	IPAddrs []IPAddress `json:"ipaddrs"`
	// Ports lists member ports.
	Ports []string `json:"ports,omitempty"`
	// IP6Addrs contains IPv6 address assignments.
	IP6Addrs []IPAddress `json:"ip6addrs"`
	// Link contains link status details.
	Link NetworkDeviceLink `json:"link"`
	// Stats contains device traffic counters.
	Stats NetworkDeviceStats `json:"stats"`
	// Flags contains device flags.
	Flags NetworkDeviceFlags `json:"flags"`
	// Type is the device numeric type.
	Type int `json:"type,omitempty"`
	// IfIndex is the interface index.
	IfIndex int `json:"ifindex,omitempty"`
	// QLen is the transmit queue length.
	QLen int `json:"qlen"`
	// MTU is the maximum transmission unit.
	MTU int `json:"mtu"`
	// Bridge reports whether the device is a bridge.
	Bridge goubus.Bool `json:"bridge,omitempty"`
	// STP reports whether spanning tree is enabled.
	STP goubus.Bool `json:"stp,omitempty"`
	// Up reports whether the device is up.
	Up goubus.Bool `json:"up"`
	// Wireless reports whether the device is wireless.
	Wireless goubus.Bool `json:"wireless"`
}

// IPAddress describes an IP address and its netmask.
type IPAddress struct {
	// Address is the IP address.
	Address string `json:"address"`
	// Netmask is the address netmask.
	Netmask string `json:"netmask"`
	// Broadcast is the optional broadcast address.
	Broadcast string `json:"broadcast,omitempty"`
}

// NetworkDeviceStats contains network traffic and packet counters.
type NetworkDeviceStats struct {
	// RxBytes is the received byte count.
	RxBytes uint64 `json:"rx_bytes"`
	// TxBytes is the transmitted byte count.
	TxBytes uint64 `json:"tx_bytes"`
	// TxErrors is the transmit error count.
	TxErrors uint64 `json:"tx_errors"`
	// RxErrors is the receive error count.
	RxErrors uint64 `json:"rx_errors"`
	// TxPackets is the transmitted packet count.
	TxPackets uint64 `json:"tx_packets"`
	// RxPackets is the received packet count.
	RxPackets uint64 `json:"rx_packets"`
	// Multicast is the multicast packet count.
	Multicast uint64 `json:"multicast"`
	// Collisions is the collision count.
	Collisions uint64 `json:"collisions"`
	// RxDropped is the dropped receive packet count.
	RxDropped uint64 `json:"rx_dropped"`
	// TxDropped is the dropped transmit packet count.
	TxDropped uint64 `json:"tx_dropped"`
}

// NetworkDeviceFlags contains network device flags.
type NetworkDeviceFlags struct {
	// Up reports whether the device is up.
	Up goubus.Bool `json:"up"`
	// Broadcast reports whether broadcast is supported.
	Broadcast goubus.Bool `json:"broadcast"`
	// Promisc reports whether promiscuous mode is enabled.
	Promisc goubus.Bool `json:"promisc"`
	// Loopback reports whether the device is loopback.
	Loopback goubus.Bool `json:"loopback"`
	// NoARP reports whether ARP is disabled.
	NoARP goubus.Bool `json:"noarp"`
	// Multicast reports whether multicast is supported.
	Multicast goubus.Bool `json:"multicast"`
	// PointToPoint reports whether the device is point-to-point.
	PointToPoint goubus.Bool `json:"pointtopoint"`
}

// NetworkDeviceLink contains link state and speed information.
type NetworkDeviceLink struct {
	// Duplex is the link duplex mode.
	Duplex string `json:"duplex,omitempty"`
	// Speed is the link speed in megabits per second.
	Speed int `json:"speed,omitempty"`
	// Changes is the link change count.
	Changes int `json:"changes"`
	// UpCount is the number of link-up events.
	UpCount int `json:"up_count"`
	// DownCount is the number of link-down events.
	DownCount int `json:"down_count"`
	// Carrier reports whether the link has carrier.
	Carrier goubus.Bool `json:"carrier"`
}

// HostHint describes a host discovered on the network.
type HostHint struct {
	// Name is the host name.
	Name string `json:"name,omitempty"`
	// IPAddrs contains IPv4 addresses for the host.
	IPAddrs []string `json:"ipaddrs"`
	// IP6Addrs contains IPv6 addresses for the host.
	IP6Addrs []string `json:"ip6addrs"`
}

// BoardJSON contains device board and network information.
type BoardJSON struct {
	// WLAN contains wireless hardware and capability information.
	WLAN map[string]BoardWLAN `json:"wlan"`
	// Model contains the board model identifiers.
	Model BoardModel `json:"model"`
	// Network contains the board's network interfaces.
	Network BoardNetwork `json:"network"`
}

// BoardModel contains board model identifiers.
type BoardModel struct {
	// ID is the board model identifier.
	ID string `json:"id"`
	// Name is the board model name.
	Name string `json:"name"`
}

// BoardNetwork contains LAN and WAN board interfaces.
type BoardNetwork struct {
	// WAN describes the WAN interface.
	WAN BoardInterface `json:"wan"`
	// LAN describes the LAN interface.
	LAN BoardInterface `json:"lan"`
}

// BoardInterface describes a board network interface.
type BoardInterface struct {
	// Device is the associated device name.
	Device string `json:"device"`
	// Protocol is the configured network protocol.
	Protocol string `json:"protocol"`
	// IPAddr is the configured IPv4 address.
	IPAddr string `json:"ipaddr,omitempty"`
	// MACAddr is the interface hardware address.
	MACAddr string `json:"macaddr"`
	// Ports lists switch ports assigned to the interface.
	Ports []string `json:"ports,omitempty"`
}

// BoardWLAN contains board wireless device information.
type BoardWLAN struct {
	// Path is the platform path for the radio.
	Path string `json:"path"`
	// Info contains radio capabilities.
	Info BoardWLANInfo `json:"info"`
}

// BoardWLANInfo contains radio hardware capabilities.
type BoardWLANInfo struct {
	// Bands maps band names to their capabilities.
	Bands map[string]BoardWLANBand `json:"bands"`
	// Radios contains radio identifiers.
	Radios []any `json:"radios"`
	// AntennaRx is the receive antenna count.
	AntennaRx int `json:"antenna_rx"`
	// AntennaTx is the transmit antenna count.
	AntennaTx int `json:"antenna_tx"`
}

// BoardWLANBand contains the modes available on a wireless band.
type BoardWLANBand struct {
	// Modes lists supported channel modes.
	Modes []string `json:"modes"`
	// MaxWidth is the maximum channel width.
	MaxWidth int `json:"max_width"`
	// DefaultChannel is the default channel.
	DefaultChannel int `json:"default_channel,omitempty"`
	// HT reports support for 802.11n.
	HT goubus.Bool `json:"ht,omitempty"`
	// VHT reports support for 802.11ac.
	VHT goubus.Bool `json:"vht,omitempty"`
	// HE reports support for 802.11ax.
	HE goubus.Bool `json:"he,omitempty"`
}

// DHCPLeases contains LuCI's DHCPv4 and DHCPv6 lease lists.
type DHCPLeases struct {
	// DHCPLeases lists DHCPv4 leases.
	DHCPLeases []DHCPv4Lease `json:"dhcp_leases"`
	// DHCP6Leases lists DHCPv6 leases.
	DHCP6Leases []DHCPv6Lease `json:"dhcp6_leases"`
}

// DHCPv4Lease describes a DHCPv4 client lease.
type DHCPv4Lease struct {
	// Hostname is the client host name.
	Hostname string `json:"hostname"`
	// IPAddr is the leased IPv4 address.
	IPAddr string `json:"ipaddr"`
	// MACAddr is the client hardware address.
	MACAddr string `json:"macaddr"`
	// Expires is the remaining lease time.
	Expires uint64 `json:"expires"`
}

// DHCPv6Lease describes a DHCPv6 client lease.
type DHCPv6Lease struct {
	// Hostname is the client host name.
	Hostname string `json:"hostname"`
	// DUID is the client DHCP unique identifier.
	DUID string `json:"duid"`
	// IAID is the client identity association identifier.
	IAID string `json:"iaid"`
	// IPAddr contains leased IPv6 addresses.
	IPAddr []string `json:"ip6addr"`
	// Expires is the remaining lease time.
	Expires uint64 `json:"expires"`
}

// WirelessDevice describes a wireless radio and its interfaces.
type WirelessDevice struct {
	// Config contains the wireless device configuration.
	Config map[string]any `json:"config"`
	// IWInfo contains wireless hardware information.
	IWInfo *WirelessIWInfo `json:"iwinfo,omitempty"`
	// Interfaces lists interfaces configured on the device.
	Interfaces []WirelessInterface `json:"interfaces"`
	// Up reports whether the device is up.
	Up goubus.Bool `json:"up"`
	// Pending reports whether a device operation is pending.
	Pending goubus.Bool `json:"pending"`
	// Autostart reports whether the device starts automatically.
	Autostart goubus.Bool `json:"autostart"`
	// Disabled reports whether the device is disabled.
	Disabled goubus.Bool `json:"disabled"`
	// RetrySetupFailed reports whether device setup retries failed.
	RetrySetupFailed goubus.Bool `json:"retry_setup_failed"`
}

// WirelessInterface describes a configured wireless interface.
type WirelessInterface struct {
	// Config contains the wireless interface configuration.
	Config map[string]any `json:"config"`
	// IWInfo contains wireless interface information.
	IWInfo *WirelessIWInfo `json:"iwinfo,omitempty"`
	// Section is the UCI section name.
	Section string `json:"section"`
	// IfName is the operating system interface name.
	IfName string `json:"ifname"`
	// VLANs lists VLANs assigned to the interface.
	VLANs []any `json:"vlans"`
	// Stations lists stations associated with the interface.
	Stations []any `json:"stations"`
}

// WirelessIWInfo contains wireless radio or interface capabilities.
type WirelessIWInfo struct {
	// Encryption contains wireless encryption details.
	Encryption *WirelessEncryption `json:"encryption,omitempty"`
	// Hardware contains the radio hardware identity.
	Hardware *WirelessHardware `json:"hardware"`
	// HWModesText contains the human-readable hardware modes.
	HWModesText string `json:"hwmodes_text"`
	// Country is the configured wireless country code.
	Country string `json:"country"`
	// PHY is the wireless physical device name.
	PHY string `json:"phy"`
	// BSSID is the basic service set identifier.
	BSSID string `json:"bssid"`
	// Mode is the wireless operating mode.
	Mode string `json:"mode"`
	// HTModes lists supported 802.11n modes.
	HTModes []string `json:"htmodes"`
	// HWModes lists supported hardware modes.
	HWModes []string `json:"hwmodes"`
	// TxPower is the transmit power in dBm.
	TxPower int `json:"txpower"`
	// FrequencyOffset is the frequency offset in kHz.
	FrequencyOffset int `json:"frequency_offset"`
	// Frequency is the operating frequency in MHz.
	Frequency int `json:"frequency"`
	// QualityMax is the maximum signal quality.
	QualityMax int `json:"quality_max"`
	// TxPowerOffset is the transmit power offset in dB.
	TxPowerOffset int `json:"txpower_offset"`
	// Noise is the measured noise level in dBm.
	Noise int `json:"noise"`
	// Channel is the wireless channel number.
	Channel int `json:"channel"`
}

// WirelessHardware identifies wireless hardware.
type WirelessHardware struct {
	// Name is the wireless hardware name.
	Name string `json:"name"`
	// ID contains the hardware identifier values.
	ID []int `json:"id"`
}

// WirelessEncryption describes wireless encryption capabilities.
type WirelessEncryption struct {
	// WPA lists supported WPA protocol versions.
	WPA []int `json:"wpa"`
	// Authentication lists supported authentication methods.
	Authentication []string `json:"authentication"`
	// Ciphers lists supported encryption ciphers.
	Ciphers []string `json:"ciphers"`
	// Enabled reports whether encryption is enabled.
	Enabled goubus.Bool `json:"enabled"`
}
