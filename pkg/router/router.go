package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ConnectionMode represents the connection dial mode.
type ConnectionMode string

const (
	ConnectionModeAuto   ConnectionMode = "auto_dial"
	ConnectionModeManual ConnectionMode = "manual_dial"
)

// BearerPreference represents the network bearer preference.
type BearerPreference string

const (
	BearerPreferenceAuto      BearerPreference = "NETWORK_auto"
	BearerPreferenceLteAndNr5g BearerPreference = "4G_AND_5G"
	BearerPreferenceNr5gNsa   BearerPreference = "LTE_AND_5G"
	BearerPreferenceOnlyNr5g  BearerPreference = "Only_5G"
	BearerPreferenceOnlyLte   BearerPreference = "Only_LTE"
	BearerPreferenceOnlyGsm   BearerPreference = "Only_GSM"
	BearerPreferenceOnlyWcdma BearerPreference = "Only_WCDMA"
)

// ApnAuthMode represents the APN authentication mode.
type ApnAuthMode string

const (
	ApnAuthModeNone    ApnAuthMode = "NONE"
	ApnAuthModePap     ApnAuthMode = "PAP"
	ApnAuthModeChap    ApnAuthMode = "CHAP"
	ApnAuthModePapChap ApnAuthMode = "PAP_CHAP"
)

// PdpType represents the PDP type.
type PdpType string

const (
	PdpTypeIPv4   PdpType = "IPv4"
	PdpTypeIPv6   PdpType = "IPv6"
	PdpTypeIPv4v6 PdpType = "IPv4v6"
)

// ApnProfile represents an APN profile configuration.
type ApnProfile struct {
	ProfileID   *string     `json:"profile_id,omitempty"`
	ProfileName string      `json:"profile_name"`
	Apn         string      `json:"apn"`
	PdpType     PdpType     `json:"pdp_type"`
	AuthMode    ApnAuthMode `json:"auth_mode"`
	Username    string      `json:"username"`
	Password    string      `json:"password"`
}

// DhcpSettings represents DHCP configuration.
type DhcpSettings struct {
	IPAddress   string `json:"ip_address"`
	SubnetMask  string `json:"subnet_mask"`
	DhcpEnabled bool   `json:"dhcp_enabled"`
	LeaseTime   uint32 `json:"lease_time"`
}

// MtuSettings represents MTU/MSS configuration.
type MtuSettings struct {
	Mtu uint32 `json:"mtu"`
	Mss uint32 `json:"mss"`
}

// SmsSettings represents SMS configuration.
type SmsSettings struct {
	Validity       string `json:"validity"`
	CenterNumber   string `json:"center_number"`
	DeliveryReport bool   `json:"delivery_report"`
}

// RouterClient is the common interface implemented by all router clients.
type RouterClient interface {
	// Login authenticates with the router.
	Login(ctx context.Context, password string) error

	// Logout logs out from the router.
	Logout(ctx context.Context) error

	// DisconnectNetwork disconnects the mobile/WAN network.
	DisconnectNetwork(ctx context.Context) error

	// ConnectNetwork connects the mobile/WAN network.
	ConnectNetwork(ctx context.Context) error

	// Reboot reboots the router.
	Reboot(ctx context.Context) error

	// GetVersion returns firmware/hardware version as (versionA, versionB).
	GetVersion(ctx context.Context) (string, string, error)

	// SetConnectionMode sets connection mode and roaming.
	SetConnectionMode(ctx context.Context, mode ConnectionMode, roam bool) error

	// SetNetworkBearerPreference sets network bearer preference.
	SetNetworkBearerPreference(ctx context.Context, preference BearerPreference) error

	// SetUpnp enables or disables UPnP.
	SetUpnp(ctx context.Context, enabled bool) error

	// SetDmz sets DMZ host IP, or disables DMZ with nil.
	SetDmz(ctx context.Context, ipAddress *string) error

	// SelectLteBand locks specific LTE bands, or unlocks all with nil.
	SelectLteBand(ctx context.Context, bands []LteBand) error

	// SetDNS sets DNS mode: nil for auto, [primary, secondary] for manual.
	SetDNS(ctx context.Context, manual *[2]string) error

	// GetStatus returns router status info as JSON.
	GetStatus(ctx context.Context) (json.RawMessage, error)

	// GetAPNMode returns the current APN mode: true = manual, false = auto.
	GetAPNMode(ctx context.Context) (bool, error)

	// SetAPNMode sets the APN mode: true = manual, false = auto.
	SetAPNMode(ctx context.Context, manual bool) error

	// GetAPNProfiles returns the list of manual APN profiles.
	GetAPNProfiles(ctx context.Context) ([]ApnProfile, error)

	// SetAPNProfile modifies an existing manual APN profile.
	SetAPNProfile(ctx context.Context, profile *ApnProfile) error

	// EnableAPNProfile sets a manual APN profile as the active/default one.
	EnableAPNProfile(ctx context.Context, profileID string) error

	// GetDHCPSettings returns current DHCP settings.
	GetDHCPSettings(ctx context.Context) (*DhcpSettings, error)

	// SetDHCPSettings sets DHCP settings.
	SetDHCPSettings(ctx context.Context, settings *DhcpSettings) error

	// GetMTUSettings returns current MTU/MSS settings.
	GetMTUSettings(ctx context.Context) (*MtuSettings, error)

	// SetMTUSettings sets MTU/MSS settings.
	SetMTUSettings(ctx context.Context, settings *MtuSettings) error

	// GetSMSSettings returns SMS settings.
	GetSMSSettings(ctx context.Context) (*SmsSettings, error)

	// GetNetworkInfo returns network/signal information.
	GetNetworkInfo(ctx context.Context) (json.RawMessage, error)

	// GetSIMInfo returns SIM card information.
	GetSIMInfo(ctx context.Context) (json.RawMessage, error)

	// GetDeviceInfo returns device information (IMEI, versions, etc).
	GetDeviceInfo(ctx context.Context) (json.RawMessage, error)

	// GetConnectedDevices returns the connected device list.
	GetConnectedDevices(ctx context.Context) (json.RawMessage, error)
}

// ErrNotSupported is returned when an operation is not supported on the model.
type ErrNotSupported struct {
	Method string
}

func (e *ErrNotSupported) Error() string {
	return fmt.Sprintf("%s is not supported on this model", e.Method)
}

// NormalizeRouterURL normalizes a router URL to ensure it has a trailing slash.
func NormalizeRouterURL(rawURL string) (string, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid router URL: %s: %w", rawURL, err)
	}

	if !target.IsAbs() {
		return "", fmt.Errorf("router URL must be an absolute base URL: %s", rawURL)
	}

	if target.RawQuery != "" || target.Fragment != "" {
		return "", fmt.Errorf("router URL must not include a query string or fragment: %s", rawURL)
	}

	path := target.Path
	if !strings.HasSuffix(path, "/") {
		target.Path = strings.TrimRight(path, "/") + "/"
	}

	return target.String(), nil
}
