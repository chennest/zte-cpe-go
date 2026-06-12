package g5ts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync/atomic"
	"time"

	"github.com/1zun4/zte-cpe-go/pkg/router"
)

const nullSession = "00000000000000000000000000000000"

// G5tsClient implements the RouterClient interface for ZTE G5TS routers
// using the ubus JSON-RPC 2.0 protocol over HTTP/HTTPS.
type G5tsClient struct {
	target    string
	client    *http.Client
	session   string
	requestID int64
}

// ubusResponse represents a single ubus JSON-RPC 2.0 response.
type ubusResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ubusError      `json:"error,omitempty"`
}

// ubusError represents a ubus JSON-RPC 2.0 error.
type ubusError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ubusError) Error() string {
	return fmt.Sprintf("ubus error %d: %s", e.Code, e.Message)
}

// loginInfo holds the response from web_login_info.
type loginInfo struct {
	ZteWebSault  string `json:"zte_web_sault"`
	LoginFailNum int64  `json:"login_fail_num"`
}

// loginResult holds the response from web_login.
type loginResult struct {
	Result         int64  `json:"result"`
	UbusRpcSession string `json:"ubus_rpc_session"`
	Timeout        int64  `json:"timeout"`
}

// New creates a new G5TS client.
func New(url string) (*G5tsClient, error) {
	normalized, err := router.NormalizeRouterURL(url)
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	client := &http.Client{
		Jar: jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	return &G5tsClient{
		target:    normalized,
		client:    client,
		session:   nullSession,
		requestID: 1,
	}, nil
}

func (c *G5tsClient) nextRequestID() int64 {
	return atomic.AddInt64(&c.requestID, 1) - 1
}

// sendCommandWithSession sends a ubus JSON-RPC command with the given session token.
// Returns the data portion of the [code, data] result array.
func (c *G5tsClient) sendCommandWithSession(ctx context.Context, session string, cmd UbusCommand) (json.RawMessage, error) {
	// Serialize command struct to get its fields as the args object
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(cmdJSON, &args); err != nil {
		return nil, fmt.Errorf("failed to parse command args: %w", err)
	}

	id := c.nextRequestID()
	ts := time.Now().Unix()

	url := fmt.Sprintf("%subus/?t=%d", c.target, ts)

	// Build the JSON-RPC request: params is [session, module, method, args]
	type rpcRequest struct {
		JSONRPC string        `json:"jsonrpc"`
		ID      int64         `json:"id"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}

	reqBody := []rpcRequest{{
		JSONRPC: "2.0",
		ID:      id,
		Method:  "call",
		Params:  []interface{}{session, cmd.Module(), cmd.Method(), args},
	}}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", c.target)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Response is a JSON array of ubus responses
	var responses []ubusResponse
	if err := json.Unmarshal(body, &responses); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w (body: %s)", err, truncate(string(body), 200))
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("empty ubus response")
	}

	rpcResp := responses[0]

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	// Result is [code] or [code, data]
	var resultArr []json.RawMessage
	if err := json.Unmarshal(rpcResp.Result, &resultArr); err != nil {
		return nil, fmt.Errorf("ubus result is not an array: %w", err)
	}

	if len(resultArr) == 0 {
		return nil, fmt.Errorf("empty result array")
	}

	var code int64
	if err := json.Unmarshal(resultArr[0], &code); err != nil {
		return nil, fmt.Errorf("failed to parse result code: %w", err)
	}

	if code != 0 {
		return nil, fmt.Errorf("ubus error code %d", code)
	}

	if len(resultArr) > 1 {
		return resultArr[1], nil
	}

	return json.RawMessage("{}"), nil
}

// sendCommand sends a ubus command using the current session token.
func (c *G5tsClient) sendCommand(ctx context.Context, cmd UbusCommand) (json.RawMessage, error) {
	return c.sendCommandWithSession(ctx, c.session, cmd)
}

// Login authenticates with the router.
func (c *G5tsClient) Login(ctx context.Context, password string) error {
	// Step 1: Get login info (salt + remaining attempts)
	result, err := c.sendCommandWithSession(ctx, nullSession, &LoginInfoCommand{})
	if err != nil {
		return fmt.Errorf("failed to get login info: %w", err)
	}

	var info loginInfo
	if err := json.Unmarshal(result, &info); err != nil {
		return fmt.Errorf("failed to parse login info: %w", err)
	}

	if info.LoginFailNum <= 0 {
		return fmt.Errorf("account is locked due to too many failed login attempts")
	}

	// Step 2: Compute password hash: SHA256(SHA256(password).UPPER + salt).UPPER
	hash := g5tsPasswordHash(password, info.ZteWebSault)

	// Step 3: Login with hashed password (using null session, c.session is still nullSession)
	loginResp, err := c.sendCommand(ctx, &LoginCommand{
		Password: hash,
	})
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}

	var lr loginResult
	if err := json.Unmarshal(loginResp, &lr); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	switch lr.Result {
	case 0:
		if lr.UbusRpcSession == "" {
			return fmt.Errorf("login succeeded but no session token returned")
		}
		c.session = lr.UbusRpcSession
	case 3:
		return fmt.Errorf("another user is already logged in")
	default:
		return fmt.Errorf("login failed (result=%d)", lr.Result)
	}

	// Step 4: Optional encryption setup — skipped (requires RSA, not in go.mod).
	// AES encryption is only needed for specific fields like SMS body / APN password.

	return nil
}

// Logout logs out from the router.
func (c *G5tsClient) Logout(ctx context.Context) error {
	_, err := c.sendCommand(ctx, &LogoutCommand{})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	c.session = nullSession
	return nil
}

// Reboot reboots the router.
func (c *G5tsClient) Reboot(ctx context.Context) error {
	_, err := c.sendCommand(ctx, NewRebootCommand())
	if err != nil {
		return fmt.Errorf("failed to reboot: %w", err)
	}
	return nil
}

// DisconnectNetwork disconnects the mobile/WAN network.
func (c *G5tsClient) DisconnectNetwork(ctx context.Context) error {
	cmd := NewSetWwanIfaceCommand()
	enable := 0
	cmd.Enable = &enable

	_, err := c.sendCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to disconnect network: %w", err)
	}
	return nil
}

// ConnectNetwork connects the mobile/WAN network.
func (c *G5tsClient) ConnectNetwork(ctx context.Context) error {
	cmd := NewSetWwanIfaceCommand()
	enable := 1
	cmd.Enable = &enable

	_, err := c.sendCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to connect network: %w", err)
	}
	return nil
}

// GetVersion returns firmware/hardware version.
func (c *G5tsClient) GetVersion(ctx context.Context) (string, string, error) {
	result, err := c.uciGet(ctx, "zwrt_common_info", strPtr("common_config"))
	if err != nil {
		return "", "", fmt.Errorf("failed to get version: %w", err)
	}

	var parsed struct {
		Values struct {
			HardwareVersion string `json:"hardware_version"`
			WaInnerVersion  string `json:"wa_inner_version"`
		} `json:"values"`
	}
	if err := json.Unmarshal(result, &parsed); err != nil {
		return "", "", fmt.Errorf("failed to parse version: %w", err)
	}

	return parsed.Values.HardwareVersion, parsed.Values.WaInnerVersion, nil
}

// SetConnectionMode sets connection mode and roaming.
func (c *G5tsClient) SetConnectionMode(ctx context.Context, mode router.ConnectionMode, roam bool) error {
	cmd := NewSetWwanIfaceCommand()

	connectMode := 1
	if mode == router.ConnectionModeAuto {
		connectMode = 0
	}
	cmd.ConnectMode = &connectMode

	roamEnable := 0
	if roam {
		roamEnable = 1
	}
	cmd.RoamEnable = &roamEnable

	_, err := c.sendCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to set connection mode: %w", err)
	}
	return nil
}

// SetNetworkBearerPreference sets network bearer preference.
func (c *G5tsClient) SetNetworkBearerPreference(ctx context.Context, preference router.BearerPreference) error {
	var netSelect string
	switch preference {
	case router.BearerPreferenceAuto, router.BearerPreferenceLteAndNr5g:
		netSelect = "4G_AND_5G"
	case router.BearerPreferenceNr5gNsa:
		netSelect = "LTE_AND_5G"
	case router.BearerPreferenceOnlyNr5g:
		netSelect = "Only_5G"
	case router.BearerPreferenceOnlyLte:
		netSelect = "Only_LTE"
	case router.BearerPreferenceOnlyGsm, router.BearerPreferenceOnlyWcdma:
		return fmt.Errorf("GSM/WCDMA-only bearer preferences are not supported on G5TS")
	default:
		return fmt.Errorf("unsupported bearer preference: %s", preference)
	}

	_, err := c.sendCommand(ctx, &SetNetSelectCommand{
		NetSelect: netSelect,
	})
	if err != nil {
		return fmt.Errorf("failed to set network bearer preference: %w", err)
	}
	return nil
}

// SetUpnp enables or disables UPnP.
func (c *G5tsClient) SetUpnp(ctx context.Context, enabled bool) error {
	enableUpnp := 0
	if enabled {
		enableUpnp = 1
	}

	_, err := c.sendCommand(ctx, &SetUpnpCommand{
		EnableUpnp: enableUpnp,
	})
	if err != nil {
		return fmt.Errorf("failed to set UPnP: %w", err)
	}
	return nil
}

// SetDmz sets DMZ host IP, or disables DMZ with nil.
func (c *G5tsClient) SetDmz(ctx context.Context, ipAddress *string) error {
	dmzEnable := 0
	if ipAddress != nil {
		dmzEnable = 1
	}

	_, err := c.sendCommand(ctx, &SetDmzCommand{
		DmzEnable: dmzEnable,
		DmzIP:     ipAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to set DMZ: %w", err)
	}
	return nil
}

// SelectLteBand is not supported on G5TS.
func (c *G5tsClient) SelectLteBand(ctx context.Context, bands []router.LteBand) error {
	return fmt.Errorf("select_lte_band is not supported on G5TS")
}

// SetDNS is not directly supported on G5TS.
func (c *G5tsClient) SetDNS(ctx context.Context, manual *[2]string) error {
	return &router.ErrNotSupported{Method: "set_dns"}
}

// GetStatus returns comprehensive router status info as JSON.
func (c *G5tsClient) GetStatus(ctx context.Context) (json.RawMessage, error) {
	simInfo, _ := c.GetSIMInfo(ctx)
	wwan, _ := c.sendCommand(ctx, NewGetWwanIfaceCommand())
	deviceInfo, _ := c.uciGet(ctx, "zwrt_zte_mdm", strPtr("device_info"))
	commonConfig, _ := c.uciGet(ctx, "zwrt_common_info", strPtr("common_config"))
	routerStatus, _ := c.sendCommand(ctx, &GetRouterStatusCommand{})

	deviceValues := extractUCIValues(deviceInfo)
	commonValues := extractUCIValues(commonConfig)

	result := map[string]json.RawMessage{
		"sim_info":      rawOrDefault(simInfo),
		"wwan":          rawOrDefault(wwan),
		"device_info":   rawOrDefault(deviceValues),
		"common_config": rawOrDefault(commonValues),
		"router_status": rawOrDefault(routerStatus),
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}
	return out, nil
}

// GetAPNMode returns the current APN mode.
func (c *G5tsClient) GetAPNMode(ctx context.Context) (bool, error) {
	result, err := c.sendCommand(ctx, &GetApnModeCommand{})
	if err != nil {
		return false, fmt.Errorf("failed to get APN mode: %w", err)
	}

	var apnMode struct {
		ApnMode json.Number `json:"apn_mode"`
	}
	if err := json.Unmarshal(result, &apnMode); err != nil {
		return false, fmt.Errorf("failed to parse APN mode: %w", err)
	}

	n, _ := apnMode.ApnMode.Int64()
	if n == 0 {
		s := apnMode.ApnMode.String()
		if s == "1" {
			return true, nil
		}
	}
	return n == 1, nil
}

// SetAPNMode sets the APN mode.
func (c *G5tsClient) SetAPNMode(ctx context.Context, manual bool) error {
	apnMode := 0
	if manual {
		apnMode = 1
	}

	_, err := c.sendCommand(ctx, &SetApnModeCommand{
		ApnMode: apnMode,
	})
	if err != nil {
		return fmt.Errorf("failed to set APN mode: %w", err)
	}
	return nil
}

// GetAPNProfiles returns the list of manual APN profiles.
func (c *G5tsClient) GetAPNProfiles(ctx context.Context) ([]router.ApnProfile, error) {
	result, err := c.sendCommand(ctx, &GetManuApnListCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get APN profiles: %w", err)
	}

	var raw struct {
		Profiles []json.RawMessage `json:"apnListArray"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse APN list: %w", err)
	}

	profiles := make([]router.ApnProfile, 0, len(raw.Profiles))
	for _, p := range raw.Profiles {
		var entry struct {
			ProfileID   interface{} `json:"profileId"`
			ProfileName string      `json:"profilename"`
			Apn         string      `json:"wanapn"`
			PdpType     string      `json:"pdpType"`
			AuthMode    string      `json:"pppAuthMode"`
			Username    string      `json:"username"`
		}
		if err := json.Unmarshal(p, &entry); err != nil {
			continue
		}

		profileID := resolveString(entry.ProfileID)

		authMode := router.ApnAuthModeNone
		switch entry.AuthMode {
		case "1", "PAP":
			authMode = router.ApnAuthModePap
		case "2", "CHAP":
			authMode = router.ApnAuthModeChap
		case "3", "PAP_CHAP":
			authMode = router.ApnAuthModePapChap
		}

		pdpType := router.PdpTypeIPv4v6
		switch entry.PdpType {
		case "IPv4", "IP":
			pdpType = router.PdpTypeIPv4
		case "IPv6":
			pdpType = router.PdpTypeIPv6
		}

		profiles = append(profiles, router.ApnProfile{
			ProfileID:   &profileID,
			ProfileName: entry.ProfileName,
			Apn:         entry.Apn,
			PdpType:     pdpType,
			AuthMode:    authMode,
			Username:    entry.Username,
			Password:    "", // password is encrypted, don't expose
		})
	}

	return profiles, nil
}

// SetAPNProfile modifies an existing manual APN profile.
func (c *G5tsClient) SetAPNProfile(ctx context.Context, profile *router.ApnProfile) error {
	if profile.ProfileID == nil {
		return fmt.Errorf("profile ID is required")
	}

	authMode := "0"
	switch profile.AuthMode {
	case router.ApnAuthModePap:
		authMode = "1"
	case router.ApnAuthModeChap:
		authMode = "2"
	case router.ApnAuthModePapChap:
		authMode = "3"
	}

	pdpType := "IPv4v6"
	switch profile.PdpType {
	case router.PdpTypeIPv4:
		pdpType = "IPv4"
	case router.PdpTypeIPv6:
		pdpType = "IPv6"
	}

	cmd := &ModifyManuApnCommand{
		ProfileID:   *profile.ProfileID,
		ProfileName: profile.ProfileName,
		PdpType:     pdpType,
		Apn:         profile.Apn,
		AuthMode:    authMode,
		Username:    profile.Username,
		Password:    profile.Password,
	}

	_, err := c.sendCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to set APN profile: %w", err)
	}
	return nil
}

