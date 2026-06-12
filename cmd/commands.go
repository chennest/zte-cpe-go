package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/1zun4/zte-cpe-go/pkg/g5ts"
	"github.com/1zun4/zte-cpe-go/pkg/mf289f"
	"github.com/1zun4/zte-cpe-go/pkg/router"
	"github.com/spf13/cobra"
)

var (
	routerType string
	routerURL  string
	password   string
)

func addCommonFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	cmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("url")
	cmd.MarkFlagRequired("password")
}

func addCommonFlagsNoPassword(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	cmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("url")
}

func jsonPrint(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}


func parseOnOff(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value: %s (use 'on' or 'off')", s)
	}
}

func getClient() (router.RouterClient, error) {
	switch routerType {
	case "mf289f":
		return mf289f.New(routerURL)
	case "g5ts":
		return g5ts.New(routerURL)
	default:
		return nil, fmt.Errorf("unsupported router type: %s (use 'mf289f' or 'g5ts')", routerType)
	}
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the router",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Println("Login successful")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from the router",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Logout(ctx); err != nil {
			return fmt.Errorf("logout failed: %w", err)
		}

		fmt.Println("Logout successful")
		return nil
	},
}

var rebootCmd = &cobra.Command{
	Use:   "reboot",
	Short: "Reboot the router",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.Reboot(ctx); err != nil {
			return fmt.Errorf("reboot failed: %w", err)
		}

		fmt.Println("Reboot command sent")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get router status",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		status, err := client.GetStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get status: %w", err)
		}

		fmt.Println(string(status))
		return nil
	},
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.ConnectNetwork(ctx); err != nil {
			return fmt.Errorf("connect failed: %w", err)
		}

		fmt.Println("Connected to network")
		return nil
	},
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.DisconnectNetwork(ctx); err != nil {
			return fmt.Errorf("disconnect failed: %w", err)
		}

		fmt.Println("Disconnected from network")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Get firmware/hardware version",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		versionA, versionB, err := client.GetVersion(ctx)
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}

		fmt.Printf("Version A: %s\n", versionA)
		fmt.Printf("Version B: %s\n", versionB)
		return nil
	},
}

var setConnectionModeCmd = &cobra.Command{
	Use:   "set-connection-mode",
	Short: "Set connection dial mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		modeStr, _ := cmd.Flags().GetString("mode")
		roam, _ := cmd.Flags().GetBool("roam")

		var mode router.ConnectionMode
		switch strings.ToLower(modeStr) {
		case "auto":
			mode = router.ConnectionModeAuto
		case "manual":
			mode = router.ConnectionModeManual
		default:
			return fmt.Errorf("invalid mode: %s (use 'auto' or 'manual')", modeStr)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SetConnectionMode(ctx, mode, roam); err != nil {
			return fmt.Errorf("failed to set connection mode: %w", err)
		}

		fmt.Println("Connection mode set successfully")
		return nil
	},
}

var setBearerCmd = &cobra.Command{
	Use:   "set-bearer [preference]",
	Short: "Set network bearer preference",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prefStr := strings.ToLower(args[0])

		var pref router.BearerPreference
		switch prefStr {
		case "auto":
			pref = router.BearerPreferenceAuto
		case "lte-and-nr5g":
			pref = router.BearerPreferenceLteAndNr5g
		case "nr5g-nsa":
			pref = router.BearerPreferenceNr5gNsa
		case "only-nr5g":
			pref = router.BearerPreferenceOnlyNr5g
		case "only-lte":
			pref = router.BearerPreferenceOnlyLte
		case "only-gsm":
			pref = router.BearerPreferenceOnlyGsm
		case "only-wcdma":
			pref = router.BearerPreferenceOnlyWcdma
		default:
			return fmt.Errorf("invalid bearer preference: %s (use auto, lte-and-nr5g, nr5g-nsa, only-nr5g, only-lte, only-gsm, only-wcdma)", prefStr)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SetNetworkBearerPreference(ctx, pref); err != nil {
			return fmt.Errorf("failed to set bearer preference: %w", err)
		}

		fmt.Println("Bearer preference set successfully")
		return nil
	},
}

