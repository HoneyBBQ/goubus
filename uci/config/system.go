package config

import (
	"github.com/honeybbq/goubus"
)

// SystemConfig represents the system configuration.
// It implements the ConfigModel interface using UCI tags.
type SystemConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields (use value types) - 根据OpenWrt文档，系统配置中没有明确标记为required的字段
	// 但hostname是最基础的，通常被视为必需
	Hostname string `uci:"hostname,case=lower" default:"OpenWrt"` // Essential system identifier

	// Optional fields (use pointer types)
	// Basic system information
	Description *string `uci:"description,omitempty" default:"none"` // Required=no: Short description
	Notes       *string `uci:"notes,omitempty" default:"none"`       // Required=no: Multi-line notes

	// Timezone configuration
	Timezone *string `uci:"timezone,omitempty" default:"UTC"` // Required=no: POSIX.1 timezone
	Zonename *string `uci:"zonename,omitempty" default:"UTC"` // Required=no: IANA/Olson timezone

	// Logging configuration with proper types and validation
	BufferSize      *int    `uci:"buffersize,omitempty,unit=kb,range=8-1024" default:"16"`      // Required=no: Kernel message buffer size in KB
	ConLogLevel     *int    `uci:"conloglevel,omitempty,range=1-8" default:"7"`                 // Required=no: Console log level (1-8)
	CronLogLevel    *int    `uci:"cronloglevel,omitempty,range=1-8" default:"5"`                // Required=no: Cron log level
	KLogConLogLevel *int    `uci:"klogconloglevel,omitempty,range=1-8" default:"7"`             // Required=no: Kernel console log level
	LogBufferSize   *int    `uci:"log_buffer_size,omitempty,unit=kb,range=8-1024" default:"16"` // Required=no: Log buffer size in KB
	LogFile         *string `uci:"log_file,omitempty" default:"no log file"`                    // Required=no: Log file path
	LogHostname     *string `uci:"log_hostname,omitempty,case=lower" default:"actual hostname"` // Required=no: Remote syslog hostname
	LogIP           *string `uci:"log_ip,omitempty" default:"none"`                             // Required=no: Remote syslog server IP
	LogPort         *int    `uci:"log_port,omitempty,range=1-65535" default:"514"`              // Required=no: Remote syslog port
	LogPrefix       *string `uci:"log_prefix,omitempty" default:"none"`                         // Required=no: Log message prefix
	LogProto        *string `uci:"log_proto,omitempty,enum=udp,tcp" default:"udp"`              // Required=no: Remote log protocol
	LogRemote       *bool   `uci:"log_remote,omitempty,bool=0/1" default:"true"`                // Required=no: Enable remote logging
	LogSize         *int    `uci:"log_size,omitempty,unit=kb,range=8-1024" default:"64"`        // Required=no: File log buffer size in KB
	LogTrailerNull  *bool   `uci:"log_trailer_null,omitempty,bool=0/1" default:"false"`         // Required=no: Use \0 trailer for TCP
	LogType         *string `uci:"log_type,omitempty,enum=circular,file" default:"circular"`    // Required=no: Log type

	// System security and access with boolean types
	TtyLogin    *bool `uci:"ttylogin,omitempty,bool=0/1" default:"false"`     // Required=no: Require local login auth
	UrandomSeed *bool `uci:"urandom_seed,omitempty,bool=0/1" default:"false"` // Required=no: Enable urandom seed

	// Memory management with proper units
	ZramCompAlgo *string `uci:"zram_comp_algo,omitempty,enum=lzo,lz4,zstd,lzo-rle" default:"lzo"` // Required=no: ZRAM compression
	ZramSizeMB   *int    `uci:"zram_size_mb,omitempty,unit=mb,range=1-1024" default:"auto"`       // Required=no: ZRAM size in MB

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// TimeServerConfig represents NTP time server configuration.
// It implements the ConfigModel interface using UCI tags.
type TimeServerConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Optional fields (use pointer types) - NTP配置都是可选的
	Hostname     *string   `uci:"hostname,omitempty,case=lower" default:"none"`           // Required=no: NTP hostname
	Port         *int      `uci:"port,omitempty,range=1-65535" default:"123"`             // Required=no: NTP port
	Enabled      *bool     `uci:"enabled,omitempty,bool=0/1" default:"true"`              // Required=no: Enable NTP client
	EnableServer *bool     `uci:"enable_server,omitempty,bool=0/1" default:"false"`       // Required=no: Enable NTP server
	Server       *[]string `uci:"server,omitempty,join= " default:"pool.ntp.org servers"` // Required=no: NTP servers
	Pool         *[]string `uci:"pool,omitempty,join= " default:"none"`                   // Required=no: NTP pools

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}

// LEDConfig represents LED configuration.
// It implements the ConfigModel interface using UCI tags.
type LEDConfig struct {
	goubus.BaseConfig `uci:"-"` // Embed to handle metadata

	// Required fields - LED配置中name通常是必需的标识符
	Name string `uci:"name,case=lower" default:"none"` // LED identifier

	// Optional fields (use pointer types)
	Sysfs    *string `uci:"sysfs,omitempty" default:"none"`                                                // Required=no: Sysfs path
	Trigger  *string `uci:"trigger,omitempty,enum=none,default-on,timer,oneshot,heartbeat" default:"none"` // Required=no: LED trigger
	Default  *bool   `uci:"default,omitempty,bool=0/1" default:"false"`                                    // Required=no: Default state
	DelayOn  *int    `uci:"delayon,omitempty,unit=ms,range=0-10000" default:"none"`                        // Required=no: Delay on time in ms
	DelayOff *int    `uci:"delayoff,omitempty,unit=ms,range=0-10000" default:"none"`                       // Required=no: Delay off time in ms
	Interval *int    `uci:"interval,omitempty,unit=ms,range=100-5000" default:"none"`                      // Required=no: Blink interval in ms
	Message  *string `uci:"message,omitempty" default:"none"`                                              // Required=no: Message
	Device   *string `uci:"dev,omitempty,case=lower" default:"none"`                                       // Required=no: Device
	Mode     *string `uci:"mode,omitempty,enum=link,tx,rx" default:"none"`                                 // Required=no: Mode

	// Extra fields for additional options not explicitly defined
	Extra map[string]string `uci:",flatten,omitempty"`
}
