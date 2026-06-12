package mf289f

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"

	"github.com/1zun4/zte-cpe-go/pkg/router"
)

type Mf289fClient struct {
	target string
	client *http.Client
}

func New(url string) (*Mf289fClient, error) {
	normalized, err := router.NormalizeRouterURL(url)
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{Jar: jar}

	return &Mf289fClient{
		target: normalized,
		client: client,
	}, nil
}

func (c *Mf289fClient) getLD(ctx context.Context) (string, error) {
	resp, err := c.getCommand(ctx, "LD")
	if err != nil {
		return "", fmt.Errorf("failed to fetch LD: %w", err)
	}

	ld, ok := resp["LD"].(string)
	if !ok {
		return "", fmt.Errorf("missing LD in response")
	}

	return ld, nil
}

func (c *Mf289fClient) getRD(ctx context.Context) (string, error) {
	resp, err := c.getCommand(ctx, "RD")
	if err != nil {
		return "", fmt.Errorf("failed to fetch RD: %w", err)
	}

	rd, ok := resp["RD"].(string)
	if !ok {
		return "", fmt.Errorf("missing RD in response")
	}

	return rd, nil
}

func (c *Mf289fClient) getAD(ctx context.Context) (string, error) {
	crVersion, waInnerVersion, err := c.GetVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch version: %w", err)
	}

	a := md5Hash(waInnerVersion + crVersion)
	u, err := c.getRD(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch RD: %w", err)
	}

	return md5Hash(a + u), nil
}

func (c *Mf289fClient) sendCommand(ctx context.Context, cmd GoformCommand) (string, error) {
	ad := ""
	if cmd.Authenticated() {
		adValue, err := c.getAD(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to fetch AD: %w", err)
		}
		ad = adValue
	}

	formData, err := serializeCommand(cmd, ad)
	if err != nil {
		return "", fmt.Errorf("failed to serialize command: %w", err)
	}

	url := fmt.Sprintf("%sgoform/goform_set_cmd_process", c.target)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(formData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", c.target)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send %s command: %w", cmd.GoformID(), err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse JSON for %s command: %w", cmd.GoformID(), err)
	}

	resultStr, ok := result["result"].(string)
	if !ok {
		return "", fmt.Errorf("missing result in response")
	}

	return resultStr, nil
}

func (c *Mf289fClient) getCommand(ctx context.Context, cmd string) (map[string]interface{}, error) {
	multiData := strings.Contains(cmd, ",")
	url := fmt.Sprintf("%sgoform/goform_get_cmd_process?isTest=false&cmd=%s", c.target, cmd)
	if multiData {
		url += "&multi_data=1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Referer", c.target)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch command %s: %w", cmd, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON for command %s: %w", cmd, err)
	}

	return result, nil
}

func (c *Mf289fClient) Login(ctx context.Context, password string) error {
	ld, err := c.getLD(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch LD: %w", err)
	}

	hashPassword := strings.ToUpper(sha256Hash(password))
	ztePass := strings.ToUpper(sha256Hash(hashPassword + ld))

	result, err := c.sendCommand(ctx, &LoginCommand{
		IsTest:   false,
		Password: ztePass,
	})
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	switch result {
	case "0":
		return nil
	case "3":
		return fmt.Errorf("invalid password")
	default:
		return fmt.Errorf("unknown error code: %s", result)
	}
}

func (c *Mf289fClient) Logout(ctx context.Context) error {
	_, err := c.sendCommand(ctx, &LogoutCommand{})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	return nil
}

func (c *Mf289fClient) GetVersion(ctx context.Context) (string, string, error) {
	resp, err := c.getCommand(ctx, "cr_version,wa_inner_version")
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch version: %w", err)
	}

	crVersion, ok := resp["cr_version"].(string)
	if !ok {
		return "", "", fmt.Errorf("missing cr_version in response")
	}

	waInnerVersion, ok := resp["wa_inner_version"].(string)
	if !ok {
		return "", "", fmt.Errorf("missing wa_inner_version in response")
	}

	return crVersion, waInnerVersion, nil
}

func (c *Mf289fClient) Reboot(ctx context.Context) error {
	result, err := c.sendCommand(ctx, &RebootCommand{})
	if err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to reboot: %s", result)
	}
	return nil
}

func (c *Mf289fClient) DisconnectNetwork(ctx context.Context) error {
	result, err := c.sendCommand(ctx, &DisconnectNetworkCommand{})
	if err != nil {
		return fmt.Errorf("failed to disconnect network: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to disconnect network: %s", result)
	}
	return nil
}

