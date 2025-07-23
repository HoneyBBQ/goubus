package config

import (
	"github.com/honeybbq/goubus"
)

// WifiDeviceConfig represents the configuration parameters for a wifi-device.
// It implements the ConfigModel interface using UCI tags.
type WifiDeviceConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields (use value types)
	Type    string `uci:"type,enum=mac80211,broadcom,ath9k,ath10k,mt76" default:"autodetected"` // Required=yes: Driver type
	Channel *int   `uci:"channel,range=1-165" default:"auto"`                                   // Required=yes: Wireless channel or "auto"

	// Context-dependent required fields (use pointers for flexibility)
	Phy     *string `uci:"phy,omitempty,case=lower" default:"autodetected"`     // Required=no/yes: Radio phy identifier
	MacAddr *string `uci:"macaddr,omitempty,case=lower" default:"autodetected"` // Required=yes/no: MAC address identifier

	// Optional fields (use pointer types)
	Disabled *bool   `uci:"disabled,omitempty,bool=0/1" default:"false"` // Required=no: Disable radio adapter
	Path     *string `uci:"path,omitempty" default:""`                   // Required=no: Device path

	// Channel and frequency configuration
	Channels *[]int  `uci:"channels,omitempty,join= "`                             // Required=no: Specific channels for auto mode
	Country  *string `uci:"country,omitempty,case=upper" default:"driver default"` // Required=no: Country code

	// Hardware mode and band configuration
	Hwmode *string `uci:"hwmode,omitempty,enum=11b,11g,11a,11n,11ac,11ax" default:"driver default"` // Required=no: Hardware mode (DEPRECATED)
	Band   *string `uci:"band,omitempty,enum=2g,5g,6g,60g" default:"driver default"`                // Required=no: Band

	// High throughput configuration
	HTMode *string `uci:"htmode,omitempty,enum=HT20,HT40,VHT20,VHT40,VHT80,VHT160,HE20,HE40,HE80,HE160" default:"driver default"` // Required=no: HT mode
	ChanBW *int    `uci:"chanbw,omitempty,enum=20,40,80,160" default:"20"`                                                        // Required=no: Channel bandwidth

	// Power and transmission settings
	TXPower   *int  `uci:"txpower,omitempty,unit=dBm,range=0-30" default:"driver default"` // Required=no: Transmission power in dBm
	TXAntenna *int  `uci:"txantenna,omitempty,range=1-8" default:"driver default"`         // Required=no: TX antenna
	RXAntenna *int  `uci:"rxantenna,omitempty,range=1-8" default:"driver default"`         // Required=no: RX antenna
	Antenna   *int  `uci:"antenna,omitempty,range=1-8" default:"driver default"`           // Required=no: Antenna configuration
	Diversity *bool `uci:"diversity,omitempty,bool=0/1" default:"true"`                    // Required=no: Antenna diversity

	// Advanced timing and threshold settings
	Distance       *int    `uci:"distance,omitempty,unit=m,range=0-100000" default:"driver default"` // Required=no: Distance in meters
	Frag           *int    `uci:"frag,omitempty,range=256-2346" default:"driver default"`            // Required=no: Fragmentation threshold
	RTS            *int    `uci:"rts,omitempty,range=0-2347" default:"driver default"`               // Required=no: RTS threshold
	BeaconInt      *int    `uci:"beacon_int,omitempty,unit=ms,range=15-65535" default:"100"`         // Required=no: Beacon interval in ms
	BasicRate      *int    `uci:"basic_rate,omitempty,range=1-54" default:"hostapd default"`         // Required=no: Basic rate in Mbps
	SupportedRates *int    `uci:"supported_rates,omitempty,range=1-300" default:"hostapd default"`   // Required=no: Supported rates in Mbps
	RequireMode    *string `uci:"require_mode,omitempty,enum=none,n,ac,ax" default:"none"`           // Required=no: Required mode
	LegacyRates    *bool   `uci:"legacy_rates,omitempty,bool=0/1" default:"true"`                    // Required=no: Legacy rates

	// Advanced device settings
	NoScan     *bool `uci:"noscan,omitempty,bool=0/1" default:"false"`     // Required=no: Disable scanning
	LogLevel   *int  `uci:"log_level,omitempty,range=0-4" default:"2"`     // Required=no: Logging level
	Short_GI   *bool `uci:"short_gi,omitempty,bool=0/1" default:"false"`   // Required=no: Short guard interval
	Greenfield *bool `uci:"greenfield,omitempty,bool=0/1" default:"false"` // Required=no: Greenfield mode

	// Rate limiting and QoS
	TXQueueLen *int `uci:"txqueuelen,omitempty,range=1-10000" default:"driver default"` // Required=no: TX queue length

	// DFS and regulatory settings
	DFS       *bool `uci:"dfs,omitempty,bool=0/1" default:"false"`       // Required=no: DFS
	CountryIE *bool `uci:"country_ie,omitempty,bool=0/1" default:"auto"` // Required=no: Country IE

	// Cell density and advanced rate configuration
	CellDensity *int `uci:"cell_density,omitempty,range=0-3" default:"0"` // Required=no: Cell density

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// WifiIfaceConfig represents the configuration for a wifi-iface section.
// It implements the ConfigModel interface using UCI tags.
type WifiIfaceConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields (use value types)
	Device  string `uci:"device,case=lower" default:"first device id"`          // Required=yes: Radio device identifier
	Network string `uci:"network,case=lower" default:"lan"`                     // Required=yes: Network interface
	Mode    string `uci:"mode,enum=ap,sta,adhoc,wds,monitor,mesh" default:"ap"` // Required=yes: Operation mode
	SSID    string `uci:"ssid" default:"OpenWrt"`                               // Required=yes: Network SSID

	// Optional fields (use pointer types)
	BSSID    *string `uci:"bssid,omitempty,case=lower" default:"driver default"`  // Required=no: BSSID
	Disabled *bool   `uci:"disabled,omitempty,bool=0/1" default:"false"`          // Required=no: Disable interface
	IfName   *string `uci:"ifname,omitempty,case=lower" default:"driver default"` // Required=no: Interface name

	// Security and encryption configuration
	Encryption *string `uci:"encryption,omitempty,enum=none,wep,wep-open,wep-shared,psk,psk2,psk-mixed,wpa,wpa2,wpa-mixed,wpa3,wpa3-mixed,sae,sae-mixed,owe,owe-mixed" default:"none"` // Required=no: Encryption mode
	Key        *string `uci:"key,omitempty" default:"none"`                                                                                                                            // Required=no: Encryption key
	Hidden     *bool   `uci:"hidden,omitempty,bool=0/1" default:"false"`                                                                                                               // Required=no: Hide SSID
	Isolate    *bool   `uci:"isolate,omitempty,bool=0/1" default:"false"`                                                                                                              // Required=no: Client isolation

	// WEP key configuration
	Key1 *string `uci:"key1,omitempty" default:"none"` // Required=no: WEP key #1
	Key2 *string `uci:"key2,omitempty" default:"none"` // Required=no: WEP key #2
	Key3 *string `uci:"key3,omitempty" default:"none"` // Required=no: WEP key #3
	Key4 *string `uci:"key4,omitempty" default:"none"` // Required=no: WEP key #4

	// Access control
	MacFilter *string   `uci:"macfilter,omitempty,enum=disable,allow,deny" default:"disable"` // Required=no: MAC filtering mode
	MacList   *[]string `uci:"maclist,omitempty,join= ,case=lower" default:"none"`            // Required=no: MAC address list

	// QoS and traffic management
	WMM      *bool   `uci:"wmm,omitempty,bool=0/1" default:"true"`                    // Required=no: WMM support
	MaxAssoc *int    `uci:"maxassoc,omitempty,range=1-255" default:"hostapd default"` // Required=no: Max clients
	MacAddr  *string `uci:"macaddr,omitempty,case=lower" default:"hostapd default"`   // Required=no: Override MAC

	// WPA/WPA2/WPA3 Configuration
	WPAVersion                *int    `uci:"wpa,omitempty,enum=1,2,3" default:"none"`                                    // Required=no: WPA version
	WPACipher                 *string `uci:"wpa_cipher,omitempty,enum=tkip,ccmp,ccmp-256,gcmp,gcmp-256" default:"none"`  // Required=no: WPA cipher
	WPA2Cipher                *string `uci:"wpa2_cipher,omitempty,enum=tkip,ccmp,ccmp-256,gcmp,gcmp-256" default:"none"` // Required=no: WPA2 cipher
	WPAGroupRekey             *int    `uci:"wpa_group_rekey,omitempty,unit=seconds,range=30-86400" default:"600"`        // Required=no: WPA group rekey in seconds
	WPAPairwiseRekey          *int    `uci:"wpa_pairwise_rekey,omitempty,unit=seconds,range=300-86400" default:"none"`   // Required=no: WPA pairwise rekey in seconds
	WPAGMKRekey               *int    `uci:"wpa_gmk_rekey,omitempty,unit=seconds,range=600-86400" default:"none"`        // Required=no: WPA GMK rekey in seconds
	WPAPSKFile                *string `uci:"wpa_psk_file,omitempty" default:"none"`                                      // Required=no: WPA PSK file
	WPADisableEAPOLKeyRetries *bool   `uci:"wpa_disable_eapol_key_retries,omitempty,bool=0/1" default:"false"`           // Required=no: Disable EAPOL retries

	// WPA Enterprise configuration
	AuthServer    *string `uci:"auth_server,omitempty" default:"none"`             // Required=no: Auth server
	AuthPort      *int    `uci:"auth_port,omitempty,range=1-65535" default:"1812"` // Required=no: Auth port
	AuthSecret    *string `uci:"auth_secret,omitempty" default:"none"`             // Required=no: Auth secret
	AcctServer    *string `uci:"acct_server,omitempty" default:"none"`             // Required=no: Accounting server
	AcctPort      *int    `uci:"acct_port,omitempty,range=1-65535" default:"1813"` // Required=no: Accounting port
	AcctSecret    *string `uci:"acct_secret,omitempty" default:"none"`             // Required=no: Accounting secret
	NASIdentifier *string `uci:"nasid,omitempty" default:"none"`                   // Required=no: NAS identifier
	OwnIPAddr     *string `uci:"own_ip_addr,omitempty" default:"none"`             // Required=no: Own IP address
	DynamicVLAN   *bool   `uci:"dynamic_vlan,omitempty,bool=0/1" default:"false"`  // Required=no: Dynamic VLAN

	// WPS (Wi-Fi Protected Setup) Configuration
	WPS           *bool   `uci:"wps_pushbutton,omitempty,bool=0/1" default:"false"`                                                                // Required=no: WPS pushbutton
	WPSLabel      *bool   `uci:"wps_label,omitempty,bool=0/1" default:"false"`                                                                     // Required=no: WPS label
	WPSPin        *string `uci:"wps_pin,omitempty" default:"none"`                                                                                 // Required=no: WPS PIN
	WPSConfig     *string `uci:"wps_config,omitempty,enum=push_button,label,display,ext_nfc_token,int_nfc_token,nfc_interface,pbc" default:"none"` // Required=no: WPS config methods
	WPSDeviceType *string `uci:"wps_device_type,omitempty" default:"6-0050F204-1"`                                                                 // Required=no: WPS device type
	WPSDeviceName *string `uci:"wps_device_name,omitempty" default:"OpenWrt AP"`                                                                   // Required=no: WPS device name
	WPSManufact   *string `uci:"wps_manufacturer,omitempty" default:"openwrt.org"`                                                                 // Required=no: WPS manufacturer

	// 802.11k Neighbor Reports
	IEEE80211k        *bool `uci:"ieee80211k,omitempty,bool=0/1" default:"false"`          // Required=no: Enable 802.11k
	RRMNeighborReport *bool `uci:"rrm_neighbor_report,omitempty,bool=0/1" default:"false"` // Required=no: Neighbor report
	RRMBeaconReport   *bool `uci:"rrm_beacon_report,omitempty,bool=0/1" default:"false"`   // Required=no: Beacon report

	// 802.11v BSS Transition Management
	IEEE80211v    *bool `uci:"ieee80211v,omitempty,bool=0/1" default:"false"`         // Required=no: Enable 802.11v
	BSSTransition *bool `uci:"bss_transition,omitempty,bool=0/1" default:"false"`     // Required=no: BSS transition
	WNMSleepMode  *bool `uci:"wnm_sleep_mode,omitempty,bool=0/1" default:"false"`     // Required=no: WNM sleep mode
	TimeAdvert    *bool `uci:"time_advertisement,omitempty,bool=0/1" default:"false"` // Required=no: Time advertisement

	// 802.11r Fast BSS Transition
	IEEE80211r         *bool   `uci:"ieee80211r,omitempty,bool=0/1" default:"false"`                           // Required=no: Enable 802.11r
	MobilityDomain     *string `uci:"mobility_domain,omitempty" default:"4f57"`                                // Required=no: Mobility domain
	FTOverDS           *bool   `uci:"ft_over_ds,omitempty,bool=0/1" default:"true"`                            // Required=no: FT over DS
	FTPSKGenerateLocal *bool   `uci:"ft_psk_generate_local,omitempty,bool=0/1" default:"false"`                // Required=no: Generate FT PSK locally
	R1KeyHolder        *string `uci:"r1_key_holder,omitempty" default:"00004f577274"`                          // Required=no: R1 key holder
	PMKR1Push          *bool   `uci:"pmk_r1_push,omitempty,bool=0/1" default:"false"`                          // Required=no: PMK-R1 push
	R0KeyLifetime      *int    `uci:"r0_key_lifetime,omitempty,unit=seconds,range=1000-86400" default:"10000"` // Required=no: R0 key lifetime in seconds
	ReassocDeadline    *int    `uci:"reassociation_deadline,omitempty,unit=ms,range=100-5000" default:"1000"`  // Required=no: Reassoc deadline in ms

	// Timing and timeout configuration
	MaxListenInt       *int  `uci:"max_listen_int,omitempty,range=1-65535" default:"65535"`             // Required=no: Max listen interval
	DTIMPeriod         *int  `uci:"dtim_period,omitempty,range=1-255" default:"2"`                      // Required=no: DTIM period
	BeaconInt          *int  `uci:"beacon_int,omitempty,unit=ms,range=15-65535" default:"100"`          // Required=no: Beacon interval in ms
	ListenInterval     *int  `uci:"listen_interval,omitempty,range=1-65535" default:"none"`             // Required=no: Listen interval
	MaxInactivity      *int  `uci:"max_inactivity,omitempty,unit=seconds,range=10-86400" default:"300"` // Required=no: Max inactivity timeout in seconds
	SkipInactivityPoll *bool `uci:"skip_inactivity_poll,omitempty,bool=0/1" default:"false"`            // Required=no: Skip inactivity poll
	DisassocLowAck     *bool `uci:"disassoc_low_ack,omitempty,bool=0/1" default:"true"`                 // Required=no: Disassoc on low ACK

	// Advanced transmission settings
	RtsThreshold  *int  `uci:"rts_threshold,omitempty,range=0-2347" default:"driver default"`    // Required=no: RTS threshold
	FragThreshold *int  `uci:"frag_threshold,omitempty,range=256-2346" default:"driver default"` // Required=no: Frag threshold
	ShortPreamble *bool `uci:"short_preamble,omitempty,bool=0/1" default:"true"`                 // Required=no: Short preamble
	StartDisabled *bool `uci:"start_disabled,omitempty,bool=0/1" default:"false"`                // Required=no: Start disabled

	// Multicast and broadcast settings
	MulticastRate      *int  `uci:"mcast_rate,omitempty,range=1-300" default:"driver default"` // Required=no: Multicast rate in Mbps
	BroadcastSSID      *bool `uci:"broadcast_ssid,omitempty,bool=0/1" default:"true"`          // Required=no: Broadcast SSID
	MulticastToUnicast *bool `uci:"multicast_to_unicast,omitempty,bool=0/1" default:"false"`   // Required=no: Multicast to unicast

	// 802.11w Management Frame Protection
	IEEE80211w             *int `uci:"ieee80211w,omitempty,enum=0,1,2" default:"0"`                                          // Required=no: MFP support (0=disabled, 1=optional, 2=required)
	IEEE80211wMaxTimeout   *int `uci:"ieee80211w_max_timeout,omitempty,unit=seconds,range=1-600" default:"hostapd default"`  // Required=no: MFP max timeout in seconds
	IEEE80211wRetryTimeout *int `uci:"ieee80211w_retry_timeout,omitempty,unit=seconds,range=1-60" default:"hostapd default"` // Required=no: MFP retry timeout in seconds

	// Advanced wireless features
	Doth   *bool   `uci:"doth,omitempty,bool=0/1" default:"false"` // Required=no: 802.11h support
	WDS    *bool   `uci:"wds,omitempty,bool=0/1" default:"false"`  // Required=no: 4-address mode
	MeshID *string `uci:"mesh_id,omitempty" default:"none"`        // Required=no: Mesh ID

	// OWE (Opportunistic Wireless Encryption)
	OWETransitionSSID  *string `uci:"owe_transition_ssid,omitempty" default:"none"`             // Required=no: OWE transition SSID
	OWETransitionBSSID *string `uci:"owe_transition_bssid,omitempty,case=lower" default:"none"` // Required=no: OWE transition BSSID

	// Client mode specific settings
	EAPType    *string `uci:"eap_type,omitempty,enum=tls,ttls,peap,fast" default:"none"`    // Required=no: EAP type
	Auth       *string `uci:"auth,omitempty,enum=MSCHAPV2,PAP,CHAP,MD5" default:"MSCHAPV2"` // Required=no: Auth method
	Identity   *string `uci:"identity,omitempty" default:"none"`                            // Required=no: EAP identity
	Password   *string `uci:"password,omitempty" default:"none"`                            // Required=no: EAP password
	CACert     *string `uci:"ca_cert,omitempty" default:"none"`                             // Required=no: CA certificate
	ClientCert *string `uci:"client_cert,omitempty" default:"none"`                         // Required=no: Client certificate
	PrivKey    *string `uci:"priv_key,omitempty" default:"none"`                            // Required=no: Private key
	PrivKeyPwd *string `uci:"priv_key_pwd,omitempty" default:"none"`                        // Required=no: Private key password

	// Advanced security settings
	TDLSProhibit *bool `uci:"tdls_prohibit,omitempty,bool=0/1" default:"false"` // Required=no: Prohibit TDLS

	// IAPP and roaming
	IAPPInterface *string `uci:"iapp_interface,omitempty,case=lower" default:"none"` // Required=no: IAPP interface
	RSNPreauth    *bool   `uci:"rsn_preauth,omitempty,bool=0/1" default:"false"`     // Required=no: RSN preauthentication

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}
