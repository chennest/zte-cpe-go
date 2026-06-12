package exporter

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1zun4/zte-cpe-go/pkg/router"
	"github.com/prometheus/client_golang/prometheus"
)

// Metric descriptors.
var (
	upDesc = prometheus.NewDesc(
		"zte_cpe_up",
		"Whether the exporter is running (always 1).",
		nil, nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		"zte_cpe_scrape_duration_seconds",
		"Duration of the last scrape in seconds.",
		nil, nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		"zte_cpe_scrape_success",
		"Whether the last scrape was successful (1) or not (0).",
		nil, nil,
	)

	signalRsrpDesc = prometheus.NewDesc(
		"zte_cpe_signal_rsrp_dbm",
		"LTE/5G RSRP in dBm.",
		[]string{"model", "network_type"}, nil,
	)
	signalRsrqDesc = prometheus.NewDesc(
		"zte_cpe_signal_rsrq_db",
		"LTE/5G RSRQ in dB.",
		[]string{"model", "network_type"}, nil,
	)
	signalSnrDesc = prometheus.NewDesc(
		"zte_cpe_signal_snr_db",
		"Signal-to-noise ratio in dB.",
		[]string{"model", "network_type"}, nil,
	)
	signalRssiDesc = prometheus.NewDesc(
		"zte_cpe_signal_rssi_dbm",
		"RSSI in dBm.",
		[]string{"model"}, nil,
	)
	signalBarDesc = prometheus.NewDesc(
		"zte_cpe_signal_bar",
		"Signal bar (0-5).",
		[]string{"model"}, nil,
	)

	connectedDevicesDesc = prometheus.NewDesc(
		"zte_cpe_connected_devices",
		"Number of connected devices.",
		[]string{"model"}, nil,
	)
	networkConnectedDesc = prometheus.NewDesc(
		"zte_cpe_network_connected",
		"Whether WAN is connected (1=connected, 0=disconnected).",
		[]string{"model"}, nil,
	)
	roamingEnabledDesc = prometheus.NewDesc(
		"zte_cpe_network_roaming_enabled",
		"Whether roaming is enabled (1) or not (0).",
		[]string{"model"}, nil,
	)
	connectModeDesc = prometheus.NewDesc(
		"zte_cpe_network_connect_mode",
		"Connection mode (0=auto, 1=manual).",
		[]string{"model"}, nil,
	)

	simReadyDesc = prometheus.NewDesc(
		"zte_cpe_sim_ready",
		"Whether SIM is ready (1) or not (0).",
		[]string{"model"}, nil,
	)
	simPinRemainingDesc = prometheus.NewDesc(
		"zte_cpe_sim_pin_remaining",
		"Remaining PIN attempts.",
		[]string{"model"}, nil,
	)
	simPukRemainingDesc = prometheus.NewDesc(
		"zte_cpe_sim_puk_remaining",
		"Remaining PUK attempts.",
		[]string{"model"}, nil,
	)

	deviceInfoDesc = prometheus.NewDesc(
		"zte_cpe_device_info",
		"Device information (always 1, labels contain info).",
		[]string{"model", "firmware", "hardware_version", "imei", "mac_address", "operator", "network_type", "iccid"}, nil,
	)

	dhcpEnabledDesc = prometheus.NewDesc(
		"zte_cpe_dhcp_enabled",
		"Whether DHCP is enabled (1) or not (0).",
		[]string{"model"}, nil,
	)
	dhcpLeaseTimeDesc = prometheus.NewDesc(
		"zte_cpe_dhcp_lease_time_seconds",
		"DHCP lease time in seconds.",
		[]string{"model"}, nil,
	)
	wanMtuDesc = prometheus.NewDesc(
		"zte_cpe_wan_mtu",
		"WAN MTU.",
		[]string{"model"}, nil,
	)
	wanMssDesc = prometheus.NewDesc(
		"zte_cpe_wan_mss",
		"WAN MSS.",
		[]string{"model"}, nil,
	)
)

// Exporter implements prometheus.Collector for ZTE CPE routers.
type Exporter struct {
	clientFn func() (router.RouterClient, error)
	model    string
	password string

	mu    sync.RWMutex
	cache *cachedData
	done  chan struct{}
}

// cachedData holds the last successfully collected data from the router.
type cachedData struct {
	status    json.RawMessage
	netInfo   json.RawMessage
	simInfo   json.RawMessage
	devInfo   json.RawMessage
	connected json.RawMessage

	dhcp *router.DhcpSettings
	mtu  *router.MtuSettings

	firmwareVersion  string
	hardwareVersion  string

	scrapeDuration float64
	scrapeSuccess  float64
}