var setUpnpCmd = &cobra.Command{
	Use:   "set-upnp",
	Short: "Enable or disable UPnP",
	RunE: func(cmd *cobra.Command, args []string) error {
		enabledStr, _ := cmd.Flags().GetString("enabled")
		enabled, err := parseOnOff(enabledStr)
		if err != nil {
			return err
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SetUpnp(ctx, enabled); err != nil {
			return fmt.Errorf("failed to set UPnP: %w", err)
		}

		fmt.Println("UPnP set successfully")
		return nil
	},
}

var setDmzCmd = &cobra.Command{
	Use:   "set-dmz [target]",
	Short: "Set DMZ host (IP address or 'off')",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		var ipPtr *string
		if strings.ToLower(target) != "off" {
			if net.ParseIP(target) == nil {
				return fmt.Errorf("invalid IP address: %s", target)
			}
			ipPtr = &target
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SetDmz(ctx, ipPtr); err != nil {
			return fmt.Errorf("failed to set DMZ: %w", err)
		}

		if ipPtr != nil {
			fmt.Printf("DMZ set to %s\n", *ipPtr)
		} else {
			fmt.Println("DMZ disabled")
		}
		return nil
	},
}

var setDnsCmd = &cobra.Command{
	Use:   "set-dns",
	Short: "Set DNS configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		modeStr, _ := cmd.Flags().GetString("mode")

		var manual *[2]string
		switch strings.ToLower(modeStr) {
		case "auto":
			manual = nil
		case "manual":
			primary, _ := cmd.Flags().GetString("primary")
			secondary, _ := cmd.Flags().GetString("secondary")
			if primary == "" || secondary == "" {
				return fmt.Errorf("--primary and --secondary are required when mode is manual")
			}
			manual = &[2]string{primary, secondary}
		default:
			return fmt.Errorf("invalid mode: %s (use 'auto' or 'manual')", modeStr)
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SetDNS(ctx, manual); err != nil {
			return fmt.Errorf("failed to set DNS: %w", err)
		}

		fmt.Println("DNS set successfully")
		return nil
	},
}

var selectLteBandCmd = &cobra.Command{
	Use:   "select-lte-band [bands]",
	Short: "Select LTE bands (comma-separated band numbers, or 'all')",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bandsArg := args[0]

		var bands []router.LteBand
		if strings.ToLower(bandsArg) != "all" {
			parts := strings.Split(bandsArg, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				band, err := router.ParseLTEBand(p)
				if err != nil {
					return err
				}
				bands = append(bands, band)
			}
			if len(bands) == 0 {
				return fmt.Errorf("no valid bands specified")
			}
		}

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		if err := client.SelectLteBand(ctx, bands); err != nil {
			return fmt.Errorf("failed to select LTE band: %w", err)
		}

		if bands == nil {
			fmt.Println("LTE bands set to all")
		} else {
			fmt.Printf("LTE bands set: %v\n", bands)
		}
		return nil
	},
}

var getApnCmd = &cobra.Command{
	Use:   "get-apn",
	Short: "Get APN mode and profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		manual, err := client.GetAPNMode(ctx)
		if err != nil {
			return fmt.Errorf("failed to get APN mode: %w", err)
		}

		profiles, err := client.GetAPNProfiles(ctx)
		if err != nil {
			return fmt.Errorf("failed to get APN profiles: %w", err)
		}

		result := map[string]interface{}{
			"manual":   manual,
			"profiles": profiles,
		}
		return jsonPrint(result)
	},
}

var setApnCmd = &cobra.Command{
	Use:   "set-apn",
	Short: "Set APN profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		apn, _ := cmd.Flags().GetString("apn")
		pdpType, _ := cmd.Flags().GetString("pdp-type")
		auth, _ := cmd.Flags().GetString("auth")
		username, _ := cmd.Flags().GetString("username")
		passwordVal, _ := cmd.Flags().GetString("apn-password")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		profiles, err := client.GetAPNProfiles(ctx)
		if err != nil {
			return fmt.Errorf("failed to get APN profiles: %w", err)
		}

		var existing *router.ApnProfile
		for i := range profiles {
			if profiles[i].ProfileID != nil && *profiles[i].ProfileID == id {
				existing = &profiles[i]
				break
			}
		}
		if existing == nil {
			return fmt.Errorf("APN profile with ID %s not found", id)
		}

		profile := *existing
		if name != "" {
			profile.ProfileName = name
		}
		if apn != "" {
			profile.Apn = apn
		}
		if pdpType != "" {
			profile.PdpType = router.PdpType(pdpType)
		}
		if auth != "" {
			profile.AuthMode = router.ApnAuthMode(auth)
		}
		if username != "" {
			profile.Username = username
		}
		if passwordVal != "" {
			profile.Password = passwordVal
		}

		if err := client.SetAPNProfile(ctx, &profile); err != nil {
			return fmt.Errorf("failed to set APN profile: %w", err)
		}

		fmt.Println("APN profile set successfully")
		return nil
	},
}

