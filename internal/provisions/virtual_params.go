package provisions

import (
	"strconv"
	"strings"
	"time"
)

// VirtualParameterEngine handles virtual parameter extraction
type VirtualParameterEngine struct {
	Parameters map[string]*VirtualParameterDef
}

// VirtualParameterDef defines a virtual parameter
type VirtualParameterDef struct {
	Name      string
	ExtractFn func(params map[string]string, manufacturer string) string
}

// NewVirtualParameterEngine creates a new virtual parameter engine
func NewVirtualParameterEngine() *VirtualParameterEngine {
	engine := &VirtualParameterEngine{
		Parameters: make(map[string]*VirtualParameterDef),
	}

	// Register default virtual parameters
	engine.registerDefaults()

	return engine
}

func (e *VirtualParameterEngine) registerDefaults() {
	// RX Power - supports multiple vendors
	e.Parameters["RXPower"] = &VirtualParameterDef{
		Name: "RXPower",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_HW_GponInterfaceConfig.RXPower",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return formatRXPower(v)
				}
			}
			return ""
		},
	}

	// TX Power
	e.Parameters["TXPower"] = &VirtualParameterDef{
		Name: "TXPower",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.TXPower",
				"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.TXPower",
				"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.TXPower",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return formatRXPower(v)
				}
			}
			return ""
		},
	}

	// Temperature
	e.Parameters["Temperature"] = &VirtualParameterDef{
		Name: "Temperature",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.Temperature",
				"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.Temperature",
				"InternetGatewayDevice.DeviceInfo.X_HW_Temperature",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return v
				}
			}
			return ""
		},
	}

	// PPPoE Username
	e.Parameters["pppoeUsername"] = &VirtualParameterDef{
		Name: "pppoeUsername",
		ExtractFn: func(params map[string]string, mfr string) string {
			// Search for any PPP username
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".Username") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE Password
	e.Parameters["pppoePassword"] = &VirtualParameterDef{
		Name: "pppoePassword",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".Password") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE External IP
	e.Parameters["pppoeIP"] = &VirtualParameterDef{
		Name: "pppoeIP",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".ExternalIPAddress") {
					if val != "" && val != "0.0.0.0" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE MAC Address
	e.Parameters["pppoeMac"] = &VirtualParameterDef{
		Name: "pppoeMac",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".MACAddress") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PON MAC Address
	e.Parameters["PonMac"] = &VirtualParameterDef{
		Name: "PonMac",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.MACAddress",
				"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.MACAddress",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return v
				}
			}
			return ""
		},
	}

	// Serial Number
	e.Parameters["getSerialNumber"] = &VirtualParameterDef{
		Name: "getSerialNumber",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.DeviceInfo.SerialNumber",
				"InternetGatewayDevice.DeviceInfo.X_HW_SerialNumber",
				"DeviceID.SerialNumber",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return v
				}
			}
			return ""
		},
	}

	// Device Uptime
	e.Parameters["getdeviceuptime"] = &VirtualParameterDef{
		Name: "getdeviceuptime",
		ExtractFn: func(params map[string]string, mfr string) string {
			if v, ok := params["InternetGatewayDevice.DeviceInfo.UpTime"]; ok {
				return formatUptime(v)
			}
			return ""
		},
	}

	// PPPoE Uptime
	e.Parameters["getpppuptime"] = &VirtualParameterDef{
		Name: "getpppuptime",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".Uptime") {
					if val != "" {
						return formatUptime(val)
					}
				}
			}
			return ""
		},
	}

	// Primary SSID
	e.Parameters["SSID"] = &VirtualParameterDef{
		Name: "SSID",
		ExtractFn: func(params map[string]string, mfr string) string {
			path := "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"
			if v, ok := params[path]; ok {
				return v
			}
			return ""
		},
	}

	// WiFi Password
	e.Parameters["WlanPassword"] = &VirtualParameterDef{
		Name: "WlanPassword",
		ExtractFn: func(params map[string]string, mfr string) string {
			paths := []string{
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase",
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase",
			}
			for _, p := range paths {
				if v, ok := params[p]; ok && v != "" {
					return v
				}
			}
			return ""
		},
	}

	// PON Mode (GPON/EPON)
	e.Parameters["getponmode"] = &VirtualParameterDef{
		Name: "getponmode",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key := range params {
				if strings.Contains(key, "GponInterfaceConfig") {
					return "GPON"
				}
				if strings.Contains(key, "EponInterfaceConfig") {
					return "EPON"
				}
			}
			return "Unknown"
		},
	}

	// Active Devices Count
	e.Parameters["activedevices"] = &VirtualParameterDef{
		Name: "activedevices",
		ExtractFn: func(params map[string]string, mfr string) string {
			count := 0
			for key := range params {
				if strings.Contains(key, "Hosts.Host.") && strings.HasSuffix(key, ".IPAddress") {
					count++
				}
			}
			return strconv.Itoa(count)
		},
	}

	// TR-069 Client IP
	e.Parameters["IPTR069"] = &VirtualParameterDef{
		Name: "IPTR069",
		ExtractFn: func(params map[string]string, mfr string) string {
			if v, ok := params["InternetGatewayDevice.ManagementServer.ConnectionRequestURL"]; ok {
				// Extract IP from URL
				parts := strings.Split(v, "//")
				if len(parts) > 1 {
					hostPart := strings.Split(parts[1], ":")[0]
					return hostPart
				}
			}
			return ""
		},
	}

	// PPPoE Connection Status
	e.Parameters["pppoeStatus"] = &VirtualParameterDef{
		Name: "pppoeStatus",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".ConnectionStatus") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE VLAN ID
	e.Parameters["pppoeVLAN"] = &VirtualParameterDef{
		Name: "pppoeVLAN",
		ExtractFn: func(params map[string]string, mfr string) string {
			// Vendor-specific VLAN paths
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") {
					if strings.HasSuffix(key, ".X_HW_VLAN") || // Huawei
						strings.HasSuffix(key, ".X_ZTE-COM_VLANID") || // ZTE
						strings.HasSuffix(key, ".X_FH_VLAN") || // FiberHome
						strings.HasSuffix(key, ".X_ALU_VLANID") || // Nokia/Alcatel
						strings.HasSuffix(key, "VLANID") {
						if val != "" && val != "0" && val != "-1" {
							return val
						}
					}
				}
			}
			return ""
		},
	}

	// PPPoE Gateway
	e.Parameters["pppoeGateway"] = &VirtualParameterDef{
		Name: "pppoeGateway",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") || strings.Contains(key, "WANIPConnection") {
					if strings.HasSuffix(key, ".DefaultGateway") || strings.HasSuffix(key, ".RemoteIPAddress") {
						if val != "" && val != "0.0.0.0" {
							return val
						}
					}
				}
			}
			return ""
		},
	}

	// PPPoE DNS Servers
	e.Parameters["pppoeDNS"] = &VirtualParameterDef{
		Name: "pppoeDNS",
		ExtractFn: func(params map[string]string, mfr string) string {
			var dns []string
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") || strings.Contains(key, "WANIPConnection") {
					if strings.Contains(key, "DNSServers") || strings.HasSuffix(key, ".DNS") {
						if val != "" {
							dns = append(dns, val)
						}
					}
				}
			}
			if len(dns) > 0 {
				return strings.Join(dns, ", ")
			}
			return ""
		},
	}

	// PPPoE Connection Type (Router/Bridge)
	e.Parameters["pppoeConnType"] = &VirtualParameterDef{
		Name: "pppoeConnType",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".ConnectionType") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// NAT Enabled
	e.Parameters["pppoeNAT"] = &VirtualParameterDef{
		Name: "pppoeNAT",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".NATEnabled") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE Service Name
	e.Parameters["pppoeServiceName"] = &VirtualParameterDef{
		Name: "pppoeServiceName",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") {
					if strings.HasSuffix(key, ".X_HW_SERVICELIST") || // Huawei
						strings.HasSuffix(key, ".X_ZTE-COM_ServiceList") || // ZTE
						strings.HasSuffix(key, ".Name") {
						if val != "" {
							return val
						}
					}
				}
			}
			return ""
		},
	}

	// PPPoE MTU
	e.Parameters["pppoeMTU"] = &VirtualParameterDef{
		Name: "pppoeMTU",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".MaxMRUSize") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// PPPoE Enable Status
	e.Parameters["pppoeEnable"] = &VirtualParameterDef{
		Name: "pppoeEnable",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".Enable") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// LAN Binding (Huawei)
	e.Parameters["pppoeLanBind"] = &VirtualParameterDef{
		Name: "pppoeLanBind",
		ExtractFn: func(params map[string]string, mfr string) string {
			var bindings []string
			lanPorts := map[string]string{
				"Lan1Enable":  "LAN1",
				"Lan2Enable":  "LAN2",
				"Lan3Enable":  "LAN3",
				"Lan4Enable":  "LAN4",
				"SSID1Enable": "WiFi1",
				"SSID2Enable": "WiFi2",
			}
			for key, val := range params {
				if strings.Contains(key, "X_HW_LANBIND") || strings.Contains(key, "X_ZTE-COM_LanBind") {
					for suffix, name := range lanPorts {
						if strings.HasSuffix(key, suffix) && (val == "1" || val == "true") {
							bindings = append(bindings, name)
						}
					}
				}
			}
			if len(bindings) > 0 {
				return strings.Join(bindings, ", ")
			}
			return ""
		},
	}

	// WAN Connection Name
	e.Parameters["pppoeConnName"] = &VirtualParameterDef{
		Name: "pppoeConnName",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".Name") {
					if val != "" {
						return val
					}
				}
			}
			return ""
		},
	}

	// Last Connection Error
	e.Parameters["pppoeLastError"] = &VirtualParameterDef{
		Name: "pppoeLastError",
		ExtractFn: func(params map[string]string, mfr string) string {
			for key, val := range params {
				if strings.Contains(key, "WANPPPConnection") && strings.HasSuffix(key, ".LastConnectionError") {
					if val != "" && val != "ERROR_NONE" {
						return val
					}
				}
			}
			return ""
		},
	}
}