// EnableAPNProfile sets a manual APN profile as the active/default one.
func (c *G5tsClient) EnableAPNProfile(ctx context.Context, profileID string) error {
	_, err := c.sendCommand(ctx, &EnableManuApnCommand{
		ProfileID: profileID,
	})
	if err != nil {
		return fmt.Errorf("failed to enable APN profile: %w", err)
	}
	return nil
}

// GetDHCPSettings returns current DHCP settings.
func (c *G5tsClient) GetDHCPSettings(ctx context.Context) (*router.DhcpSettings, error) {
	lan, err := c.uciGet(ctx, "network", strPtr("lan"))
	if err != nil {
		return nil, fmt.Errorf("failed to get LAN settings: %w", err)
	}
	dhcp, err := c.uciGet(ctx, "dhcp", strPtr("lan"))
	if err != nil {
		return nil, fmt.Errorf("failed to get DHCP settings: %w", err)
	}
	routerDhcp, err := c.uciGet(ctx, "zwrt_router", strPtr("dhcp"))
	if err != nil {
		return nil, fmt.Errorf("failed to get router DHCP settings: %w", err)
	}

	lanVals := extractUCIValues(lan)
	dhcpVals := extractUCIValues(dhcp)
	routerVals := extractUCIValues(routerDhcp)

	ipAddress := getStringFromRaw(lanVals, "ipaddr", "192.168.0.1")
	subnetMask := getStringFromRaw(lanVals, "netmask", "255.255.255.0")

	ignore := getStringFromRaw(dhcpVals, "ignore", "0")
	dhcpEnabled := ignore != "1"

	leaseTimeStr := getStringFromRaw(routerVals, "leasetime", "24h")
	leaseTimeStr = strings.TrimSuffix(leaseTimeStr, "h")
	var leaseTime uint32 = 24
	if n, err := fmt.Sscanf(leaseTimeStr, "%d", &leaseTime); err == nil && n == 1 {
	}

	return &router.DhcpSettings{
		IPAddress:   ipAddress,
		SubnetMask:  subnetMask,
		DhcpEnabled: dhcpEnabled,
		LeaseTime:   leaseTime,
	}, nil
}

