package tr069

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"go-acs/internal/models"
)

// DeviceParameterParser handles parsing of TR-069 parameters for ONU devices
type DeviceParameterParser struct {
	device   *models.Device
	vendor   string
	model    string
}

// NewDeviceParameterParser creates a new parameter parser
func NewDeviceParameterParser(device *models.Device, vendor, model string) *DeviceParameterParser {
	return &DeviceParameterParser{
		device: device,
		vendor: strings.ToUpper(vendor),
		model:  model,
	}
}

// ParseParameter parses a single TR-069 parameter and updates device fields
func (p *DeviceParameterParser) ParseParameter(paramName, paramValue string) {
	if paramValue == "" || paramValue == "null" || paramValue == "default" {
		return
	}

	// Normalize parameter name
	paramName = strings.TrimSpace(paramName)
	paramValue = strings.TrimSpace(paramValue)

	// Parse based on parameter category
	p.parseDeviceInfo(paramName, paramValue)
	p.parseOpticalParameters(paramName, paramValue)
	p.parseWANParameters(paramName, paramValue)
	p.parseWiFiParameters(paramName, paramValue)
	p.parseEthernetParameters(paramName, paramValue)
	p.parseTemperatureParameters(paramName, paramValue)
	p.parseClientCountParameters(paramName, paramValue)
	p.parseVendorSpecificParameters(paramName, paramValue)
}

// parseDeviceInfo parses device information parameters
func (p *DeviceParameterParser) parseDeviceInfo(paramName, paramValue string) {
	switch paramName {
	case "Device.DeviceInfo.SoftwareVersion",
		"InternetGatewayDevice.DeviceInfo.SoftwareVersion":
		p.device.SoftwareVersion = paramValue
	case "Device.DeviceInfo.HardwareVersion",
		"InternetGatewayDevice.DeviceInfo.HardwareVersion":
		p.device.HardwareVersion = paramValue
	case "Device.DeviceInfo.ModelName",
		"InternetGatewayDevice.DeviceInfo.ModelName":
		p.device.ModelName = paramValue
	case "Device.DeviceInfo.Description",
		"InternetGatewayDevice.DeviceInfo.Description":
		if len(paramValue) < 50 && !strings.Contains(paramValue, "[]") {
			p.device.Template = paramValue
		}
	case "Device.DeviceInfo.UpTime",
		"InternetGatewayDevice.DeviceInfo.UpTime":
		if uptime, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
			p.device.Uptime = uptime
		}
	case "Device.ManagementServer.ConnectionRequestURL",
		"InternetGatewayDevice.ManagementServer.ConnectionRequestURL":
		p.device.ConnectionRequest = paramValue
	}
}

// parseOpticalParameters parses optical interface parameters
func (p *DeviceParameterParser) parseOpticalParameters(paramName, paramValue string) {
	// Check for optical power parameters
	if strings.Contains(paramName, "RXPower") || strings.Contains(paramName, "RxPower") || 
	   strings.Contains(paramName, "OpticalSignalLevel") {
		if power, err := p.parseOpticalPower(paramValue); err == nil {
			p.device.RXPower = power
		}
	}
	
	if strings.Contains(paramName, "TXPower") || strings.Contains(paramName, "TxPower") || 
	   strings.Contains(paramName, "TransmitOpticalLevel") {
		if power, err := p.parseOpticalPower(paramValue); err == nil {
			// Store TX power in device parameters or create separate field
			if p.device.Parameters == nil {
				p.device.Parameters = make(map[string]string)
			}
			p.device.Parameters["tx_power"] = fmt.Sprintf("%.2f", power)
		}
	}
	
	// GPON specific parameters
	if strings.Contains(paramName, "ONTID") || strings.Contains(paramName, "ONT_ID") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["ont_id"] = paramValue
	}
	
	if strings.Contains(paramName, "EquipmentID") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["equipment_id"] = paramValue
	}
}