// NewExporter creates a new ZTE CPE Prometheus exporter.
// clientFn creates a fresh router client on each call.
// model is the router model identifier (e.g. "g5ts", "mf289f").
// password is the router admin password used for login.
func NewExporter(clientFn func() (router.RouterClient, error), model, password string) *Exporter {
	return &Exporter{
		clientFn: clientFn,
		model:    model,
		password: password,
		cache:    &cachedData{},
		done:     make(chan struct{}),
	}
}

// Describe sends all possible metric descriptors to the channel.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
	ch <- signalRsrpDesc
	ch <- signalRsrqDesc
	ch <- signalSnrDesc
	ch <- signalRssiDesc
	ch <- signalBarDesc
	ch <- connectedDevicesDesc
	ch <- networkConnectedDesc
	ch <- roamingEnabledDesc
	ch <- connectModeDesc
	ch <- simReadyDesc
	ch <- simPinRemainingDesc
	ch <- simPukRemainingDesc
	ch <- deviceInfoDesc
	ch <- dhcpEnabledDesc
	ch <- dhcpLeaseTimeDesc
	ch <- wanMtuDesc
	ch <- wanMssDesc
}

// Collect reads cached data and emits Prometheus metrics.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	c := e.cache

	// Exporter self-metrics (always emitted).
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, c.scrapeDuration)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, c.scrapeSuccess)

	// Parse cached JSON into maps.
	statusMap := parseJSONMap(c.status)
	netMap := parseJSONMap(c.netInfo)
	simMap := parseJSONMap(c.simInfo)
	devMap := parseJSONMap(c.devInfo)

	// G5TS: status has nested sub-objects.
	wwanMap := getSubMap(statusMap, "wwan")
	statusDevMap := getSubMap(statusMap, "device_info")
	commonMap := getSubMap(statusMap, "common_config")

	// G5TS: device info response has nested sub-objects.
	devInfoMap := getSubMap(devMap, "device_info")
	devCommonMap := getSubMap(devMap, "common_config")

	e.collectSignalMetrics(ch, netMap, statusMap)
	e.collectConnectionMetrics(ch, wwanMap, statusMap)
	e.collectSIMMetrics(ch, simMap)
	e.collectDeviceInfoMetric(ch, devInfoMap, devCommonMap, statusDevMap, commonMap, statusMap, netMap, simMap)
	e.collectConnectedDevicesMetric(ch, c.connected)

	if c.dhcp != nil {
		dhcpVal := 0.0
		if c.dhcp.DhcpEnabled {
			dhcpVal = 1
		}
		ch <- prometheus.MustNewConstMetric(dhcpEnabledDesc, prometheus.GaugeValue, dhcpVal, e.model)
		ch <- prometheus.MustNewConstMetric(dhcpLeaseTimeDesc, prometheus.GaugeValue, float64(c.dhcp.LeaseTime), e.model)
	}

	if c.mtu != nil {
		ch <- prometheus.MustNewConstMetric(wanMtuDesc, prometheus.GaugeValue, float64(c.mtu.Mtu), e.model)
		ch <- prometheus.MustNewConstMetric(wanMssDesc, prometheus.GaugeValue, float64(c.mtu.Mss), e.model)
	}
}