// SetDHCPSettings sets DHCP settings.
func (c *G5tsClient) SetDHCPSettings(ctx context.Context, settings *router.DhcpSettings) error {
	_, err := c.sendCommand(ctx, &SetLanParaCommand{
		IPAddr:    settings.IPAddress,
		Netmask:   settings.SubnetMask,
		Ignore:    boolToZeroOne(!settings.DhcpEnabled),
		Leasetime: fmt.Sprintf("%dh", settings.LeaseTime),
	})
	if err != nil {
		return fmt.Errorf("failed to set DHCP settings: %w", err)
	}
	return nil
}

// GetMTUSettings returns current MTU/MSS settings.
func (c *G5tsClient) GetMTUSettings(ctx context.Context) (*router.MtuSettings, error) {
	resp, err := c.uciGet(ctx, "zwrt_router", strPtr("network"))
	if err != nil {
		return nil, fmt.Errorf("failed to get MTU settings: %w", err)
	}

	vals := extractUCIValues(resp)
	mtuStr := getStringFromRaw(vals, "mtu", "1500")
	mssStr := getStringFromRaw(vals, "mss", "1460")

	var mtu, mss uint32 = 1500, 1460
	fmt.Sscanf(mtuStr, "%d", &mtu)
	fmt.Sscanf(mssStr, "%d", &mss)

	return &router.MtuSettings{Mtu: mtu, Mss: mss}, nil
}

