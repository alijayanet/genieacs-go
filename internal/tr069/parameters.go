package tr069

import (
	"fmt"
	"math"
	"strconv"
)

// TR069Parameter represents a TR-069 parameter with its path and metadata
type TR069Parameter struct {
	Path        string      `json:"path"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	DataType    string      `json:"dataType"`
	Writable    bool        `json:"writable"`
	Vendor      string      `json:"vendor,omitempty"`
	Model       string      `json:"model,omitempty"`
}

// ParameterCategory groups related parameters
type ParameterCategory struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  []TR069Parameter  `json:"parameters"`
}

// GetStandardONUParameters returns comprehensive TR-069 parameters for ONU devices
func GetStandardONUParameters() []ParameterCategory {
	return []ParameterCategory{
		{
			Name:        "Device Information",
			Description: "Basic device identification and software information",
			Parameters: []TR069Parameter{
				{Path: "Device.DeviceInfo.Manufacturer", Name: "Manufacturer", Description: "Device manufacturer", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.ManufacturerOUI", Name: "Manufacturer OUI", Description: "Organizationally unique identifier", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.ModelName", Name: "Model Name", Description: "Device model name", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.Description", Name: "Description", Description: "Device description", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.ProductClass", Name: "Product Class", Description: "Product class identifier", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.SerialNumber", Name: "Serial Number", Description: "Device serial number", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.HardwareVersion", Name: "Hardware Version", Description: "Hardware version", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.SoftwareVersion", Name: "Software Version", Description: "Software version", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.ModemFirmwareVersion", Name: "Modem Firmware", Description: "Modem firmware version", DataType: "string", Writable: false},
				{Path: "Device.DeviceInfo.UpTime", Name: "Uptime", Description: "Device uptime in seconds", DataType: "unsignedInt", Writable: false},
				{Path: "Device.DeviceInfo.FirstUseDate", Name: "First Use Date", Description: "Date when device was first used", DataType: "dateTime", Writable: false},
				{Path: "Device.DeviceInfo.ProvisioningCode", Name: "Provisioning Code", Description: "Service provider provisioning code", DataType: "string", Writable: true},
			},
		},
		{
			Name:        "Optical Interface (PON)",
			Description: "PON optical interface parameters and statistics",
			Parameters: []TR069Parameter{
				{Path: "Device.Optical.Interface.1.Enable", Name: "Optical Enable", Description: "Enable/disable optical interface", DataType: "boolean", Writable: true},
				{Path: "Device.Optical.Interface.1.Status", Name: "Optical Status", Description: "Current optical interface status", DataType: "string", Writable: false},
				{Path: "Device.Optical.Interface.1.Alias", Name: "Optical Alias", Description: "Interface alias", DataType: "string", Writable: true},
				{Path: "Device.Optical.Interface.1.Name", Name: "Optical Name", Description: "Interface name", DataType: "string", Writable: false},
				{Path: "Device.Optical.Interface.1.LastChange", Name: "Last Change", Description: "Last interface change timestamp", DataType: "unsignedInt", Writable: false},
				{Path: "Device.Optical.Interface.1.LowerLayers", Name: "Lower Layers", Description: "Lower layer interfaces", DataType: "string", Writable: true},
				{Path: "Device.Optical.Interface.1.Upstream", Name: "Upstream", Description: "Upstream direction", DataType: "boolean", Writable: false},
				{Path: "Device.Optical.Interface.1.OpticalSignalLevel", Name: "Optical Signal Level", Description: "Received optical signal level in dBm", DataType: "int", Writable: false},
				{Path: "Device.Optical.Interface.1.LowerOpticalThreshold", Name: "Lower Optical Threshold", Description: "Lower optical threshold in dBm", DataType: "int", Writable: true},
				{Path: "Device.Optical.Interface.1.UpperOpticalThreshold", Name: "Upper Optical Threshold", Description: "Upper optical threshold in dBm", DataType: "int", Writable: true},
				{Path: "Device.Optical.Interface.1.TransmitOpticalLevel", Name: "Transmit Optical Level", Description: "Transmit optical level in dBm", DataType: "int", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.BytesSent", Name: "Bytes Sent", Description: "Total bytes sent", DataType: "unsignedLong", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.BytesReceived", Name: "Bytes Received", Description: "Total bytes received", DataType: "unsignedLong", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.PacketsSent", Name: "Packets Sent", Description: "Total packets sent", DataType: "unsignedLong", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.PacketsReceived", Name: "Packets Received", Description: "Total packets received", DataType: "unsignedLong", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.ErrorsSent", Name: "Errors Sent", Description: "Total send errors", DataType: "unsignedLong", Writable: false},
				{Path: "Device.Optical.Interface.1.Stats.ErrorsReceived", Name: "Errors Received", Description: "Total receive errors", DataType: "unsignedLong", Writable: false},
			},
		},
		{
			Name:        "GPON Interface",
			Description: "GPON-specific parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.GPON.Interface.1.Enable", Name: "GPON Enable", Description: "Enable GPON interface", DataType: "boolean", Writable: true},
				{Path: "Device.GPON.Interface.1.Status", Name: "GPON Status", Description: "GPON interface status", DataType: "string", Writable: false},
				{Path: "Device.GPON.Interface.1.SerialNumber", Name: "GPON Serial", Description: "GPON serial number", DataType: "string", Writable: false},
				{Path: "Device.GPON.Interface.1.RXPower", Name: "GPON RX Power", Description: "GPON receive power in dBm", DataType: "int", Writable: false},
				{Path: "Device.GPON.Interface.1.TXPower", Name: "GPON TX Power", Description: "GPON transmit power in dBm", DataType: "int", Writable: false},
				{Path: "Device.GPON.Interface.1.Temperature", Name: "GPON Temperature", Description: "GPON transceiver temperature", DataType: "int", Writable: false},
				{Path: "Device.GPON.Interface.1.Voltage", Name: "GPON Voltage", Description: "GPON transceiver voltage", DataType: "int", Writable: false},
				{Path: "Device.GPON.Interface.1.BiasCurrent", Name: "GPON Bias Current", Description: "GPON laser bias current", DataType: "int", Writable: false},
				{Path: "Device.GPON.Interface.1.ONTID", Name: "ONT ID", Description: "ONT identifier", DataType: "unsignedInt", Writable: false},
				{Path: "Device.GPON.Interface.1.EquipmentID", Name: "Equipment ID", Description: "Equipment identifier", DataType: "string", Writable: false},
				{Path: "Device.GPON.Interface.1.VendorID", Name: "Vendor ID", Description: "Vendor identifier", DataType: "string", Writable: false},
				{Path: "Device.GPON.Interface.1.Version", Name: "GPON Version", Description: "GPON version", DataType: "string", Writable: false},
				{Path: "Device.GPON.Interface.1.Stats.BytesSent", Name: "GPON Bytes Sent", Description: "GPON bytes sent", DataType: "unsignedLong", Writable: false},
				{Path: "Device.GPON.Interface.1.Stats.BytesReceived", Name: "GPON Bytes Received", Description: "GPON bytes received", DataType: "unsignedLong", Writable: false},
			},
		},
		{
			Name:        "EPON Interface",
			Description: "EPON-specific parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.EPON.Interface.1.Enable", Name: "EPON Enable", Description: "Enable EPON interface", DataType: "boolean", Writable: true},
				{Path: "Device.EPON.Interface.1.Status", Name: "EPON Status", Description: "EPON interface status", DataType: "string", Writable: false},
				{Path: "Device.EPON.Interface.1.MACAddress", Name: "EPON MAC", Description: "EPON MAC address", DataType: "string", Writable: false},
				{Path: "Device.EPON.Interface.1.RXPower", Name: "EPON RX Power", Description: "EPON receive power in dBm", DataType: "int", Writable: false},
				{Path: "Device.EPON.Interface.1.TXPower", Name: "EPON TX Power", Description: "EPON transmit power in dBm", DataType: "int", Writable: false},
				{Path: "Device.EPON.Interface.1.Stats.BytesSent", Name: "EPON Bytes Sent", Description: "EPON bytes sent", DataType: "unsignedLong", Writable: false},
				{Path: "Device.EPON.Interface.1.Stats.BytesReceived", Name: "EPON Bytes Received", Description: "EPON bytes received", DataType: "unsignedLong", Writable: false},
			},
		},
		{
			Name:        "WAN Connection",
			Description: "WAN connection parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.IP.Interface.1.Enable", Name: "WAN Enable", Description: "Enable WAN interface", DataType: "boolean", Writable: true},
				{Path: "Device.IP.Interface.1.Status", Name: "WAN Status", Description: "WAN interface status", DataType: "string", Writable: false},
				{Path: "Device.IP.Interface.1.IPv4Address.1.IPAddress", Name: "WAN IP Address", Description: "WAN IPv4 address", DataType: "string", Writable: false},
				{Path: "Device.IP.Interface.1.IPv4Address.1.SubnetMask", Name: "WAN Subnet Mask", Description: "WAN subnet mask", DataType: "string", Writable: false},
				{Path: "Device.IP.Interface.1.IPv4Address.1.AddressingType", Name: "WAN Addressing Type", Description: "WAN addressing type (DHCP/Static)", DataType: "string", Writable: false},
				{Path: "Device.DHCPv4.Client.1.DHCPServer", Name: "DHCP Server", Description: "DHCP server IP address", DataType: "string", Writable: false},
				{Path: "Device.DHCPv4.Client.1.LeaseTimeRemaining", Name: "Lease Time Remaining", Description: "DHCP lease time remaining", DataType: "int", Writable: false},
			},
		},
		{
			Name:        "PPPoE Connection",
			Description: "PPPoE connection parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.PPP.Interface.1.Enable", Name: "PPP Enable", Description: "Enable PPP interface", DataType: "boolean", Writable: true},
				{Path: "Device.PPP.Interface.1.Status", Name: "PPP Status", Description: "PPP interface status", DataType: "string", Writable: false},
				{Path: "Device.PPP.Interface.1.Username", Name: "PPP Username", Description: "PPP username", DataType: "string", Writable: true},
				{Path: "Device.PPP.Interface.1.Password", Name: "PPP Password", Description: "PPP password", DataType: "string", Writable: true},
				{Path: "Device.PPP.Interface.1.ConnectionStatus", Name: "PPP Connection Status", Description: "PPP connection status", DataType: "string", Writable: false},
				{Path: "Device.PPP.Interface.1.LastConnectionError", Name: "Last Connection Error", Description: "Last PPP connection error", DataType: "string", Writable: false},
				{Path: "Device.PPP.Interface.1.ConnectTime", Name: "Connect Time", Description: "PPP connection time in seconds", DataType: "unsignedInt", Writable: false},
				{Path: "Device.PPP.Interface.1.BytesSent", Name: "PPP Bytes Sent", Description: "PPP bytes sent", DataType: "unsignedLong", Writable: false},
				{Path: "Device.PPP.Interface.1.BytesReceived", Name: "PPP Bytes Received", Description: "PPP bytes received", DataType: "unsignedLong", Writable: false},
			},
		},
		{
			Name:        "WiFi Configuration",
			Description: "WiFi interface parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.WiFi.Radio.1.Enable", Name: "WiFi Radio Enable", Description: "Enable WiFi radio", DataType: "boolean", Writable: true},
				{Path: "Device.WiFi.Radio.1.Status", Name: "WiFi Radio Status", Description: "WiFi radio status", DataType: "string", Writable: false},
				{Path: "Device.WiFi.SSID.1.SSID", Name: "WiFi SSID", Description: "WiFi network name", DataType: "string", Writable: true},
				{Path: "Device.WiFi.SSID.1.Enable", Name: "WiFi SSID Enable", Description: "Enable WiFi SSID", DataType: "boolean", Writable: true},
				{Path: "Device.WiFi.SSID.1.Status", Name: "WiFi SSID Status", Description: "WiFi SSID status", DataType: "string", Writable: false},
				{Path: "Device.WiFi.AccessPoint.1.Security.ModeEnabled", Name: "Security Mode", Description: "WiFi security mode", DataType: "string", Writable: true},
				{Path: "Device.WiFi.AccessPoint.1.Security.KeyPassphrase", Name: "WiFi Password", Description: "WiFi password", DataType: "string", Writable: true},
				{Path: "Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries", Name: "Associated Devices Count", Description: "Number of associated devices", DataType: "unsignedInt", Writable: false},
			},
		},
		{
			Name:        "Ethernet Interfaces",
			Description: "Ethernet interface parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.Ethernet.Interface.1.Enable", Name: "Ethernet Enable", Description: "Enable Ethernet interface", DataType: "boolean", Writable: true},
				{Path: "Device.Ethernet.Interface.1.Status", Name: "Ethernet Status", Description: "Ethernet interface status", DataType: "string", Writable: false},
				{Path: "Device.Ethernet.Interface.1.Alias", Name: "Ethernet Alias", Description: "Ethernet interface alias", DataType: "string", Writable: true},
				{Path: "Device.Ethernet.Interface.1.Name", Name: "Ethernet Name", Description: "Ethernet interface name", DataType: "string", Writable: false},
				{Path: "Device.Ethernet.Interface.1.MACAddress", Name: "Ethernet MAC", Description: "Ethernet MAC address", DataType: "string", Writable: false},
				{Path: "Device.Ethernet.Interface.1.MaxBitRate", Name: "Max Bit Rate", Description: "Maximum bit rate", DataType: "unsignedInt", Writable: true},
				{Path: "Device.Ethernet.Interface.1.DuplexMode", Name: "Duplex Mode", Description: "Duplex mode", DataType: "string", Writable: true},
			},
		},
		{
			Name:        "Device Management",
			Description: "Device management and monitoring parameters",
			Parameters: []TR069Parameter{
				{Path: "Device.ManagementServer.URL", Name: "ACS URL", Description: "ACS server URL", DataType: "string", Writable: true},
				{Path: "Device.ManagementServer.Username", Name: "ACS Username", Description: "ACS username", DataType: "string", Writable: true},
				{Path: "Device.ManagementServer.Password", Name: "ACS Password", Description: "ACS password", DataType: "string", Writable: true},
				{Path: "Device.ManagementServer.PeriodicInformEnable", Name: "Periodic Inform Enable", Description: "Enable periodic inform", DataType: "boolean", Writable: true},
				{Path: "Device.ManagementServer.PeriodicInformInterval", Name: "Periodic Inform Interval", Description: "Periodic inform interval in seconds", DataType: "unsignedInt", Writable: true},
				{Path: "Device.ManagementServer.ConnectionRequestURL", Name: "Connection Request URL", Description: "Connection request URL", DataType: "string", Writable: false},
				{Path: "Device.ManagementServer.ConnectionRequestUsername", Name: "Connection Request Username", Description: "Connection request username", DataType: "string", Writable: true},
				{Path: "Device.ManagementServer.ConnectionRequestPassword", Name: "Connection Request Password", Description: "Connection request password", DataType: "string", Writable: true},
				{Path: "Device.DeviceInfo.TemperatureSensor.1.Temperature", Name: "Temperature", Description: "Device temperature", DataType: "int", Writable: false},
				{Path: "Device.DeviceInfo.TemperatureSensor.1.LowAlarmValue", Name: "Low Temperature Alarm", Description: "Low temperature alarm threshold", DataType: "int", Writable: true},
				{Path: "Device.DeviceInfo.TemperatureSensor.1.HighAlarmValue", Name: "High Temperature Alarm", Description: "High temperature alarm threshold", DataType: "int", Writable: true},
			},
		},
		{
			Name:        "Vendor Specific (Legacy)",
			Description: "Legacy vendor-specific parameters for backward compatibility",
			Parameters: []TR069Parameter{
				// ZTE specific
				{Path: "InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower", Name: "ZTE RX Power", Description: "ZTE PON RX power", DataType: "int", Writable: false, Vendor: "ZTE"},
				{Path: "InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.TXPower", Name: "ZTE TX Power", Description: "ZTE PON TX power", DataType: "int", Writable: false, Vendor: "ZTE"},
				
				// Huawei specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.RXPower", Name: "Huawei RX Power", Description: "Huawei GPON RX power", DataType: "int", Writable: false, Vendor: "Huawei"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.TXPower", Name: "Huawei TX Power", Description: "Huawei GPON TX power", DataType: "int", Writable: false, Vendor: "Huawei"},
				
				// Fiberhome specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower", Name: "Fiberhome RX Power", Description: "Fiberhome GPON RX power", DataType: "int", Writable: false, Vendor: "Fiberhome"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.TXPower", Name: "Fiberhome TX Power", Description: "Fiberhome GPON TX power", DataType: "int", Writable: false, Vendor: "Fiberhome"},
				
				// CMCC (China Mobile) specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.RXPower", Name: "CMCC EPON RX Power", Description: "CMCC EPON RX power", DataType: "int", Writable: false, Vendor: "CMCC"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.RXPower", Name: "CMCC GPON RX Power", Description: "CMCC GPON RX power", DataType: "int", Writable: false, Vendor: "CMCC"},
				
				// CT-COM (China Telecom) specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower", Name: "CT-COM EPON RX Power", Description: "CT-COM EPON RX power", DataType: "int", Writable: false, Vendor: "CT-COM"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower", Name: "CT-COM GPON RX Power", Description: "CT-COM GPON RX power", DataType: "int", Writable: false, Vendor: "CT-COM"},
				
				// CU (China Unicom) specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.RXPower", Name: "CU EPON RX Power", Description: "CU EPON RX power", DataType: "int", Writable: false, Vendor: "CU"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.TXPower", Name: "CU EPON TX Power", Description: "CU EPON TX power", DataType: "int", Writable: false, Vendor: "CU"},
				
				// TP-Link specific
				{Path: "InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.RXPower", Name: "TP-Link RX Power", Description: "TP-Link GPON RX power", DataType: "int", Writable: false, Vendor: "TP-Link"},
				{Path: "InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.TXPower", Name: "TP-Link TX Power", Description: "TP-Link GPON TX power", DataType: "int", Writable: false, Vendor: "TP-Link"},
				
				// Nokia/ALU specific
				{Path: "InternetGatewayDevice.X_ALU_OntOpticalParam.RXPower", Name: "Nokia RX Power", Description: "Nokia ONT RX power", DataType: "int", Writable: false, Vendor: "Nokia"},
				{Path: "InternetGatewayDevice.X_ALU_OntOpticalParam.TXPower", Name: "Nokia TX Power", Description: "Nokia ONT TX power", DataType: "int", Writable: false, Vendor: "Nokia"},
			},
		},
	}
}

// GetONUCommonParameters returns most common ONU parameters that should be supported
func GetONUCommonParameters() []string {
	return []string{
		// Device Info
		"Device.DeviceInfo.Manufacturer",
		"Device.DeviceInfo.ManufacturerOUI",
		"Device.DeviceInfo.ModelName",
		"Device.DeviceInfo.Description",
		"Device.DeviceInfo.ProductClass",
		"Device.DeviceInfo.SerialNumber",
		"Device.DeviceInfo.HardwareVersion",
		"Device.DeviceInfo.SoftwareVersion",
		"Device.DeviceInfo.ModemFirmwareVersion",
		"Device.DeviceInfo.UpTime",
		
		// Management Server
		"Device.ManagementServer.URL",
		"Device.ManagementServer.ConnectionRequestURL",
		"Device.ManagementServer.PeriodicInformEnable",
		"Device.ManagementServer.PeriodicInformInterval",
		
		// Optical Interface
		"Device.Optical.Interface.1.Enable",
		"Device.Optical.Interface.1.Status",
		"Device.Optical.Interface.1.OpticalSignalLevel",
		"Device.Optical.Interface.1.TransmitOpticalLevel",
		"Device.Optical.Interface.1.Stats.BytesSent",
		"Device.Optical.Interface.1.Stats.BytesReceived",
		
		// GPON Interface
		"Device.GPON.Interface.1.Enable",
		"Device.GPON.Interface.1.Status",
		"Device.GPON.Interface.1.RXPower",
		"Device.GPON.Interface.1.TXPower",
		"Device.GPON.Interface.1.Temperature",
		"Device.GPON.Interface.1.Voltage",
		"Device.GPON.Interface.1.BiasCurrent",
		"Device.GPON.Interface.1.ONTID",
		
		// WAN IP Interface
		"Device.IP.Interface.1.Enable",
		"Device.IP.Interface.1.Status",
		"Device.IP.Interface.1.IPv4Address.1.IPAddress",
		"Device.IP.Interface.1.IPv4Address.1.SubnetMask",
		"Device.IP.Interface.1.IPv4Address.1.AddressingType",
		
		// PPP Interface
		"Device.PPP.Interface.1.Enable",
		"Device.PPP.Interface.1.Status",
		"Device.PPP.Interface.1.Username",
		"Device.PPP.Interface.1.ConnectionStatus",
		"Device.PPP.Interface.1.ConnectTime",
		"Device.PPP.Interface.1.BytesSent",
		"Device.PPP.Interface.1.BytesReceived",
		
		// WiFi
		"Device.WiFi.Radio.1.Enable",
		"Device.WiFi.Radio.1.Status",
		"Device.WiFi.SSID.1.SSID",
		"Device.WiFi.SSID.1.Enable",
		"Device.WiFi.SSID.1.Status",
		"Device.WiFi.AccessPoint.1.Security.ModeEnabled",
		"Device.WiFi.AccessPoint.1.AssociatedDeviceNumberOfEntries",
		
		// Ethernet
		"Device.Ethernet.Interface.1.Enable",
		"Device.Ethernet.Interface.1.Status",
		"Device.Ethernet.Interface.1.MACAddress",
		
		// Temperature
		"Device.DeviceInfo.TemperatureSensor.1.Temperature",
		"Device.DeviceInfo.TemperatureSensor.1.LowAlarmValue",
		"Device.DeviceInfo.TemperatureSensor.1.HighAlarmValue",
	}
}

// GetVendorSpecificParameters returns vendor-specific parameter paths
func GetVendorSpecificParameters() map[string][]string {
	return map[string][]string{
		"ZTE": {
			"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.TXPower",
			"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.Temperature",
			"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.Voltage",
			"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.BiasCurrent",
		},
		"Huawei": {
			"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.TXPower",
			"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.Temperature",
			"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.Voltage",
			"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.BiasCurrent",
		},
		"Fiberhome": {
			"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.TXPower",
		},
		"CMCC": {
			"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.TXPower",
			"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.TXPower",
		},
		"CT-COM": {
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.TXPower",
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.TXPower",
		},
		"CU": {
			"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.TXPower",
		},
		"TP-Link": {
			"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.TXPower",
		},
		"Nokia": {
			"InternetGatewayDevice.X_ALU_OntOpticalParam.RXPower",
			"InternetGatewayDevice.X_ALU_OntOpticalParam.TXPower",
		},
	}
}

// ParseOpticalPower converts raw optical power values to dBm
func ParseOpticalPower(rawValue string, vendor string) (float64, error) {
	// Try to parse as float first
	if power, err := strconv.ParseFloat(rawValue, 64); err == nil {
		// If already negative, it's likely already in dBm
		if power < 0 {
			return math.Round(power*100) / 100, nil
		}
		
		// If positive, apply vendor-specific conversion
		switch vendor {
		case "ZTE", "CIOT", "CT-COM", "CMCC":
			// Formula: (10 * log10(power)) - 40
			dbm := (10 * math.Log10(power)) - 40
			return math.Round(dbm*100) / 100, nil
		case "Huawei", "Fiberhome":
			// Direct conversion for some Huawei models
			return math.Round(power*100) / 100, nil
		default:
			// Default conversion for unknown vendors
			dbm := (10 * math.Log10(power)) - 40
			return math.Round(dbm*100) / 100, nil
		}
	}
	
	return 0, fmt.Errorf("invalid optical power value: %s", rawValue)
}