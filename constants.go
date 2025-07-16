package goubus

// JSON-RPC 相关常量
const (
	// JSON-RPC version
	JSONRPCVersion = "2.0"
	// JSON-RPC method name for ubus calls
	JSONRPCMethodCall = "call"
)

// HTTP 相关常量
const (
	// HTTP Content-Type for JSON
	ContentTypeJSON = "application/json"
	// ubus HTTP endpoint path
	UbusEndpointPath = "/ubus"
)

// ubus 服务名常量
const (
	// 认证服务
	ServiceSession = "session"

	// 系统服务
	ServiceSystem = "system"

	// 网络相关服务
	ServiceNetwork          = "network"
	ServiceNetworkInterface = "network.interface"
	ServiceNetworkDevice    = "network.device"

	// 无线网络服务
	ServiceWireless = "wireless"
	ServiceIwinfo   = "iwinfo"

	// 文件系统服务
	ServiceFile = "file"

	// DHCP服务
	ServiceDHCP = "dhcp"

	// 服务管理
	ServiceRC      = "rc"
	ServiceService = "service"

	// UCI配置服务
	ServiceUCI = "uci"

	// 日志服务
	ServiceLog = "log"

	// 事件服务
	ServiceUbus = "ubus"

	// LUCI RPC服务
	ServiceLuciRPC = "luci-rpc"
)

// ubus 方法名常量
const (
	// 认证方法
	MethodLogin   = "login"
	MethodAccess  = "access"
	MethodDestroy = "destroy"

	// 系统方法
	MethodInfo   = "info"
	MethodBoard  = "board"
	MethodReboot = "reboot"

	// 网络方法
	MethodStatus = "status"
	MethodDump   = "dump"

	// 无线方法
	MethodScan        = "scan"
	MethodDevices     = "devices"
	MethodCountryList = "countrylist"
	MethodTxPowerList = "txpowerlist"
	MethodFreqList    = "freqlist"
	MethodAssocList   = "assoclist"

	// 文件方法
	MethodRead  = "read"
	MethodWrite = "write"
	MethodExec  = "exec"
	MethodStat  = "stat"
	MethodList  = "list"

	// DHCP方法
	MethodGetDHCPLeases = "getDHCPLeases"

	// 服务管理方法
	MethodStart   = "start"
	MethodStop    = "stop"
	MethodRestart = "restart"
	MethodInit    = "init"

	// UCI方法
	MethodGet     = "get"
	MethodSet     = "set"
	MethodAdd     = "add"
	MethodAddList = "add_list"
	MethodDelete  = "delete"
	MethodCommit  = "commit"

	// 日志方法
	// MethodRead, MethodWrite already defined above

	// 事件方法
	MethodSend      = "send"
	MethodSubscribe = "subscribe"
)

// UCI 配置文件名常量
const (
	ConfigNetwork  = "network"
	ConfigWireless = "wireless"
	ConfigDHCP     = "dhcp"
	ConfigSystem   = "system"
	ConfigFirewall = "firewall"
)

// UCI 配置段类型常量
const (
	TypeInterface  = "interface"
	TypeWifiDevice = "wifi-device"
	TypeWifiIface  = "wifi-iface"
	TypeHost       = "host"
	TypeDnsmasq    = "dnsmasq"
)

// 网络配置字段常量
const (
	// 基本网络配置
	FieldProto    = "proto"
	FieldIPAddr   = "ipaddr"
	FieldNetmask  = "netmask"
	FieldGateway  = "gateway"
	FieldDNS      = "dns"
	FieldDevice   = "device"
	FieldType     = "type"
	FieldIfName   = "ifname"
	FieldDisabled = "disabled"
	FieldAuto     = "auto"
	FieldMetric   = "metric"
	FieldMTU      = "mtu"

	// PPPoE 配置
	FieldUsername = "username"
	FieldPassword = "password"
	FieldService  = "service"
)

// 无线配置字段常量
const (
	// 无线设备配置
	FieldChannel = "channel"
	FieldCountry = "country"
	FieldHTMode  = "htmode"
	FieldTXPower = "txpower"
	FieldPath    = "path"
	FieldHwmode  = "hwmode"
	FieldLegacy  = "legacy_rates"

	// 无线接口配置
	FieldSSID       = "ssid"
	FieldEncryption = "encryption"
	FieldKey        = "key"
	FieldNetwork    = "network"
	FieldMode       = "mode"
	FieldHidden     = "hidden"
	FieldIsolate    = "isolate"
	FieldBSSID      = "bssid"
	FieldWPS        = "wps_pushbutton"
	FieldMaxAssoc   = "maxassoc"
)

// DHCP 配置字段常量
const (
	FieldName   = "name"
	FieldMAC    = "mac"
	FieldIP     = "ip"
	FieldHostID = "hostid"
	FieldDUID   = "duid"
)

// 网络协议常量
const (
	ProtoStatic = "static"
	ProtoDHCP   = "dhcp"
	ProtoPPPoE  = "pppoe"
	ProtoNone   = "none"
)

// 无线模式常量
const (
	WirelessModeAP      = "ap"
	WirelessModeStation = "sta"
	WirelessModeAdhoc   = "adhoc"
	WirelessModeMonitor = "monitor"
)

// 无线加密方式常量
const (
	EncryptionNone     = "none"
	EncryptionWEP      = "wep"
	EncryptionPSK      = "psk"
	EncryptionPSK2     = "psk2"
	EncryptionPSKMixed = "psk-mixed"
)

// 服务动作常量
const (
	ActionStart   = "start"
	ActionStop    = "stop"
	ActionRestart = "restart"
	ActionReload  = "reload"
	ActionEnable  = "enable"
	ActionDisable = "disable"
)

// 布尔值字符串常量
const (
	BoolTrue  = "1"
	BoolFalse = "0"
)

// 常用的ubus参数名常量
const (
	ParamName    = "name"
	ParamDevice  = "device"
	ParamMAC     = "mac"
	ParamPath    = "path"
	ParamLines   = "lines"
	ParamStream  = "stream"
	ParamOneshot = "oneshot"
	ParamEvent   = "event"
	ParamType    = "type"
	ParamData    = "data"
	ParamTypes   = "types"
	ParamVerbose = "verbose"
	ParamCommand = "command"
	ParamParams  = "params"
	ParamAppend  = "append"
	ParamMode    = "mode"
	ParamBase64  = "base64"
)

// 特殊会话ID常量
const (
	NullSessionID = "00000000000000000000000000000000"
)

// JSON 字段名常量
const (
	JSONFieldUsername       = "username"
	JSONFieldPassword       = "password"
	JSONFieldUbusRPCSession = "ubus_rpc_session"
	JSONFieldTimeout        = "timeout"
	JSONFieldExpires        = "expires"
	JSONFieldACLs           = "acls"
	JSONFieldAccessGroup    = "access-group"
	JSONFieldUbus           = "ubus"
	JSONFieldUci            = "uci"
	JSONFieldCode           = "code"
	JSONFieldStdout         = "stdout"
)
