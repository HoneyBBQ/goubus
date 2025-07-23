package config

import (
	"github.com/honeybbq/goubus"
)

// UhttpdConfig represents uhttpd configuration.
// It implements the ConfigModel interface using UCI tags.
type UhttpdConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	ListenHTTP        []string          `uci:"listen_http,omitempty,join= "`
	ListenHTTPS       []string          `uci:"listen_https,omitempty,join= "`
	Home              *string           `uci:"home,omitempty" default:"/www"`
	RootDir           *string           `uci:"rfc1918_filter,omitempty" default:"/www"`
	MaxRequests       *int              `uci:"max_requests,omitempty,range=1-1000" default:"3"`
	MaxConnections    *int              `uci:"max_connections,omitempty,range=1-100" default:"100"`
	Cert              *string           `uci:"cert,omitempty" default:"/etc/uhttpd.crt"`
	Key               *string           `uci:"key,omitempty" default:"/etc/uhttpd.key"`
	CGIPrefix         *string           `uci:"cgi_prefix,omitempty" default:"/cgi-bin"`
	LuaPrefix         *string           `uci:"lua_prefix,omitempty" default:"/lua"`
	LuaHandler        *string           `uci:"lua_handler,omitempty" default:"/usr/lib/lua/uhttpd/handler.lua"`
	UbusPrefix        *string           `uci:"ubus_prefix,omitempty" default:"/ubus"`
	UbusSocket        *string           `uci:"ubus_socket,omitempty" default:"/var/run/ubus.sock"`
	ScriptTimeout     *int              `uci:"script_timeout,omitempty,unit=seconds,range=1-300" default:"60"`
	NetworkTimeout    *int              `uci:"network_timeout,omitempty,unit=seconds,range=1-300" default:"30"`
	HTTPKeepAlive     *int              `uci:"http_keepalive,omitempty,unit=seconds,range=0-600" default:"20"`
	TCPKeepAlive      *bool             `uci:"tcp_keepalive,omitempty,bool=0/1" default:"false"`
	UbusNoauth        *bool             `uci:"ubus_noauth,omitempty,bool=0/1" default:"false"`
	UbusCORS          *bool             `uci:"ubus_cors,omitempty,bool=0/1" default:"false"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// DropbearConfig represents Dropbear SSH configuration.
// It implements the ConfigModel interface using UCI tags.
type DropbearConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Enable            *bool             `uci:"enable,omitempty,bool=0/1" default:"true"`
	Verbose           *bool             `uci:"verbose,omitempty,bool=0/1" default:"false"`
	Port              *int              `uci:"Port,omitempty,range=1-65535" default:"22"`
	RootLogin         *bool             `uci:"RootLogin,omitempty,bool=0/1" default:"true"`
	PasswordAuth      *bool             `uci:"PasswordAuth,omitempty,bool=0/1" default:"true"`
	GatewayPorts      *bool             `uci:"GatewayPorts,omitempty,bool=0/1" default:"false"`
	Interface         *string           `uci:"Interface,omitempty,case=lower" default:""`
	RootPasswordAuth  *bool             `uci:"RootPasswordAuth,omitempty,bool=0/1" default:"true"`
	BannerFile        *string           `uci:"BannerFile,omitempty" default:""`
	SSHKeepAlive      *int              `uci:"SSHKeepAlive,omitempty,unit=seconds,range=0-3600" default:"300"`
	IdleTimeout       *int              `uci:"IdleTimeout,omitempty,unit=seconds,range=0-3600" default:"0"`
	MaxAuthTries      *int              `uci:"MaxAuthTries,omitempty,range=1-10" default:"6"`
	RecvWindowSize    *int              `uci:"RecvWindowSize,omitempty,unit=kb,range=1-1024" default:"24"`
	MDNSName          *string           `uci:"mdns,omitempty,case=lower" default:""`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// LuciConfig represents LuCI web interface configuration.
// It implements the ConfigModel interface using UCI tags.
type LuciConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Lang              *string           `uci:"lang,omitempty,case=lower" default:"auto"`
	MediaDir          *string           `uci:"mediaurlbase,omitempty" default:"/luci-static/bootstrap"`
	ResourceBase      *string           `uci:"resourcebase,omitempty" default:"/luci-static/resources"`
	TemplateDir       *string           `uci:"templatedir,omitempty" default:"/usr/lib/lua/luci/view"`
	SessionTimeout    *int              `uci:"sessiontime,omitempty,unit=seconds,range=300-86400" default:"3600"`
	SessionPath       *string           `uci:"sessionpath,omitempty" default:"/tmp/luci-sessions"`
	CcacheDir         *string           `uci:"ccachedir,omitempty" default:"/tmp/luci-modulecache"`
	DiagMode          *bool             `uci:"diag_mode,omitempty,bool=0/1" default:"false"`
	UbusSocket        *string           `uci:"ubuspath,omitempty" default:"/var/run/ubus.sock"`
	Main              *MainConfig       `uci:",flatten,omitempty"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