// collectSignalMetrics emits RSRP, RSRQ, SNR, RSSI, and signal bar metrics.
func (e *Exporter) collectSignalMetrics(ch chan<- prometheus.Metric, netMap, statusMap map[string]interface{}) {
	networkType := firstNonEmpty(
		getString(netMap, "network_type"),
		getString(statusMap, "network_type"),
	)

	// RSRP: try nr5g_rsrp, then lte_rsrp, then status.lte_rsrp
	if v, ok := getFloat(netMap, "nr5g_rsrp"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrpDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(netMap, "lte_rsrp"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrpDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(statusMap, "lte_rsrp"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrpDesc, prometheus.GaugeValue, v, e.model, networkType)
	}

	// RSRQ: try nr5g_rsrq, then rsrq, then status.rsrq
	if v, ok := getFloat(netMap, "nr5g_rsrq"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrqDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(netMap, "rsrq"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrqDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(statusMap, "rsrq"); ok {
		ch <- prometheus.MustNewConstMetric(signalRsrqDesc, prometheus.GaugeValue, v, e.model, networkType)
	}

	// SNR: try nr5g_snr, then Z_SINR, then status.Z_SINR
	if v, ok := getFloat(netMap, "nr5g_snr"); ok {
		ch <- prometheus.MustNewConstMetric(signalSnrDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(netMap, "Z_SINR"); ok {
		ch <- prometheus.MustNewConstMetric(signalSnrDesc, prometheus.GaugeValue, v, e.model, networkType)
	} else if v, ok := getFloat(statusMap, "Z_SINR"); ok {
		ch <- prometheus.MustNewConstMetric(signalSnrDesc, prometheus.GaugeValue, v, e.model, networkType)
	}

	// RSSI
	if v, ok := getFloat(netMap, "rssi"); ok {
		ch <- prometheus.MustNewConstMetric(signalRssiDesc, prometheus.GaugeValue, v, e.model)
	} else if v, ok := getFloat(statusMap, "rssi"); ok {
		ch <- prometheus.MustNewConstMetric(signalRssiDesc, prometheus.GaugeValue, v, e.model)
	}

	// Signal bar
	if v, ok := getFloat(netMap, "signalbar"); ok {
		ch <- prometheus.MustNewConstMetric(signalBarDesc, prometheus.GaugeValue, v, e.model)
	} else if v, ok := getFloat(statusMap, "signalbar"); ok {
		ch <- prometheus.MustNewConstMetric(signalBarDesc, prometheus.GaugeValue, v, e.model)
	}
}

// collectConnectionMetrics emits connected, roaming, and connection mode metrics.
func (e *Exporter) collectConnectionMetrics(ch chan<- prometheus.Metric, wwanMap, statusMap map[string]interface{}) {
	// Connected: G5TS wwan.connect_status, MF289F ppp_status
	if v := getString(wwanMap, "connect_status"); v != "" {
		ch <- prometheus.MustNewConstMetric(networkConnectedDesc, prometheus.GaugeValue, boolToFloat(isConnectedString(v)), e.model)
	} else if v := getString(statusMap, "ppp_status"); v != "" {
		ch <- prometheus.MustNewConstMetric(networkConnectedDesc, prometheus.GaugeValue, boolToFloat(isConnectedString(v)), e.model)
	}

	// Roaming: wwan.roam_enable
	if v, ok := getFloat(wwanMap, "roam_enable"); ok {
		ch <- prometheus.MustNewConstMetric(roamingEnabledDesc, prometheus.GaugeValue, v, e.model)
	}

	// Connect mode: wwan.connect_mode
	if v, ok := getFloat(wwanMap, "connect_mode"); ok {
		ch <- prometheus.MustNewConstMetric(connectModeDesc, prometheus.GaugeValue, v, e.model)
	}
}

// collectSIMMetrics emits SIM status metrics.
func (e *Exporter) collectSIMMetrics(ch chan<- prometheus.Metric, simMap map[string]interface{}) {
	if simMap == nil {
		return
	}

	if v := getString(simMap, "sim_status"); v != "" {
		ready := 0.0
		if strings.EqualFold(v, "ready") {
			ready = 1
		}
		ch <- prometheus.MustNewConstMetric(simReadyDesc, prometheus.GaugeValue, ready, e.model)
	}

	if v, ok := getFloat(simMap, "pin_remaining"); ok {
		ch <- prometheus.MustNewConstMetric(simPinRemainingDesc, prometheus.GaugeValue, v, e.model)
	}

	if v, ok := getFloat(simMap, "puk_remaining"); ok {
		ch <- prometheus.MustNewConstMetric(simPukRemainingDesc, prometheus.GaugeValue, v, e.model)
	}
}

// collectDeviceInfoMetric emits the zte_cpe_device_info metric with all label values.
func (e *Exporter) collectDeviceInfoMetric(
	ch chan<- prometheus.Metric,
	devInfoMap, devCommonMap, statusDevMap, commonMap, statusMap, netMap, simMap map[string]interface{},
) {
	firmware := e.cache.firmwareVersion
	hwVersion := e.cache.hardwareVersion

	// IMEI: devInfo.device_info.IMEI, status.device_info.IMEI, status.imei
	imei := firstNonEmpty(
		getString(devInfoMap, "IMEI"),
		getString(statusDevMap, "IMEI"),
		getString(statusMap, "imei"),
	)

	// MAC: devInfo.device_info.macaddr, status.device_info.macaddr
	mac := firstNonEmpty(
		getString(devInfoMap, "macaddr"),
		getString(statusDevMap, "macaddr"),
	)

	// Operator: netInfo.network_provider, status.network_provider
	operator := firstNonEmpty(
		getString(netMap, "network_provider"),
		getString(statusMap, "network_provider"),
	)

	// Network type: common_config, devInfo.common_config, netInfo, status
	networkType := firstNonEmpty(
		getString(devCommonMap, "network_type"),
		getString(commonMap, "network_type"),
		getString(netMap, "network_type"),
		getString(statusMap, "network_type"),
	)

	// ICCID: common_config, devInfo.common_config, simInfo
	iccid := firstNonEmpty(
		getString(devCommonMap, "iccid"),
		getString(commonMap, "iccid"),
		getString(simMap, "iccid"),
	)

	ch <- prometheus.MustNewConstMetric(deviceInfoDesc, prometheus.GaugeValue, 1,
		e.model, firmware, hwVersion, imei, mac, operator, networkType, iccid,
	)
}

// collectConnectedDevicesMetric emits the connected devices count.
func (e *Exporter) collectConnectedDevicesMetric(ch chan<- prometheus.Metric, data json.RawMessage) {
	if data == nil {
		return
	}

	count := countDevices(data)
	ch <- prometheus.MustNewConstMetric(connectedDevicesDesc, prometheus.GaugeValue, count, e.model)
}

// Start begins the background collection goroutine.
func (e *Exporter) Start(interval time.Duration) {
	go func() {
		// Initial collection immediately.
		e.collectOnce()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				e.collectOnce()
			case <-e.done:
				return
			}
		}
	}()
}

// Stop signals the background goroutine to stop.
func (e *Exporter) Stop() {
	close(e.done)
}

// collectOnce performs a single collection cycle: login, gather data, logout.
func (e *Exporter) collectOnce() {
	start := time.Now()

	client, err := e.clientFn()
	if err != nil {
		log.Printf("exporter: failed to create client: %v", err)
		e.mu.Lock()
		e.cache.scrapeSuccess = 0
		e.cache.scrapeDuration = time.Since(start).Seconds()
		e.mu.Unlock()
		return
	}

	ctx := context.Background()
	if err := client.Login(ctx, e.password); err != nil {
		log.Printf("exporter: login failed: %v", err)
		e.mu.Lock()
		e.cache.scrapeSuccess = 0
		e.cache.scrapeDuration = time.Since(start).Seconds()
		e.mu.Unlock()
		return
	}
	defer func() {
		if err := client.Logout(ctx); err != nil {
			log.Printf("exporter: logout failed: %v", err)
		}
	}()

	newCache := &cachedData{}

	if status, err := client.GetStatus(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetStatus failed: %v", err)
		}
	} else {
		newCache.status = status
	}

	if netInfo, err := client.GetNetworkInfo(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetNetworkInfo failed: %v", err)
		}
	} else {
		newCache.netInfo = netInfo
	}

	if simInfo, err := client.GetSIMInfo(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetSIMInfo failed: %v", err)
		}
	} else {
		newCache.simInfo = simInfo
	}

	if devInfo, err := client.GetDeviceInfo(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetDeviceInfo failed: %v", err)
		}
	} else {
		newCache.devInfo = devInfo
	}

	if connected, err := client.GetConnectedDevices(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetConnectedDevices failed: %v", err)
		}
	} else {
		newCache.connected = connected
	}

	if dhcp, err := client.GetDHCPSettings(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetDHCPSettings failed: %v", err)
		}
	} else {
		newCache.dhcp = dhcp
	}

	if mtu, err := client.GetMTUSettings(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetMTUSettings failed: %v", err)
		}
	} else {
		newCache.mtu = mtu
	}

	if hw, fw, err := client.GetVersion(ctx); err != nil {
		if !isNotSupported(err) {
			log.Printf("exporter: GetVersion failed: %v", err)
		}
	} else {
		newCache.hardwareVersion = hw
		newCache.firmwareVersion = fw
	}

	newCache.scrapeDuration = time.Since(start).Seconds()
	newCache.scrapeSuccess = 1

	e.mu.Lock()
	e.cache = newCache
	e.mu.Unlock()
}

// --- Helper functions ---

func isNotSupported(err error) bool {
	var ns *router.ErrNotSupported
	return errors.As(err, &ns)
}

func parseJSONMap(data json.RawMessage) map[string]interface{} {
	if data == nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}

func getSubMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	sub, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return sub
}

func getFloat(m map[string]interface{}, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return f, true
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}

func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func isConnectedString(s string) bool {
	return strings.Contains(strings.ToLower(s), "connected")
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func countDevices(data json.RawMessage) float64 {
	// Try as JSON array directly.
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err == nil {
		return float64(len(arr))
	}

	// Try as object with a "devices" array.
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		if devArr, ok := obj["devices"]; ok {
			if b, err := json.Marshal(devArr); err == nil {
				var devices []json.RawMessage
				if err := json.Unmarshal(b, &devices); err == nil {
					return float64(len(devices))
				}
			}
		}
	}

	return 0
}