var getDhcpCmd = &cobra.Command{
	Use:   "get-dhcp",
	Short: "Get DHCP settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		settings, err := client.GetDHCPSettings(ctx)
		if err != nil {
			return fmt.Errorf("failed to get DHCP settings: %w", err)
		}

		return jsonPrint(settings)
	},
}

var setDhcpCmd = &cobra.Command{
	Use:   "set-dhcp",
	Short: "Set DHCP settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, _ := cmd.Flags().GetString("ip")
		subnet, _ := cmd.Flags().GetString("subnet")
		enabledStr, _ := cmd.Flags().GetString("enabled")
		leaseTime, _ := cmd.Flags().GetUint32("lease-time")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		current, err := client.GetDHCPSettings(ctx)
		if err != nil {
			return fmt.Errorf("failed to get DHCP settings: %w", err)
		}

		settings := *current
		if ip != "" {
			settings.IPAddress = ip
		}
		if subnet != "" {
			settings.SubnetMask = subnet
		}
		if enabledStr != "" {
			enabled, err := parseOnOff(enabledStr)
			if err != nil {
				return err
			}
			settings.DhcpEnabled = enabled
		}
		if leaseTime > 0 {
			settings.LeaseTime = leaseTime
		}

		if err := client.SetDHCPSettings(ctx, &settings); err != nil {
			return fmt.Errorf("failed to set DHCP settings: %w", err)
		}

		fmt.Println("DHCP settings set successfully")
		return nil
	},
}

var getMtuCmd = &cobra.Command{
	Use:   "get-mtu",
	Short: "Get MTU/MSS settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		settings, err := client.GetMTUSettings(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MTU settings: %w", err)
		}

		return jsonPrint(settings)
	},
}

var setMtuCmd = &cobra.Command{
	Use:   "set-mtu",
	Short: "Set MTU/MSS settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		mtu, _ := cmd.Flags().GetUint32("mtu")
		mss, _ := cmd.Flags().GetUint32("mss")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		current, err := client.GetMTUSettings(ctx)
		if err != nil {
			return fmt.Errorf("failed to get MTU settings: %w", err)
		}

		settings := *current
		if mtu > 0 {
			settings.Mtu = mtu
		}
		if mss > 0 {
			settings.Mss = mss
		}

		if err := client.SetMTUSettings(ctx, &settings); err != nil {
			return fmt.Errorf("failed to set MTU settings: %w", err)
		}

		fmt.Println("MTU settings set successfully")
		return nil
	},
}

var getSmsSettingsCmd = &cobra.Command{
	Use:   "get-sms-settings",
	Short: "Get SMS settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		settings, err := client.GetSMSSettings(ctx)
		if err != nil {
			return fmt.Errorf("failed to get SMS settings: %w", err)
		}

		return jsonPrint(settings)
	},
}

var networkInfoCmd = &cobra.Command{
	Use:   "network-info",
	Short: "Get network/signal information",
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		info, err := client.GetNetworkInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get network info: %w", err)
		}

		if pretty {
			var v interface{}
			if err := json.Unmarshal(info, &v); err != nil {
				fmt.Println(string(info))
				return nil
			}
			return jsonPrint(v)
		}
		fmt.Println(string(info))
		return nil
	},
}

var simInfoCmd = &cobra.Command{
	Use:   "sim-info",
	Short: "Get SIM card information",
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		info, err := client.GetSIMInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get SIM info: %w", err)
		}

		if pretty {
			var v interface{}
			if err := json.Unmarshal(info, &v); err != nil {
				fmt.Println(string(info))
				return nil
			}
			return jsonPrint(v)
		}
		fmt.Println(string(info))
		return nil
	},
}

var deviceInfoCmd = &cobra.Command{
	Use:   "device-info",
	Short: "Get device information",
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		info, err := client.GetDeviceInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to get device info: %w", err)
		}

		if pretty {
			var v interface{}
			if err := json.Unmarshal(info, &v); err != nil {
				fmt.Println(string(info))
				return nil
			}
			return jsonPrint(v)
		}
		fmt.Println(string(info))
		return nil
	},
}