func (c *Mf289fClient) ConnectNetwork(ctx context.Context) error {
	result, err := c.sendCommand(ctx, &ConnectNetworkCommand{})
	if err != nil {
		return fmt.Errorf("failed to connect network: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to connect network: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SetConnectionMode(ctx context.Context, mode router.ConnectionMode, roam bool) error {
	roamStr := "off"
	if roam {
		roamStr = "on"
	}

	result, err := c.sendCommand(ctx, &ConnectionModeCommand{
		ConnectionMode:    string(mode),
		RoamSettingOption: roamStr,
	})
	if err != nil {
		return fmt.Errorf("failed to set connection mode: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to set connection mode: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SetNetworkBearerPreference(ctx context.Context, preference router.BearerPreference) error {
	switch preference {
	case router.BearerPreferenceLteAndNr5g, router.BearerPreferenceNr5gNsa, router.BearerPreferenceOnlyNr5g:
		return fmt.Errorf("5G bearer preferences are not supported on MF289F")
	}

	result, err := c.sendCommand(ctx, &BearerPreferenceCommand{
		BearerPreference: string(preference),
	})
	if err != nil {
		return fmt.Errorf("failed to set network bearer preference: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to set network bearer preference: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SetUpnp(ctx context.Context, enabled bool) error {
	upnpValue := 0
	if enabled {
		upnpValue = 1
	}

	result, err := c.sendCommand(ctx, &UpnpCommand{
		UpnpSettingOption: upnpValue,
	})
	if err != nil {
		return fmt.Errorf("failed to set UPnP: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to set UPnP: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SetDmz(ctx context.Context, ipAddress *string) error {
	result, err := c.sendCommand(ctx, &DmzCommand{
		DMZEnabled:   boolToInt(ipAddress != nil),
		DMZIPAddress: ipAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to set DMZ: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to set DMZ: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SelectLteBand(ctx context.Context, bands []router.LteBand) error {
	lteBandLock := router.SelectLTEBand(bands)

	result, err := c.sendCommand(ctx, &LockLteBandCommand{
		LteBandLock: lteBandLock,
	})
	if err != nil {
		return fmt.Errorf("failed to select LTE band: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to select LTE band: %s", result)
	}
	return nil
}

func (c *Mf289fClient) SetDNS(ctx context.Context, manual *[2]string) error {
	dnsMode := "auto"
	preferDNSManual := ""
	standbyDNSManual := ""

	if manual != nil {
		dnsMode = "manual"
		preferDNSManual = manual[0]
		standbyDNSManual = manual[1]
	}

	result, err := c.sendCommand(ctx, &DnsModeCommand{
		DnsMode:          dnsMode,
		PreferDNSManual:  preferDNSManual,
		StandbyDNSManual: standbyDNSManual,
	})
	if err != nil {
		return fmt.Errorf("failed to set DNS: %w", err)
	}
	if result != "success" {
		return fmt.Errorf("failed to set DNS: %s", result)
	}
	return nil
}

func (c *Mf289fClient) GetStatus(ctx context.Context) (json.RawMessage, error) {
	commandSet := "imei,imsi,dns_mode,prefer_dns_manual,standby_dns_manual,network_type,network_provider,mcc,mnc,rssi,rsrq,lte_rsrp,wan_lte_ca,lte_ca_pcell_band,lte_ca_pcell_bandwidth,lte_ca_scell_band,lte_ca_scell_bandwidth,lte_ca_pcell_arfcn,lte_ca_scell_arfcn,Z_SINR,Z_CELL_ID,Z_eNB_id,Z_rsrq,lte_ca_scell_info,wan_ipaddr,ipv6_wan_ipaddr,static_wan_ipaddr,opms_wan_mode,opms_wan_auto_mode,ppp_status,loginfo"

	resp, err := c.getCommand(ctx, commandSet)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}

	return json.RawMessage(data), nil
}

func (c *Mf289fClient) GetAPNMode(ctx context.Context) (bool, error) {
	return false, &router.ErrNotSupported{Method: "get_apn_mode"}
}

func (c *Mf289fClient) SetAPNMode(ctx context.Context, manual bool) error {
	return &router.ErrNotSupported{Method: "set_apn_mode"}
}

func (c *Mf289fClient) GetAPNProfiles(ctx context.Context) ([]router.ApnProfile, error) {
	return nil, &router.ErrNotSupported{Method: "get_apn_profiles"}
}

func (c *Mf289fClient) SetAPNProfile(ctx context.Context, profile *router.ApnProfile) error {
	return &router.ErrNotSupported{Method: "set_apn_profile"}
}

func (c *Mf289fClient) EnableAPNProfile(ctx context.Context, profileID string) error {
	return &router.ErrNotSupported{Method: "enable_apn_profile"}
}

func (c *Mf289fClient) GetDHCPSettings(ctx context.Context) (*router.DhcpSettings, error) {
	return nil, &router.ErrNotSupported{Method: "get_dhcp_settings"}
}

func (c *Mf289fClient) SetDHCPSettings(ctx context.Context, settings *router.DhcpSettings) error {
	return &router.ErrNotSupported{Method: "set_dhcp_settings"}
}

func (c *Mf289fClient) GetMTUSettings(ctx context.Context) (*router.MtuSettings, error) {
	return nil, &router.ErrNotSupported{Method: "get_mtu_settings"}
}

func (c *Mf289fClient) SetMTUSettings(ctx context.Context, settings *router.MtuSettings) error {
	return &router.ErrNotSupported{Method: "set_mtu_settings"}
}

func (c *Mf289fClient) GetSMSSettings(ctx context.Context) (*router.SmsSettings, error) {
	return nil, &router.ErrNotSupported{Method: "get_sms_settings"}
}

func (c *Mf289fClient) GetNetworkInfo(ctx context.Context) (json.RawMessage, error) {
	return nil, &router.ErrNotSupported{Method: "get_network_info"}
}

func (c *Mf289fClient) GetSIMInfo(ctx context.Context) (json.RawMessage, error) {
	return nil, &router.ErrNotSupported{Method: "get_sim_info"}
}

func (c *Mf289fClient) GetDeviceInfo(ctx context.Context) (json.RawMessage, error) {
	return nil, &router.ErrNotSupported{Method: "get_device_info"}
}

func (c *Mf289fClient) GetConnectedDevices(ctx context.Context) (json.RawMessage, error) {
	return nil, &router.ErrNotSupported{Method: "get_connected_devices"}
}

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return strings.ToUpper(fmt.Sprintf("%x", h))
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