type MainConfig struct {
	Lang         *string `uci:"lang,omitempty,case=lower" default:"auto"`
	MediaDir     *string `uci:"mediaurlbase,omitempty" default:"/luci-static/bootstrap"`
	ResourceBase *string `uci:"resourcebase,omitempty" default:"/luci-static/resources"`
}

// DDNSServiceConfig represents DDNS service configuration.
// It implements the ConfigModel interface using UCI tags.
type DDNSServiceConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Enabled           *bool             `uci:"enabled,omitempty,bool=0/1" default:"false"`
	Service           *string           `uci:"service_name,omitempty" default:""`
	Domain            *string           `uci:"domain,omitempty,case=lower" default:""`
	Username          *string           `uci:"username,omitempty" default:""`
	Password          *string           `uci:"password,omitempty" default:""`
	Interface         *string           `uci:"interface,omitempty,case=lower" default:""`
	IPSource          *string           `uci:"ip_source,omitempty,enum=network,web,interface,script" default:"network"`
	IPNetwork         *string           `uci:"ip_network,omitempty,case=lower" default:""`
	IPUrl             *string           `uci:"ip_url,omitempty" default:""`
	ForceInterval     *int              `uci:"force_interval,omitempty,unit=seconds,range=3600-604800" default:"72"`
	ForceUnit         *string           `uci:"force_unit,omitempty,enum=seconds,minutes,hours,days" default:"hours"`
	CheckInterval     *int              `uci:"check_interval,omitempty,unit=seconds,range=300-3600" default:"10"`
	CheckUnit         *string           `uci:"check_unit,omitempty,enum=seconds,minutes,hours" default:"minutes"`
	RetryInterval     *int              `uci:"retry_interval,omitempty,unit=seconds,range=60-3600" default:"60"`
	RetryUnit         *string           `uci:"retry_unit,omitempty,enum=seconds,minutes,hours" default:"seconds"`
	RetryCount        *int              `uci:"retry_count,omitempty,range=0-10" default:"5"`
	UseHTTPS          *bool             `uci:"use_https,omitempty,bool=0/1" default:"false"`
	UseSyslog         *bool             `uci:"use_syslog,omitempty,bool=0/1" default:"false"`
	CAPath            *string           `uci:"cacert,omitempty" default:""`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// UpnpdConfig represents UPnP daemon configuration.
// It implements the ConfigModel interface using UCI tags.
type UpnpdConfig struct {
	goubus.BaseConfig   `uci:"-"`         // Embed to handle metadata
	Enabled             *bool             `uci:"enabled,omitempty,bool=0/1" default:"false"`
	EnableNATpmp        *bool             `uci:"enable_natpmp,omitempty,bool=0/1" default:"true"`
	EnableUPnP          *bool             `uci:"enable_upnp,omitempty,bool=0/1" default:"true"`
	SecureMode          *bool             `uci:"secure_mode,omitempty,bool=0/1" default:"true"`
	LogOutput           *bool             `uci:"log_output,omitempty,bool=0/1" default:"false"`
	DownloadSpeed       *int              `uci:"download,omitempty,unit=kbps,range=0-1000000" default:"1024"`
	UploadSpeed         *int              `uci:"upload,omitempty,unit=kbps,range=0-1000000" default:"512"`
	InternalIface       *string           `uci:"internal_iface,omitempty,case=lower" default:"lan"`
	Port                *int              `uci:"port,omitempty,range=1-65535" default:"5000"`
	PresentationURL     *string           `uci:"presentation_url,omitempty" default:""`
	Notify_interval     *int              `uci:"notify_interval,omitempty,unit=seconds,range=30-86400" default:"30"`
	CleanRulesInterval  *int              `uci:"clean_ruleset_interval,omitempty,unit=seconds,range=600-86400" default:"600"`
	CleanRulesThreshold *int              `uci:"clean_ruleset_threshold,omitempty,range=10-1000" default:"20"`
	UUID                *string           `uci:"uuid,omitempty" default:""`
	SerialNumber        *string           `uci:"serial,omitempty" default:""`
	ModelNumber         *string           `uci:"model_number,omitempty" default:""`
	AllowedClients      *[]string         `uci:"upnp_allow,omitempty,join= " default:"0.0.0.0/0 1024-65535"`
	DeniedClients       *[]string         `uci:"upnp_deny,omitempty,join= " default:""`
	Extra               map[string]string `uci:",flatten,omitempty"`
}
