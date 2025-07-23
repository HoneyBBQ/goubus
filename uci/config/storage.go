package config

import (
	"github.com/honeybbq/goubus"
)

// FstabMountConfig represents mount point configuration in fstab.
// It implements the ConfigModel interface using UCI tags.
type FstabMountConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Device            string            `uci:"device,omitempty"`
	Target            string            `uci:"target,omitempty"`
	FSType            string            `uci:"fstype,omitempty"`
	Options           []string          `uci:"options,omitempty"`
	Enabled           string            `uci:"enabled,omitempty"`
	UUID              string            `uci:"uuid,omitempty"`
	Label             string            `uci:"label,omitempty"`
	EnableAuto        string            `uci:"enable_auto,omitempty"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// FstabSwapConfig represents swap configuration in fstab.
// It implements the ConfigModel interface using UCI tags.
type FstabSwapConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Device            string            `uci:"device,omitempty"`
	Enabled           string            `uci:"enabled,omitempty"`
	Priority          string            `uci:"priority,omitempty"`
	UUID              string            `uci:"uuid,omitempty"`
	Label             string            `uci:"label,omitempty"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// SambaGlobalConfig represents Samba global configuration.
// It implements the ConfigModel interface using UCI tags.
type SambaGlobalConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Name              string            `uci:"name,omitempty"`
	Workgroup         string            `uci:"workgroup,omitempty"`
	Description       string            `uci:"description,omitempty"`
	Charset           string            `uci:"charset,omitempty"`
	Interface         string            `uci:"interface,omitempty"`
	HomesEnabled      string            `uci:"homes,omitempty"`
	NetbiosEnabled    string            `uci:"netbios,omitempty"`
	NullPasswords     string            `uci:"nullpasswords,omitempty"`
	LocalMaster       string            `uci:"local_master,omitempty"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}

// SambaShareConfig represents Samba share configuration.
// It implements the ConfigModel interface using UCI tags.
type SambaShareConfig struct {
	goubus.BaseConfig  `uci:"-"`         // Embed to handle metadata
	Name               string            `uci:"name,omitempty"`
	Path               string            `uci:"path,omitempty"`
	ReadOnly           string            `uci:"read_only,omitempty"`
	GuestOK            string            `uci:"guest_ok,omitempty"`
	GuestOnly          string            `uci:"guest_only,omitempty"`
	InheritOwner       string            `uci:"inherit_owner,omitempty"`
	InheritPermissions string            `uci:"inherit_permissions,omitempty"`
	CreateMask         string            `uci:"create_mask,omitempty"`
	DirMask            string            `uci:"dir_mask,omitempty"`
	Users              string            `uci:"users,omitempty"`
	Extra              map[string]string `uci:",flatten,omitempty"`
}

// SambaUserConfig represents Samba user configuration.
// It implements the ConfigModel interface using UCI tags.
type SambaUserConfig struct {
	goubus.BaseConfig `uci:"-"`         // Embed to handle metadata
	Name              string            `uci:"name,omitempty"`
	Password          string            `uci:"password,omitempty"`
	Extra             map[string]string `uci:",flatten,omitempty"`
}
