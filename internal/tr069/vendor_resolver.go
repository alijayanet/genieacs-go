package tr069

// VendorSpecificParameterResolver handles vendor-specific parameter mapping
type VendorSpecificParameterResolver struct {
	vendor string
	model  string
}

// NewVendorSpecificParameterResolver creates a new vendor-specific resolver
func NewVendorSpecificParameterResolver(vendor, model string) *VendorSpecificParameterResolver {
	return &VendorSpecificParameterResolver{
		vendor: vendor,
		model:  model,
	}
}

// GetVendorSpecificParameters returns vendor-specific parameters for the device
func (v *VendorSpecificParameterResolver) GetVendorSpecificParameters() []string {
	params := []string{}
	
	// Add vendor-specific parameters based on vendor
	switch v.vendor {
	case "ZTE":
		params = append(params, v.getZTEParameters()...)
	case "HUAWEI":
		params = append(params, v.getHuaweiParameters()...)
	case "FIBERHOME":
		params = append(params, v.getFiberhomeParameters()...)
	case "CMCC":
		params = append(params, v.getCMCCParameters()...)
	case "CT-COM":
		params = append(params, v.getCTComParameters()...)
	case "CU":
		params = append(params, v.getCUParameters()...)
	case "TP-LINK":
		params = append(params, v.getTPLinkParameters()...)
	case "NOKIA", "ALU":
		params = append(params, v.getNokiaParameters()...)
	}
	
	return params
}

// getZTEParameters returns ZTE-specific parameters
func (v *VendorSpecificParameterResolver) getZTEParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.OpticalModuleType",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.FECMode",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.EncryptionMode",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.BER",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.FrameLoss",
		"InternetGatewayDevice.WANDevice.1.WANDeviceIF.1.X_ZTE-COM_WANPONInterfaceConfig.OpticalSignalToNoiseRatio",
	}
}

// getHuaweiParameters returns Huawei-specific parameters
func (v *VendorSpecificParameterResolver) getHuaweiParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.OpticalModuleType",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.FECMode",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.EncryptionMode",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.BER",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.FrameLoss",
		"InternetGatewayDevice.WANDevice.1.X_GponInterafceConfig.OpticalSignalToNoiseRatio",
	}
}

// getFiberhomeParameters returns Fiberhome-specific parameters
func (v *VendorSpecificParameterResolver) getFiberhomeParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.OpticalModuleType",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.FECMode",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.EncryptionMode",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.BER",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.FrameLoss",
		"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.OpticalSignalToNoiseRatio",
	}
}

// getCMCCParameters returns China Mobile-specific parameters
func (v *VendorSpecificParameterResolver) getCMCCParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.Distance",
	}
}

// getCTComParameters returns China Telecom-specific parameters
func (v *VendorSpecificParameterResolver) getCTComParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.Distance",
		"InternetGatewayDevice.WANDevice.1.X_CT-COM_UserInfo.UserName",
	}
}

// getCUParameters returns China Unicom-specific parameters
func (v *VendorSpecificParameterResolver) getCUParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.Distance",
	}
}

// getTPLinkParameters returns TP-Link-specific parameters
func (v *VendorSpecificParameterResolver) getTPLinkParameters() []string {
	return []string{
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.RXPower",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.TXPower",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.Temperature",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.Voltage",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.BiasCurrent",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.ONTState",
		"InternetGatewayDevice.WANDevice.1.X_TPLINK_GponInterfaceConfig.Distance",
	}
}

// getNokiaParameters returns Nokia/ALU-specific parameters
func (v *VendorSpecificParameterResolver) getNokiaParameters() []string {
	return []string{
		"InternetGatewayDevice.X_ALU_OntOpticalParam.RXPower",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.TXPower",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.Temperature",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.Voltage",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.BiasCurrent",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.ONTState",
		"InternetGatewayDevice.X_ALU_OntOpticalParam.Distance",
	}
}

// GetAllParametersForDevice returns all parameters for a specific device vendor
func GetAllParametersForDevice(vendor, model string) []string {
	// Start with standard parameters
	params := GetONUCommonParameters()
	
	// Add vendor-specific parameters
	resolver := NewVendorSpecificParameterResolver(vendor, model)
	vendorParams := resolver.GetVendorSpecificParameters()
	
	// Combine and remove duplicates
	paramMap := make(map[string]bool)
	for _, param := range params {
		paramMap[param] = true
	}
	for _, param := range vendorParams {
		paramMap[param] = true
	}
	
	// Convert back to slice
	result := make([]string, 0, len(paramMap))
	for param := range paramMap {
		result = append(result, param)
	}
	
	return result
}