// SetMTUSettings sets MTU/MSS settings.
func (c *G5tsClient) SetMTUSettings(ctx context.Context, settings *router.MtuSettings) error {
	_, err := c.sendCommand(ctx, &SetWanMtuCommand{
		Mtu: fmt.Sprintf("%d", settings.Mtu),
		Mss: fmt.Sprintf("%d", settings.Mss),
	})
	if err != nil {
		return fmt.Errorf("failed to set MTU/MSS: %w", err)
	}
	return nil
}

// GetSMSSettings returns SMS settings.
func (c *G5tsClient) GetSMSSettings(ctx context.Context) (*router.SmsSettings, error) {
	result, err := c.sendCommand(ctx, &GetSmsParameterCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SMS settings: %w", err)
	}

	validity := getStringFromRaw(result, "tp_validity_period", "")
	centerNumber := getStringFromRaw(result, "sca", "")
	deliveryReport := getStringFromRaw(result, "status_report_on", "0")

	return &router.SmsSettings{
		Validity:       validity,
		CenterNumber:   centerNumber,
		DeliveryReport: deliveryReport == "1",
	}, nil
}

// GetNetworkInfo returns network/signal information.
func (c *G5tsClient) GetNetworkInfo(ctx context.Context) (json.RawMessage, error) {
	result, err := c.sendCommand(ctx, &GetNetInfoCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %w", err)
	}
	return result, nil
}

