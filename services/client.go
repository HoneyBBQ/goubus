package services

import (
	"github.com/honeybbq/goubus/v2"
	"github.com/honeybbq/goubus/v2/services/container"
	"github.com/honeybbq/goubus/v2/services/dhcp"
	"github.com/honeybbq/goubus/v2/services/file"
	"github.com/honeybbq/goubus/v2/services/hostapd"
	ubuslog "github.com/honeybbq/goubus/v2/services/log"
	"github.com/honeybbq/goubus/v2/services/luci"
	"github.com/honeybbq/goubus/v2/services/network"
	"github.com/honeybbq/goubus/v2/services/rc"
	"github.com/honeybbq/goubus/v2/services/rpcsys"
	"github.com/honeybbq/goubus/v2/services/service"
	"github.com/honeybbq/goubus/v2/services/session"
	"github.com/honeybbq/goubus/v2/services/system"
	"github.com/honeybbq/goubus/v2/services/uci"
	"github.com/honeybbq/goubus/v2/services/wireless"
	"github.com/honeybbq/goubus/v2/services/wpasupplicant"
)

// Option configures selected clients in a services collection.
type Option interface {
	apply(config *options)
}

type options struct {
	dhcp []dhcp.Option
	luci []luci.Option
}

type clientOption func(*options)

func (option clientOption) apply(config *options) { option(config) }

// WithDHCP passes options to the DHCP client.
func WithDHCP(opts ...dhcp.Option) Option {
	return clientOption(func(config *options) {
		config.dhcp = append(config.dhcp, opts...)
	})
}

// WithLuCI passes options to the LuCI client.
func WithLuCI(opts ...luci.Option) Option {
	return clientOption(func(config *options) {
		config.luci = append(config.luci, opts...)
	})
}

// Client holds one instance of every service client for a shared caller.
type Client struct {
	container     *container.Client
	dhcp          *dhcp.Client
	file          *file.Client
	hostapd       *hostapd.Client
	log           *ubuslog.Client
	luci          *luci.Client
	network       *network.Client
	rc            *rc.Client
	rpcsys        *rpcsys.Client
	service       *service.Client
	session       *session.Client
	system        *system.Client
	uci           *uci.Client
	wireless      *wireless.Client
	wpasupplicant *wpasupplicant.Client
}

// New constructs all service clients for caller. It panics if caller is nil.
func New(caller goubus.Caller, opts ...Option) *Client {
	if caller == nil {
		panic("goubus/services: nil caller")
	}
	config := options{}
	for _, option := range opts {
		if option != nil {
			option.apply(&config)
		}
	}

	return &Client{
		container:     container.New(caller),
		dhcp:          dhcp.New(caller, config.dhcp...),
		file:          file.New(caller),
		hostapd:       hostapd.New(caller),
		log:           ubuslog.New(caller),
		luci:          luci.New(caller, config.luci...),
		network:       network.New(caller),
		rc:            rc.New(caller),
		rpcsys:        rpcsys.New(caller),
		service:       service.New(caller),
		session:       session.New(caller),
		system:        system.New(caller),
		uci:           uci.New(caller),
		wireless:      wireless.New(caller),
		wpasupplicant: wpasupplicant.New(caller),
	}
}

// Container returns the container client.
func (client *Client) Container() *container.Client { return client.container }

// DHCP returns the DHCP client.
func (client *Client) DHCP() *dhcp.Client { return client.dhcp }

// File returns the file client.
func (client *Client) File() *file.Client { return client.file }

// Hostapd returns the hostapd client.
func (client *Client) Hostapd() *hostapd.Client { return client.hostapd }

// Log returns the log client.
func (client *Client) Log() *ubuslog.Client { return client.log }

// LuCI returns the LuCI client.
func (client *Client) LuCI() *luci.Client { return client.luci }

// Network returns the network client.
func (client *Client) Network() *network.Client { return client.network }

// RC returns the rc client.
func (client *Client) RC() *rc.Client { return client.rc }

// RPCSys returns the rpc.sys client.
func (client *Client) RPCSys() *rpcsys.Client { return client.rpcsys }

// Service returns the service client.
func (client *Client) Service() *service.Client { return client.service }

// Session returns the session client.
func (client *Client) Session() *session.Client { return client.session }

// System returns the system client.
func (client *Client) System() *system.Client { return client.system }

// UCI returns the UCI client.
func (client *Client) UCI() *uci.Client { return client.uci }

// Wireless returns the wireless client.
func (client *Client) Wireless() *wireless.Client { return client.wireless }

// WPASupplicant returns the wpa_supplicant client.
func (client *Client) WPASupplicant() *wpasupplicant.Client { return client.wpasupplicant }