// parseWANParameters parses WAN connection parameters
func (p *DeviceParameterParser) parseWANParameters(paramName, paramValue string) {
	// IP Address extraction
	if strings.HasSuffix(paramName, "ExternalIPAddress") ||
		strings.HasSuffix(paramName, "IPv4Address.1.IPAddress") {
		if paramValue != "" && paramValue != "0.0.0.0" {
			p.device.WANIP = paramValue
		}
	}
	
	// Connection type
	if strings.Contains(paramName, "ConnectionType") && strings.Contains(paramName, "WAN") {
		p.device.WANConnectionType = paramValue
	}
	
	// PPPoE Username extraction
	if (strings.Contains(paramName, "WANPPPConnection") && strings.HasSuffix(paramName, "Username")) ||
		strings.HasSuffix(paramName, "X_CT-COM_UserInfo.UserName") ||
		strings.HasSuffix(paramName, "X_CMCC_UserInfo.UserName") {
		if paramValue != "" && paramValue != "default" && paramValue != "null" {
			p.device.PPPoEUsername = paramValue
			p.device.Template = paramValue // Also set template for display
		}
	}
	
	// PPP connection status
	if strings.Contains(paramName, "PPP") && strings.HasSuffix(paramName, "ConnectionStatus") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["ppp_connection_status"] = paramValue
	}
	
	// PPP connect time
	if strings.Contains(paramName, "PPP") && strings.HasSuffix(paramName, "ConnectTime") {
		if connectTime, err := strconv.ParseInt(paramValue, 10, 64); err == nil {
			if p.device.Parameters == nil {
				p.device.Parameters = make(map[string]string)
			}
			p.device.Parameters["ppp_connect_time"] = fmt.Sprintf("%d", connectTime)
		}
	}
}

// parseWiFiParameters parses WiFi parameters
func (p *DeviceParameterParser) parseWiFiParameters(paramName, paramValue string) {
	// Client count from various sources
	if strings.Contains(paramName, "WLANConfiguration") &&
		(strings.HasSuffix(paramName, "TotalAssociations") ||
			strings.HasSuffix(paramName, "WLAN_AssociatedDeviceNumberOfEntries") ||
			strings.HasSuffix(paramName, "AssociatedDeviceNumberOfEntities")) {
		if clientCount, err := strconv.Atoi(paramValue); err == nil {
			p.device.ClientCount += clientCount
		}
	}
	
	// Global host count (usually more accurate)
	if strings.HasSuffix(paramName, "HostNumberOfEntries") {
		if hostCount, err := strconv.Atoi(paramValue); err == nil && hostCount > 0 {
			p.device.ClientCount = hostCount
		}
	}
	
	// WiFi SSID
	if strings.Contains(paramName, "WiFi") && strings.HasSuffix(paramName, "SSID") && 
		!strings.Contains(paramName, "Enable") && !strings.Contains(paramName, "Status") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["wifi_ssid"] = paramValue
	}
	
	// WiFi security mode
	if strings.Contains(paramName, "WiFi") && strings.Contains(paramName, "Security") && 
		strings.HasSuffix(paramName, "ModeEnabled") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["wifi_security"] = paramValue
	}
}

// parseEthernetParameters parses Ethernet interface parameters
func (p *DeviceParameterParser) parseEthernetParameters(paramName, paramValue string) {
	if strings.Contains(paramName, "Ethernet") && strings.HasSuffix(paramName, "MACAddress") {
		if p.device.MACAddress == "" && paramValue != "" {
			p.device.MACAddress = paramValue
		}
	}
	
	if strings.Contains(paramName, "Ethernet") && strings.HasSuffix(paramName, "DuplexMode") {
		if p.device.Parameters == nil {
			p.device.Parameters = make(map[string]string)
		}
		p.device.Parameters["ethernet_duplex"] = paramValue
	}
}

// parseTemperatureParameters parses temperature parameters
func (p *DeviceParameterParser) parseTemperatureParameters(paramName, paramValue string) {
	if strings.Contains(strings.ToLower(paramName), "temperature") {
		if temp, err := strconv.ParseFloat(paramValue, 64); err == nil {
			// Apply conversion logic based on value range
			if temp > 1000 {
				p.device.Temperature = temp / 256.0
			} else if temp > 100 {
				p.device.Temperature = temp / 10.0
			} else {
				p.device.Temperature = temp
			}
		}
	}
}

// parseClientCountParameters parses client count parameters
func (p *DeviceParameterParser) parseClientCountParameters(paramName, paramValue string) {
	// This is handled in WiFi parsing, but can be extended here
}

// parseVendorSpecificParameters parses vendor-specific parameters
func (p *DeviceParameterParser) parseVendorSpecificParameters(paramName, paramValue string) {
	// Handle vendor-specific parameters based on vendor name
	switch p.vendor {
	case "ZTE":
		p.parseZTEParameters(paramName, paramValue)
	case "HUAWEI":
		p.parseHuaweiParameters(paramName, paramValue)
	case "FIBERHOME":
		p.parseFiberhomeParameters(paramName, paramValue)
	case "NOKIA", "ALU":
		p.parseNokiaParameters(paramName, paramValue)
	}
}