// GetValue extracts a virtual parameter value
func (e *VirtualParameterEngine) GetValue(name string, params map[string]string, manufacturer string) string {
	if vp, ok := e.Parameters[name]; ok {
		return vp.ExtractFn(params, manufacturer)
	}
	return ""
}

// GetAllValues extracts all virtual parameter values
func (e *VirtualParameterEngine) GetAllValues(params map[string]string, manufacturer string) map[string]string {
	result := make(map[string]string)
	for name, vp := range e.Parameters {
		val := vp.ExtractFn(params, manufacturer)
		if val != "" {
			result[name] = val
		}
	}
	return result
}

// formatRXPower formats RX power value to dBm
func formatRXPower(value string) string {
	// Some devices return values in different formats
	// e.g., "2048" needs to be converted to dBm
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}

	// If value is large, it might need conversion
	if v > 100 || v < -100 {
		// Convert from internal format to dBm
		// Common formula: dBm = (value - 10000) / 100
		if v > 1000 {
			dbm := (v - 10000) / 100
			return strconv.FormatFloat(dbm, 'f', 2, 64) + " dBm"
		}
	}

	return strconv.FormatFloat(v, 'f', 2, 64) + " dBm"
}

// formatUptime formats uptime seconds to human readable
func formatUptime(value string) string {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return value
	}

	duration := time.Duration(seconds) * time.Second

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m"
	} else if hours > 0 {
		return strconv.Itoa(hours) + "h " + strconv.Itoa(minutes) + "m"
	}
	return strconv.Itoa(minutes) + "m"
}
