package provisions

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"go-acs/internal/database"
	"go-acs/internal/models"
)

// Provision represents a provisioning script that runs on device inform
type Provision struct {
	Name        string
	Description string
	Weight      int // Lower weight = higher priority
	Filter      DeviceFilter
	Actions     []ProvisionAction
}

// DeviceFilter determines which devices this provision applies to
type DeviceFilter struct {
	Manufacturer   string   // Empty = all manufacturers
	ProductClass   string   // Empty = all product classes
	SerialPatterns []string // Regex patterns for serial numbers
	Tags           []string // Device must have these tags
	ExcludeVendors []string // Exclude these vendors
}

// ProvisionAction represents an action to perform
type ProvisionAction struct {
	Type       ActionType
	Parameter  string
	Value      interface{}
	RefreshAge time.Duration // How often to refresh this parameter
}

// ActionType defines the type of provisioning action
type ActionType string

const (
	ActionDeclare      ActionType = "declare"      // Read parameter
	ActionSetValue     ActionType = "setValue"     // Set parameter value
	ActionRefresh      ActionType = "refresh"      // Force refresh
	ActionVirtualParam ActionType = "virtualParam" // Virtual parameter
)

// Refresh intervals
const (
	RefreshMinute = time.Minute
	RefreshHourly = time.Hour
	RefreshDaily  = 24 * time.Hour
)

// ProvisionEngine handles provisioning logic
type ProvisionEngine struct {
	DB         *database.DB
	Provisions []*Provision
}

// NewProvisionEngine creates a new provision engine with default provisions
func NewProvisionEngine(db *database.DB) *ProvisionEngine {
	engine := &ProvisionEngine{
		DB:         db,
		Provisions: make([]*Provision, 0),
	}

	// Register default provisions
	engine.registerDefaultProvisions()

	return engine
}