// GetSIMInfo returns SIM card information.
func (c *G5tsClient) GetSIMInfo(ctx context.Context) (json.RawMessage, error) {
	result, err := c.sendCommand(ctx, &GetSimInfoCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get SIM info: %w", err)
	}
	return result, nil
}

// GetDeviceInfo returns device information.
func (c *G5tsClient) GetDeviceInfo(ctx context.Context) (json.RawMessage, error) {
	deviceInfo, err := c.uciGet(ctx, "zwrt_zte_mdm", strPtr("device_info"))
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}
	commonConfig, err := c.uciGet(ctx, "zwrt_common_info", strPtr("common_config"))
	if err != nil {
		return nil, fmt.Errorf("failed to get common config: %w", err)
	}

	deviceValues := extractUCIValues(deviceInfo)
	commonValues := extractUCIValues(commonConfig)

	result := map[string]json.RawMessage{
		"device_info":   rawOrDefault(deviceValues),
		"common_config": rawOrDefault(commonValues),
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal device info: %w", err)
	}
	return out, nil
}

// GetConnectedDevices returns the connected device list.
func (c *G5tsClient) GetConnectedDevices(ctx context.Context) (json.RawMessage, error) {
	userCount, err := c.sendCommand(ctx, &GetUserListNumCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user list count: %w", err)
	}

	total := getInt64FromRaw(userCount, "access_total_num")
	if total == 0 {
		return json.RawMessage(`{"devices":[]}`), nil
	}

	lanList, err := c.sendCommand(ctx, &GetLanAccessListCommand{})
	if err != nil {
		return nil, fmt.Errorf("failed to get LAN access list: %w", err)
	}

	return lanList, nil
}

