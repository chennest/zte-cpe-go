package mf289f

import (
	"fmt"
	"net/url"
	"strconv"
)

type GoformCommand interface {
	GoformID() string
	Authenticated() bool
}

type LoginCommand struct {
	IsTest   bool   `json:"isTest"`
	Password string `json:"password"`
}

func (c *LoginCommand) GoformID() string   { return "LOGIN" }
func (c *LoginCommand) Authenticated() bool { return false }

type LogoutCommand struct{}

func (c *LogoutCommand) GoformID() string   { return "LOGOUT" }
func (c *LogoutCommand) Authenticated() bool { return true }

type RebootCommand struct{}

func (c *RebootCommand) GoformID() string   { return "REBOOT_DEVICE" }
func (c *RebootCommand) Authenticated() bool { return true }

type DisconnectNetworkCommand struct{}

func (c *DisconnectNetworkCommand) GoformID() string   { return "DISCONNECT_NETWORK" }
func (c *DisconnectNetworkCommand) Authenticated() bool { return true }

type ConnectNetworkCommand struct{}

func (c *ConnectNetworkCommand) GoformID() string   { return "CONNECT_NETWORK" }
func (c *ConnectNetworkCommand) Authenticated() bool { return true }

type ConnectionModeCommand struct {
	ConnectionMode    string `json:"ConnectionMode"`
	RoamSettingOption string `json:"roam_setting_option"`
}

func (c *ConnectionModeCommand) GoformID() string   { return "SET_CONNECTION_MODE" }
func (c *ConnectionModeCommand) Authenticated() bool { return true }

type BearerPreferenceCommand struct {
	BearerPreference string `json:"BearerPreference"`
}

func (c *BearerPreferenceCommand) GoformID() string   { return "SET_BEARER_PREFERENCE" }
func (c *BearerPreferenceCommand) Authenticated() bool { return true }

type LockLteBandCommand struct {
	LteBandLock string `json:"lte_band_lock"`
}

func (c *LockLteBandCommand) GoformID() string   { return "SET_LTE_BAND_LOCK" }
func (c *LockLteBandCommand) Authenticated() bool { return true }

type DnsModeCommand struct {
	DnsMode          string `json:"dns_mode"`
	PreferDNSManual  string `json:"prefer_dns_manual"`
	StandbyDNSManual string `json:"standby_dns_manual"`
}

func (c *DnsModeCommand) GoformID() string   { return "ROUTER_DNS_SETTING" }
func (c *DnsModeCommand) Authenticated() bool { return true }

type UpnpCommand struct {
	UpnpSettingOption int `json:"upnp_setting_option"`
}

func (c *UpnpCommand) GoformID() string   { return "UPNP_SETTING" }
func (c *UpnpCommand) Authenticated() bool { return true }

type DmzCommand struct {
	DMZEnabled   int     `json:"DMZEnabled"`
	DMZIPAddress *string `json:"DMZIPAddress,omitempty"`
}

func (c *DmzCommand) GoformID() string   { return "DMZ_SETTING" }
func (c *DmzCommand) Authenticated() bool { return true }

type WiFiCoverageCommand struct {
	WiFiCoverage string `json:"WiFiCoverage"`
}

func (c *WiFiCoverageCommand) GoformID() string   { return "setWiFiCoverage" }
func (c *WiFiCoverageCommand) Authenticated() bool { return true }

type AutoUpgradeCommand struct {
	UpgMode           int `json:"UpgMode"`
	UpgIntervalDay    int `json:"UpgIntervalDay"`
	UpgRoamPermission int `json:"UpgRoamPermission"`
}

func (c *AutoUpgradeCommand) GoformID() string   { return "SetUpgAutoSetting" }
func (c *AutoUpgradeCommand) Authenticated() bool { return true }

func serializeCommand(cmd GoformCommand, ad string) (string, error) {
	values := url.Values{}
	values.Set("isTest", "false")
	values.Set("goformId", cmd.GoformID())
	if ad != "" {
		values.Set("AD", ad)
	}

	switch c := cmd.(type) {
	case *LoginCommand:
		values.Set("password", c.Password)
	case *ConnectionModeCommand:
		values.Set("ConnectionMode", c.ConnectionMode)
		values.Set("roam_setting_option", c.RoamSettingOption)
	case *BearerPreferenceCommand:
		values.Set("BearerPreference", c.BearerPreference)
	case *LockLteBandCommand:
		values.Set("lte_band_lock", c.LteBandLock)
	case *DnsModeCommand:
		values.Set("dns_mode", c.DnsMode)
		values.Set("prefer_dns_manual", c.PreferDNSManual)
		values.Set("standby_dns_manual", c.StandbyDNSManual)
	case *UpnpCommand:
		values.Set("upnp_setting_option", strconv.Itoa(c.UpnpSettingOption))
	case *DmzCommand:
		values.Set("DMZEnabled", strconv.Itoa(c.DMZEnabled))
		if c.DMZIPAddress != nil {
			values.Set("DMZIPAddress", *c.DMZIPAddress)
		}
	case *WiFiCoverageCommand:
		values.Set("WiFiCoverage", c.WiFiCoverage)
	case *AutoUpgradeCommand:
		values.Set("UpgMode", strconv.Itoa(c.UpgMode))
		values.Set("UpgIntervalDay", strconv.Itoa(c.UpgIntervalDay))
		values.Set("UpgRoamPermission", strconv.Itoa(c.UpgRoamPermission))
	default:
		return "", fmt.Errorf("unknown command type: %T", cmd)
	}

	return values.Encode(), nil
}