var connectedDevicesCmd = &cobra.Command{
	Use:   "connected-devices",
	Short: "Get connected devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")

		client, err := getClient()
		if err != nil {
			return err
		}

		ctx := context.Background()
		if err := client.Login(ctx, password); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer client.Logout(ctx)

		info, err := client.GetConnectedDevices(ctx)
		if err != nil {
			return fmt.Errorf("failed to get connected devices: %w", err)
		}

		if pretty {
			var v interface{}
			if err := json.Unmarshal(info, &v); err != nil {
				fmt.Println(string(info))
				return nil
			}
			return jsonPrint(v)
		}
		fmt.Println(string(info))
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	loginCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	loginCmd.MarkFlagRequired("type")
	loginCmd.MarkFlagRequired("url")
	loginCmd.MarkFlagRequired("password")

	logoutCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	logoutCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	logoutCmd.MarkFlagRequired("type")
	logoutCmd.MarkFlagRequired("url")

	rebootCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	rebootCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	rebootCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	rebootCmd.MarkFlagRequired("type")
	rebootCmd.MarkFlagRequired("url")
	rebootCmd.MarkFlagRequired("password")

	statusCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	statusCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	statusCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	statusCmd.MarkFlagRequired("type")
	statusCmd.MarkFlagRequired("url")
	statusCmd.MarkFlagRequired("password")

	connectCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	connectCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	connectCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	connectCmd.MarkFlagRequired("type")
	connectCmd.MarkFlagRequired("url")
	connectCmd.MarkFlagRequired("password")

	disconnectCmd.Flags().StringVarP(&routerType, "type", "t", "", "Router type (mf289f or g5ts)")
	disconnectCmd.Flags().StringVarP(&routerURL, "url", "u", "", "Router URL (e.g., http://192.168.0.1)")
	disconnectCmd.Flags().StringVarP(&password, "password", "p", "", "Router password")
	disconnectCmd.MarkFlagRequired("type")
	disconnectCmd.MarkFlagRequired("url")
	disconnectCmd.MarkFlagRequired("password")

	addCommonFlags(versionCmd)
	addCommonFlags(setConnectionModeCmd)
	addCommonFlags(setBearerCmd)
	addCommonFlags(setUpnpCmd)
	addCommonFlags(setDmzCmd)
	addCommonFlags(setDnsCmd)
	addCommonFlags(selectLteBandCmd)
	addCommonFlags(getApnCmd)
	addCommonFlags(setApnCmd)
	addCommonFlags(getDhcpCmd)
	addCommonFlags(setDhcpCmd)
	addCommonFlags(getMtuCmd)
	addCommonFlags(setMtuCmd)
	addCommonFlags(getSmsSettingsCmd)
	addCommonFlags(networkInfoCmd)
	addCommonFlags(simInfoCmd)
	addCommonFlags(deviceInfoCmd)
	addCommonFlags(connectedDevicesCmd)

	setConnectionModeCmd.Flags().String("mode", "", "Connection mode (auto|manual)")
	setConnectionModeCmd.Flags().Bool("roam", false, "Enable roaming")
	setConnectionModeCmd.MarkFlagRequired("mode")

	setUpnpCmd.Flags().String("enabled", "", "Enable or disable UPnP (on|off)")
	setUpnpCmd.MarkFlagRequired("enabled")

	setDnsCmd.Flags().String("mode", "", "DNS mode (auto|manual)")
	setDnsCmd.Flags().String("primary", "", "Primary DNS server (required when mode=manual)")
	setDnsCmd.Flags().String("secondary", "", "Secondary DNS server (required when mode=manual)")
	setDnsCmd.MarkFlagRequired("mode")

	setApnCmd.Flags().String("id", "", "APN profile ID to modify")
	setApnCmd.Flags().String("name", "", "Profile name")
	setApnCmd.Flags().String("apn", "", "APN address")
	setApnCmd.Flags().String("pdp-type", "", "PDP type (IPv4|IPv6|IPv4v6)")
	setApnCmd.Flags().String("auth", "", "Auth mode (NONE|PAP|CHAP|PAP_CHAP)")
	setApnCmd.Flags().String("username", "", "APN username")
	setApnCmd.Flags().String("apn-password", "", "APN password")
	setApnCmd.MarkFlagRequired("id")

	setDhcpCmd.Flags().String("ip", "", "Router IP address")
	setDhcpCmd.Flags().String("subnet", "", "Subnet mask")
	setDhcpCmd.Flags().String("enabled", "", "DHCP enabled (on|off)")
	setDhcpCmd.Flags().Uint32("lease-time", 0, "DHCP lease time in seconds")

	setMtuCmd.Flags().Uint32("mtu", 0, "MTU value")
	setMtuCmd.Flags().Uint32("mss", 0, "MSS value")

	networkInfoCmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
	simInfoCmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
	deviceInfoCmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
	connectedDevicesCmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
}