// uciGet reads a UCI config section via ubus.
func (c *G5tsClient) uciGet(ctx context.Context, config string, section *string) (json.RawMessage, error) {
	return c.sendCommand(ctx, &UciGetCommand{
		Config:  config,
		Section: section,
	})
}

// g5tsPasswordHash computes SHA256(SHA256(password).UPPER + salt).UPPER
func g5tsPasswordHash(password, salt string) string {
	h1 := sha256.Sum256([]byte(password))
	hash1 := strings.ToUpper(fmt.Sprintf("%x", h1))
	concat := hash1 + salt
	h2 := sha256.Sum256([]byte(concat))
	return strings.ToUpper(fmt.Sprintf("%x", h2))
}

func strPtr(s string) *string {
	return &s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func rawOrDefault(msg json.RawMessage) json.RawMessage {
	if msg == nil {
		return json.RawMessage("{}")
	}
	return msg
}

// extractUCIValues extracts the "values" field from a UCI response.
func extractUCIValues(resp json.RawMessage) json.RawMessage {
	if resp == nil {
		return nil
	}
	var m struct {
		Values json.RawMessage `json:"values"`
	}
	if err := json.Unmarshal(resp, &m); err != nil || m.Values == nil {
		return nil
	}
	return m.Values
}

// getStringFromRaw extracts a string field from raw JSON.
func getStringFromRaw(raw json.RawMessage, key, defaultVal string) string {
	if raw == nil {
		return defaultVal
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return defaultVal
	}
	val, ok := m[key]
	if !ok {
		return defaultVal
	}
	var s string
	if err := json.Unmarshal(val, &s); err != nil {
		return defaultVal
	}
	return s
}

// getInt64FromRaw extracts an int64 field from raw JSON.
func getInt64FromRaw(raw json.RawMessage, key string) int64 {
	if raw == nil {
		return 0
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return 0
	}
	val, ok := m[key]
	if !ok {
		return 0
	}
	var n int64
	if err := json.Unmarshal(val, &n); err != nil {
		return 0
	}
	return n
}

// resolveString converts an interface{} (string or float64 from JSON) to string.
func resolveString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	case json.Number:
		return val.String()
	default:
		return ""
	}
}

func boolToZeroOne(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