// parseZTEParameters parses ZTE-specific parameters
func (p *DeviceParameterParser) parseZTEParameters(paramName, paramValue string) {
	if strings.Contains(paramName, "X_ZTE-COM") {
		if strings.Contains(paramName, "RXPower") {
			if power, err := p.parseZTEOpticalPower(paramValue); err == nil {
				p.device.RXPower = power
			}
		}
		if strings.Contains(paramName, "TXPower") {
			if power, err := p.parseZTEOpticalPower(paramValue); err == nil {
				if p.device.Parameters == nil {
					p.device.Parameters = make(map[string]string)
				}
				p.device.Parameters["tx_power"] = fmt.Sprintf("%.2f", power)
			}
		}
		if strings.Contains(paramName, "Temperature") {
			if temp, err := strconv.ParseFloat(paramValue, 64); err == nil {
				p.device.Temperature = temp / 256.0
			}
		}
	}
}

// parseHuaweiParameters parses Huawei-specific parameters
func (p *DeviceParameterParser) parseHuaweiParameters(paramName, paramValue string) {
	if strings.Contains(paramName, "X_GponInterafceConfig") {
		if strings.Contains(paramName, "RXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				p.device.RXPower = power
			}
		}
		if strings.Contains(paramName, "TXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				if p.device.Parameters == nil {
					p.device.Parameters = make(map[string]string)
				}
				p.device.Parameters["tx_power"] = fmt.Sprintf("%.2f", power)
			}
		}
	}
}

// parseFiberhomeParameters parses Fiberhome-specific parameters
func (p *DeviceParameterParser) parseFiberhomeParameters(paramName, paramValue string) {
	if strings.Contains(paramName, "X_FH_GponInterfaceConfig") {
		if strings.Contains(paramName, "RXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				p.device.RXPower = power
			}
		}
		if strings.Contains(paramName, "TXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				if p.device.Parameters == nil {
					p.device.Parameters = make(map[string]string)
				}
				p.device.Parameters["tx_power"] = fmt.Sprintf("%.2f", power)
			}
		}
	}
}

// parseNokiaParameters parses Nokia/ALU-specific parameters
func (p *DeviceParameterParser) parseNokiaParameters(paramName, paramValue string) {
	if strings.Contains(paramName, "X_ALU_OntOpticalParam") {
		if strings.Contains(paramName, "RXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				p.device.RXPower = power
			}
		}
		if strings.Contains(paramName, "TXPower") {
			if power, err := strconv.ParseFloat(paramValue, 64); err == nil {
				if p.device.Parameters == nil {
					p.device.Parameters = make(map[string]string)
				}
				p.device.Parameters["tx_power"] = fmt.Sprintf("%.2f", power)
			}
		}
	}
}

// parseOpticalPower parses optical power values with vendor-specific logic
func (p *DeviceParameterParser) parseOpticalPower(rawValue string) (float64, error) {
	if power, err := strconv.ParseFloat(rawValue, 64); err == nil {
		// If already negative, it's likely already in dBm
		if power < 0 {
			return math.Round(power*100) / 100, nil
		}
		
		// Apply vendor-specific conversion
		switch p.vendor {
		case "ZTE", "CIOT", "CT-COM", "CMCC":
			// Formula: (10 * log10(power)) - 40
			dbm := (10 * math.Log10(power)) - 40
			return math.Round(dbm*100) / 100, nil
		case "HUAWEI", "FIBERHOME":
			// Direct conversion for some models
			return math.Round(power*100) / 100, nil
		default:
			// Default conversion
			dbm := (10 * math.Log10(power)) - 40
			return math.Round(dbm*100) / 100, nil
		}
	}
	
	return 0, fmt.Errorf("invalid optical power value: %s", rawValue)
}

// parseZTEOpticalPower parses ZTE-specific optical power format
func (p *DeviceParameterParser) parseZTEOpticalPower(rawValue string) (float64, error) {
	if power, err := strconv.ParseFloat(rawValue, 64); err == nil {
		// ZTE formula: (10 * log10(power)) - 40
		dbm := (10 * math.Log10(power)) - 40
		return math.Round(dbm*100) / 100, nil
	}
	return 0, fmt.Errorf("invalid ZTE optical power value: %s", rawValue)
}

// GetDeviceData returns the parsed device data
func (p *DeviceParameterParser) GetDeviceData() *models.Device {
	return p.device
}