// registerDefaultProvisions loads the default provisioning scripts
func (e *ProvisionEngine) registerDefaultProvisions() {
	// Basic device info provision (all devices)
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "basic-info",
		Description: "Collect basic device information",
		Weight:      0,
		Filter:      DeviceFilter{ExcludeVendors: []string{"MikroTik"}},
		Actions: []ProvisionAction{
			{Type: ActionDeclare, Parameter: "DeviceID.Manufacturer", RefreshAge: RefreshDaily},
			{Type: ActionDeclare, Parameter: "DeviceID.ProductClass", RefreshAge: RefreshDaily},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.DeviceInfo.HardwareVersion", RefreshAge: RefreshDaily},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.DeviceInfo.SoftwareVersion", RefreshAge: RefreshDaily},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.DeviceInfo.UpTime", RefreshAge: RefreshMinute},
		},
	})

	// WiFi configuration provision
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "wifi-config",
		Description: "Monitor WiFi configuration",
		Weight:      10,
		Filter:      DeviceFilter{ExcludeVendors: []string{"MikroTik"}},
		Actions: []ProvisionAction{
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.SSID", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.PreSharedKey.1.KeyPassphrase", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.Enable", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.Channel", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.BeaconType", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.TransmitPower", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.SSIDAdvertisementEnabled", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.TotalAssociations", RefreshAge: RefreshHourly},
		},
	})

	// WiFi clients provision
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "wifi-clients",
		Description: "Monitor connected WiFi clients",
		Weight:      15,
		Filter:      DeviceFilter{ExcludeVendors: []string{"MikroTik"}},
		Actions: []ProvisionAction{
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceMACAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceIPAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.AssociatedDeviceRssi", RefreshAge: RefreshMinute},
		},
	})

	// WAN/PPPoE configuration
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "wan-config",
		Description: "Monitor WAN connections",
		Weight:      20,
		Filter:      DeviceFilter{ExcludeVendors: []string{"MikroTik"}},
		Actions: []ProvisionAction{
			// PPPoE Connection basic info
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.Name", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.Enable", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.Username", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.ConnectionType", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.NATEnabled", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.ExternalIPAddress", RefreshAge: RefreshMinute},
			// PPPoE Connection status and uptime
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.ConnectionStatus", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.Uptime", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.LastConnectionError", RefreshAge: RefreshHourly},
			// PPPoE Network info
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.MACAddress", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.RemoteIPAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.DefaultGateway", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.DNSServers", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.MaxMRUSize", RefreshAge: RefreshHourly},
			// WAN IP Connection
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.Name", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.Enable", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.ExternalIPAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.ConnectionStatus", RefreshAge: RefreshMinute},
		},
	})

	// LAN Hosts
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "lan-hosts",
		Description: "Monitor LAN hosts",
		Weight:      25,
		Filter:      DeviceFilter{ExcludeVendors: []string{"MikroTik"}},
		Actions: []ProvisionAction{
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.*.Hosts.Host.*.HostName", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.*.Hosts.Host.*.IPAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.*.Hosts.Host.*.MACAddress", RefreshAge: RefreshMinute},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.*.Hosts.Host.*.InterfaceType", RefreshAge: RefreshMinute},
		},
	})

	// Huawei specific provision
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "huawei-specific",
		Description: "Huawei device specific parameters",
		Weight:      100,
		Filter:      DeviceFilter{Manufacturer: "Huawei"},
		Actions: []ProvisionAction{
			// RX Power
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower", RefreshAge: RefreshMinute},
			// Serial Number
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.DeviceInfo.X_HW_SerialNumber", RefreshAge: RefreshDaily},
			// VLAN Config
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_HW_VLAN", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_HW_SERVICELIST", RefreshAge: RefreshDaily},
			// LAN Bind
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_HW_LANBIND.Lan1Enable", RefreshAge: RefreshDaily},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_HW_LANBIND.SSID1Enable", RefreshAge: RefreshDaily},
			// WiFi Client RSSI
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.1.WLANConfiguration.*.AssociatedDevice.*.X_HW_RSSI", RefreshAge: RefreshMinute},
			// Remote Access
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_HW_Security.AclServices.HTTPWanEnable", Value: true},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_HW_Security.AclServices.TELNETWanEnable", Value: true},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_HW_Security.X_HW_FirewallLevel", Value: "Custom"},
		},
	})

	// ZTE specific provision
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "zte-specific",
		Description: "ZTE device specific parameters",
		Weight:      100,
		Filter:      DeviceFilter{Manufacturer: "ZTE"},
		Actions: []ProvisionAction{
			// RX Power
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower", RefreshAge: RefreshMinute},
			// VLAN
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_ZTE-COM_VLANID", RefreshAge: RefreshHourly},
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.X_ZTE-COM_ServiceList", RefreshAge: RefreshDaily},
			// WiFi Client info
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.AssociatedDevice.*.X_ZTE-COM_AssociatedDeviceName", RefreshAge: RefreshMinute},
			// Remote Access
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.ServiceType", Value: "HTTP"},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.Ingress", Value: "WAN_ALL"},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.Enable", Value: true},
		},
	})

	// FiberHome specific provision
	e.Provisions = append(e.Provisions, &Provision{
		Name:        "fiberhome-specific",
		Description: "FiberHome device specific parameters",
		Weight:      100,
		Filter:      DeviceFilter{Manufacturer: "FiberHome"},
		Actions: []ProvisionAction{
			// RX Power
			{Type: ActionDeclare, Parameter: "InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower", RefreshAge: RefreshMinute},
			// Remote Access
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_FH_ACL.Enable", Value: 1},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_FH_FireWall.REMOTEACCEnable", Value: true},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_FH_ACL.Rule.1.Enable", Value: 1},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_FH_ACL.Rule.1.Direction", Value: 1},
			{Type: ActionSetValue, Parameter: "InternetGatewayDevice.X_FH_ACL.Rule.1.Protocol", Value: "ALL"},
		},
	})

	log.Printf("Registered %d default provisions", len(e.Provisions))
}

