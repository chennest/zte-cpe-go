package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zte-cpe",
	Short: "ZTE CPE Router Management Tool",
	Long:  "A command-line tool for managing ZTE CPE routers (MF289F, G5TS)",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(rebootCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(disconnectCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(setConnectionModeCmd)
	rootCmd.AddCommand(setBearerCmd)
	rootCmd.AddCommand(setUpnpCmd)
	rootCmd.AddCommand(setDmzCmd)
	rootCmd.AddCommand(setDnsCmd)
	rootCmd.AddCommand(selectLteBandCmd)
	rootCmd.AddCommand(getApnCmd)
	rootCmd.AddCommand(setApnCmd)
	rootCmd.AddCommand(getDhcpCmd)
	rootCmd.AddCommand(setDhcpCmd)
	rootCmd.AddCommand(getMtuCmd)
	rootCmd.AddCommand(setMtuCmd)
	rootCmd.AddCommand(getSmsSettingsCmd)
	rootCmd.AddCommand(networkInfoCmd)
	rootCmd.AddCommand(simInfoCmd)
	rootCmd.AddCommand(deviceInfoCmd)
	rootCmd.AddCommand(connectedDevicesCmd)
}