// GetProvisionActions returns all actions to execute for a device
func (e *ProvisionEngine) GetProvisionActions(device *models.Device) []ProvisionAction {
	var actions []ProvisionAction

	for _, provision := range e.Provisions {
		if e.matchesFilter(device, provision.Filter) {
			actions = append(actions, provision.Actions...)
		}
	}

	return actions
}

// matchesFilter checks if a device matches the provision filter
func (e *ProvisionEngine) matchesFilter(device *models.Device, filter DeviceFilter) bool {
	// Check excluded vendors
	for _, excluded := range filter.ExcludeVendors {
		if strings.EqualFold(device.Manufacturer, excluded) {
			return false
		}
	}

	// Check manufacturer filter
	if filter.Manufacturer != "" && !strings.EqualFold(device.Manufacturer, filter.Manufacturer) {
		return false
	}

	// Check product class
	if filter.ProductClass != "" && !strings.EqualFold(device.ProductClass, filter.ProductClass) {
		return false
	}

	return true
}

// ProcessInform handles the provisioning logic when a device sends an Inform
func (e *ProvisionEngine) ProcessInform(device *models.Device, params map[string]string) *ProvisionResult {
	result := &ProvisionResult{
		DeviceID:          device.ID,
		ParametersToRead:  make([]string, 0),
		ParametersToSet:   make(map[string]interface{}),
		VirtualParameters: make(map[string]string),
	}

	// Get all applicable actions
	actions := e.GetProvisionActions(device)

	for _, action := range actions {
		switch action.Type {
		case ActionDeclare:
			// Check if parameter needs refresh
			if e.needsRefresh(device.ID, action.Parameter, action.RefreshAge) {
				result.ParametersToRead = append(result.ParametersToRead, action.Parameter)
			}

		case ActionSetValue:
			result.ParametersToSet[action.Parameter] = action.Value

		case ActionVirtualParam:
			// Process virtual parameter
			value := e.processVirtualParameter(action.Parameter, params)
			if value != "" {
				result.VirtualParameters[action.Parameter] = value
			}
		}
	}

	return result
}

// needsRefresh checks if a parameter needs to be refreshed
func (e *ProvisionEngine) needsRefresh(deviceID int64, parameter string, maxAge time.Duration) bool {
	// Get last update time from database
	params, err := e.DB.GetDeviceParameters(deviceID, parameter)
	if err != nil || len(params) == 0 {
		return true // Parameter doesn't exist, needs refresh
	}

	// Check if older than maxAge
	for _, p := range params {
		if time.Since(p.UpdatedAt) > maxAge {
			return true
		}
	}

	return false
}

// processVirtualParameter extracts virtual parameter values
func (e *ProvisionEngine) processVirtualParameter(name string, params map[string]string) string {
	switch name {
	case "RXPower":
		// Try different vendor-specific RX Power paths
		paths := []string{
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
			"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower",
		}
		for _, path := range paths {
			if val, ok := params[path]; ok && val != "" {
				return val
			}
		}

	case "pppoeUsername":
		paths := []string{
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username",
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.2.WANPPPConnection.1.Username",
		}
		for _, path := range paths {
			if val, ok := params[path]; ok && val != "" {
				return val
			}
		}

	case "pppoeIP":
		paths := []string{
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.ExternalIPAddress",
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.2.WANPPPConnection.1.ExternalIPAddress",
		}
		for _, path := range paths {
			if val, ok := params[path]; ok && val != "" {
				return val
			}
		}

	case "SSID":
		if val, ok := params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"]; ok {
			return val
		}

	case "WlanPassword":
		if val, ok := params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"]; ok {
			return val
		}
	}

	return ""
}

// ProvisionResult contains the result of processing provisions
type ProvisionResult struct {
	DeviceID          int64
	ParametersToRead  []string
	ParametersToSet   map[string]interface{}
	VirtualParameters map[string]string
}

// ToJSON converts result to JSON for storage
func (r *ProvisionResult) ToJSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// VirtualParameter represents a computed/virtual parameter
type VirtualParameter struct {
	Name       string
	Script     string // JavaScript-like script for extraction
	Paths      []string
	Aggregator string // "first", "sum", "avg", "concat"
}

// DefaultVirtualParameters returns the default virtual parameters
func DefaultVirtualParameters() []VirtualParameter {
	return []VirtualParameter{
		{
			Name: "RXPower",
			Paths: []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
				"InternetGatewayDevice.WANDevice.1.X_FH_GponInterfaceConfig.RXPower",
			},
			Aggregator: "first",
		},
		{
			Name: "pppoeUsername",
			Paths: []string{
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.Username",
			},
			Aggregator: "first",
		},
		{
			Name: "pppoeIP",
			Paths: []string{
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANPPPConnection.*.ExternalIPAddress",
			},
			Aggregator: "first",
		},
		{
			Name: "SSID",
			Paths: []string{
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID",
			},
			Aggregator: "first",
		},
		{
			Name: "WlanPassword",
			Paths: []string{
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase",
			},
			Aggregator: "first",
		},
		{
			Name: "getSerialNumber",
			Paths: []string{
				"InternetGatewayDevice.DeviceInfo.SerialNumber",
				"InternetGatewayDevice.DeviceInfo.X_HW_SerialNumber",
			},
			Aggregator: "first",
		},
		{
			Name: "getdeviceuptime",
			Paths: []string{
				"InternetGatewayDevice.DeviceInfo.UpTime",
			},
			Aggregator: "first",
		},
		{
			Name: "activedevices",
			Paths: []string{
				"InternetGatewayDevice.LANDevice.1.Hosts.HostNumberOfEntries",
			},
			Aggregator: "first",
		},
		{
			Name: "getponmode",
			Paths: []string{
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_GponInterfaceConfig",
				"InternetGatewayDevice.WANDevice.1.X_CT-COM_EponInterfaceConfig",
			},
			Aggregator: "first",
		},
	}
}

// GetVirtualParameterValue extracts a virtual parameter value from device params
func GetVirtualParameterValue(vp VirtualParameter, params map[string]string) string {
	var values []string

	for _, path := range vp.Paths {
		// Handle wildcard paths
		if strings.Contains(path, "*") {
			for key, val := range params {
				if matchWildcard(path, key) && val != "" {
					values = append(values, val)
				}
			}
		} else {
			if val, ok := params[path]; ok && val != "" {
				values = append(values, val)
			}
		}
	}

	switch vp.Aggregator {
	case "first":
		if len(values) > 0 {
			return values[0]
		}
	case "concat":
		return strings.Join(values, ",")
	}

	return ""
}

// matchWildcard checks if a path matches a wildcard pattern
func matchWildcard(pattern, path string) bool {
	patternParts := strings.Split(pattern, ".")
	pathParts := strings.Split(path, ".")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i, pp := range patternParts {
		if pp == "*" {
			continue
		}
		if pp != pathParts[i] {
			return false
		}
	}

	return true
}

// ProvisionScript represents a user-defined provision script
type ProvisionScript struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Script      string    `json:"script"` // JavaScript-like DSL
	Filter      string    `json:"filter"` // JSON filter
	Weight      int       `json:"weight"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Declare represents a parameter declaration
type Declare struct {
	Path        string
	PathConfig  *RefreshConfig
	ValueConfig *RefreshConfig
	SetValue    interface{}
}

// RefreshConfig represents refresh configuration
type RefreshConfig struct {
	MaxAge time.Duration
}

// Log provision execution
func (e *ProvisionEngine) LogExecution(deviceID int64, provisionName string, actions int, errors []string) {
	level := "info"
	if len(errors) > 0 {
		level = "error"
	}

	msg := fmt.Sprintf("Provision '%s' executed: %d actions", provisionName, actions)
	details := ""
	if len(errors) > 0 {
		details = strings.Join(errors, "; ")
	}

	e.DB.CreateLog(&deviceID, level, "provision", msg, details)
}
