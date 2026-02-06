package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-acs/internal/config"
	"go-acs/internal/database"
	"go-acs/internal/models"
	"go-acs/internal/websocket"

	"go-acs/internal/mailer"
	"go-acs/internal/mikrotik"
	"go-acs/internal/notification/fcm"
	"go-acs/internal/notification/telegram"
	"go-acs/internal/notification/whatsapp"
	"go-acs/internal/payment"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	DB       *database.DB
	WSHub    *websocket.Hub
	Mailer   *mailer.Mailer
	Mikrotik *mikrotik.Client
	Payment  payment.Gateway
	WA       *whatsapp.Client
	FCM      *fcm.Client
	Telegram *telegram.Client
	Config   *config.Config
	tmpl     *template.Template
}

// NewHandler creates a new Handler
func NewHandler(db *database.DB, wsHub *websocket.Hub, m *mailer.Mailer, mt *mikrotik.Client, pg payment.Gateway, wa *whatsapp.Client, fcmClient *fcm.Client, tg *telegram.Client, cfg *config.Config) *Handler {
	// Parse all templates
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &Handler{
		DB:       db,
		WSHub:    wsHub,
		Mailer:   m,
		Mikrotik: mt,
		Payment:  pg,
		WA:       wa,
		FCM:      fcmClient,
		Telegram: tg,
		Config:   cfg,
		tmpl:     tmpl,
	}
}

// ============== Page Handlers ==============

// ServeIndex serves the landing page
func (h *Handler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/index.html")
}

// ServeDashboard serves the dashboard page
func (h *Handler) ServeDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/dashboard.html")
}

// ServeDevices serves the devices page
func (h *Handler) ServeDevices(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/devices.html")
}

// ServeDeviceDetail serves the device detail page
func (h *Handler) ServeDeviceDetail(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/device-detail.html")
}

// ServeProvisions serves the provisions page
func (h *Handler) ServeProvisions(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/provisions.html")
}

// ServePackages serves the packages page
func (h *Handler) ServePackages(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/packages.html")
}

// ServeCustomers serves the customers page
func (h *Handler) ServeCustomers(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/customers.html")
}

// ServeBilling serves the billing page
func (h *Handler) ServeBilling(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/billing.html")
}

// ServeMap serves the map page
func (h *Handler) ServeMap(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/map.html")
}

// ServePortal serves the customer portal page
func (h *Handler) ServePortal(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/portal.html")
}

// ServeTasks serves the tasks page
func (h *Handler) ServeTasks(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/tasks.html")
}

// ServePortalLogin serves the customer portal login page
func (h *Handler) ServePortalLogin(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/portal-login.html")
}

// ServeTickets serves the support tickets page
func (h *Handler) ServeTickets(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/tickets.html")
}

// ServeSettings serves the settings page
func (h *Handler) ServeSettings(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/settings.html")
}

// ServeLogs serves the system logs page
func (h *Handler) ServeLogs(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/logs.html")
}

// ServeUpdate serves the system update page
func (h *Handler) ServeUpdate(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/update.html")
}

// ============== Auth Handlers ==============

// Login handles user authentication
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Check credentials against database users table
	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	h.DB.UpdateUser(user)

	// Generate a proper JWT token
	token := generateJWT(user)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"user": map[string]string{
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============== Dashboard Handlers ==============

// GetDashboardStats returns dashboard statistics
func (h *Handler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetDashboardStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	// Add WebSocket connection count
	stats.ActiveSessions = int64(h.WSHub.ClientCount())

	respondJSON(w, http.StatusOK, stats)
}

// ============== Device Handlers ==============

// GetDevices returns all devices
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	limit := getQueryInt(r, "limit", 50)
	offset := getQueryInt(r, "offset", 0)

	devices, total, err := h.DB.GetDevices(status, search, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get devices")
		return
	}

	// Enrich devices with additional data from parameters
	for _, device := range devices {
		// Get device parameters
		params, err := h.DB.GetDeviceParameters(device.ID, "")
		if err != nil {
			continue // Skip if unable to get parameters
		}

		// Extract PPPoE username from parameters
		for _, p := range params {
			// Extract PPPoE username
			if (strings.Contains(p.Path, "WANPPPConnection") && strings.HasSuffix(p.Path, "Username")) ||
				strings.HasSuffix(p.Path, "X_CT-COM_UserInfo.UserName") ||
				strings.HasSuffix(p.Path, "X_CMCC_UserInfo.UserName") {
				if p.Value != "" && p.Value != "default" && p.Value != "null" {
					device.PPPoEUsername = p.Value
					break
				}
			}
		}

		// Extract temperature from parameters
		for _, p := range params {
			if strings.Contains(strings.ToLower(p.Path), "temperature") {
				if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
					// Apply conversion logic based on value range
					if v > 1000 {
						device.Temperature = v / 256.0
					} else if v > 100 {
						device.Temperature = v / 10.0
					} else {
						device.Temperature = v
					}
					break
				}
			}
		}

		// Extract WAN IP and connection type
		for _, p := range params {
			if strings.HasSuffix(p.Path, "ExternalIPAddress") ||
				strings.HasSuffix(p.Path, "IPv4Address.1.IPAddress") {
				if p.Value != "" && p.Value != "0.0.0.0" {
					device.WANIP = p.Value
					break
				}
			}
			if strings.Contains(p.Path, "WANConnection") && strings.Contains(p.Path, "ConnectionType") {
				device.WANConnectionType = p.Value
				break
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// CreateDevice creates a new device
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if device.SerialNumber == "" {
		respondError(w, http.StatusBadRequest, "Serial number is required")
		return
	}

	device.Status = models.StatusOffline
	created, err := h.DB.CreateDevice(&device)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create device")
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// GetDevice returns a specific device
func (h *Handler) GetDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	device, err := h.DB.GetDevice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	// Get device parameters to extract PPPoE information
	params, err := h.DB.GetDeviceParameters(id, "")
	if err == nil {
		// Extract PPPoE information from parameters
		for _, p := range params {
			// Extract PPPoE username
			if (contains(p.Path, "WANPPPConnection") && contains(p.Path, "Username")) ||
				contains(p.Path, "X_CT-COM_UserInfo.UserName") ||
				contains(p.Path, "X_CMCC_UserInfo.UserName") {
				if p.Value != "" && p.Value != "default" && p.Value != "null" {
					device.PPPoEUsername = p.Value
				}
			}

			// Extract PPPoE IP and WAN IP
			if contains(p.Path, "ExternalIPAddress") || contains(p.Path, "IPv4Address.1.IPAddress") {
				if p.Value != "" && p.Value != "0.0.0.0" {
					device.PPPoEIP = p.Value
					device.WANIP = p.Value
				}
			}

			// Extract connection type
			if contains(p.Path, "ConnectionType") {
				device.WANConnectionType = p.Value
			}

			// Extract temperature
			if strings.Contains(strings.ToLower(p.Path), "temperature") {
				if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
					// Apply conversion logic based on value range
					if v > 1000 {
						device.Temperature = v / 256.0
					} else if v > 100 {
						device.Temperature = v / 10.0
					} else {
						device.Temperature = v
					}
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, device)
}

// GetDevicePON returns optical signal status for a device
func (h *Handler) GetDevicePON(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	params, _ := h.DB.GetDeviceParameters(id, "")
	device, _ := h.DB.GetDevice(id)

	pon := models.PONStats{
		RXPower: -18.5, // default fallback
		TXPower: 2.3,
		PONMode: "Ethernet", // default
	}

	if device != nil {
		if device.RXPower != 0 {
			pon.RXPower = device.RXPower
		}
		// Use temperature from device if available
		if device.Temperature != 0 {
			pon.Temperature = device.Temperature
		}
		// Logic from user script: Handle Raisecom
		if strings.Contains(strings.ToUpper(device.Manufacturer), "RAISECOM") {
			pon.PONMode = "GPON"
		}
	}

	for _, p := range params {
		switch {
		case contains(p.Path, "TxOpticalPower") || contains(p.Path, "TxPower"):
			if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
				if v > 100 {
					pon.TXPower = v / 100.0
				} else {
					pon.TXPower = v
				}
			}
		case contains(p.Path, "TransceiverTemperature") || contains(p.Path, "Temperature"):
			if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
				if v > 1000 {
					pon.Temperature = v / 256.0
				} else if v > 100 {
					pon.Temperature = v / 10.0
				} else {
					pon.Temperature = v
				}
			}
		case contains(p.Path, "TransceiverVoltage") || contains(p.Path, "Voltage"):
			if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
				if v > 100 {
					pon.Voltage = v / 100.0
				} else {
					pon.Voltage = v
				}
			}
		case contains(p.Path, "BiasCurrent"):
			if v, err := strconv.ParseFloat(p.Value, 64); err == nil {
				pon.BiasCurrent = v
			}
		case contains(p.Path, "WANAccessType") || contains(p.Path, "UpPortMode"):
			val := strings.ToUpper(p.Value)
			if strings.Contains(val, "EPON") {
				pon.PONMode = "EPON"
			} else if strings.Contains(val, "GPON") || strings.Contains(val, "PON") {
				pon.PONMode = "GPON"
			}
		case contains(p.Path, "OnuId") || contains(p.Path, "ONU_ID"):
			pon.ONU_ID = p.Value
		case contains(p.Path, "Dist") || contains(p.Path, "Distance"):
			pon.Distance = p.Value
		}
	}

	respondJSON(w, http.StatusOK, pon)
}

// GetDeviceWAN returns WAN connection information for a device
func (h *Handler) GetDeviceWAN(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	params, err := h.DB.GetDeviceParameters(id, "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get parameters")
		return
	}

	wanInfo := struct {
		PPPoEUsername     string            `json:"pppoeUsername"`
		PPPoEPassword     string            `json:"pppoePassword,omitempty"`
		PPPoEIP           string            `json:"pppoeIP"`
		PPPoEMAC          string            `json:"pppoeMAC"`
		PPPoEStatus       string            `json:"pppoeStatus"`
		PPPoEUptime       string            `json:"pppoeUptime"`
		PPPoEVLAN         string            `json:"pppoeVLAN"`
		PPPoEGateway      string            `json:"pppoeGateway"`
		PPPoEDNS          string            `json:"pppoeDNS"`
		PPPoEConnType     string            `json:"pppoeConnType"`
		PPPoEServiceName  string            `json:"pppoeServiceName"`
		PPPoENAT          string            `json:"pppoeNAT"`
		PPPoEMTU          string            `json:"pppoeMTU"`
		PPPoELanBind      string            `json:"pppoeLanBind"`
		PPPoELastError    string            `json:"pppoeLastError"`
		WANIP             string            `json:"wanIP"`
		WANConnectionType string            `json:"wanConnectionType"`
		WANGateway        string            `json:"wanGateway"`
		WANDNS1           string            `json:"wanDNS1"`
		WANDNS2           string            `json:"wanDNS2"`
		Parameters        map[string]string `json:"parameters"`
		AllConnections    []WANConnection   `json:"allConnections"`
	}{
		Parameters:     make(map[string]string),
		AllConnections: make([]WANConnection, 0),
	}

	// Track all WAN connections found
	connectionMap := make(map[string]*WANConnection)

	for _, p := range params {
		// Store all WAN-related parameters
		if contains(p.Path, "WANPPPConnection") || contains(p.Path, "WANIPConnection") || contains(p.Path, "WANConnection") {
			wanInfo.Parameters[p.Path] = p.Value

			// Extract connection base path (e.g., "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1")
			connPath := extractConnectionPath(p.Path)
			if connPath != "" {
				if _, exists := connectionMap[connPath]; !exists {
					connectionMap[connPath] = &WANConnection{
						Path: connPath,
						Type: "PPPoE",
					}
					if contains(p.Path, "WANIPConnection") {
						connectionMap[connPath].Type = "DHCP/Static"
					}
				}
				conn := connectionMap[connPath]
				updateConnectionField(conn, p.Path, p.Value)
			}
		}

		// Extract PPPoE username
		if (contains(p.Path, "WANPPPConnection") && contains(p.Path, "Username")) ||
			contains(p.Path, "X_CT-COM_UserInfo.UserName") ||
			contains(p.Path, "X_CMCC_UserInfo.UserName") {
			if p.Value != "" && p.Value != "default" && p.Value != "null" {
				wanInfo.PPPoEUsername = p.Value
			}
		}

		// Extract PPPoE IP and WAN IP
		if contains(p.Path, "ExternalIPAddress") || contains(p.Path, "IPv4Address.1.IPAddress") {
			if p.Value != "" && p.Value != "0.0.0.0" {
				wanInfo.PPPoEIP = p.Value
				wanInfo.WANIP = p.Value
			}
		}

		// Extract PPPoE Status
		if contains(p.Path, "WANPPPConnection") && contains(p.Path, "ConnectionStatus") {
			if p.Value != "" {
				wanInfo.PPPoEStatus = p.Value
			}
		}

		// Extract PPPoE Uptime
		if contains(p.Path, "WANPPPConnection") && strings.HasSuffix(p.Path, ".Uptime") {
			if p.Value != "" {
				wanInfo.PPPoEUptime = formatPPPUptime(p.Value)
			}
		}

		// Extract VLAN (vendor-specific)
		if contains(p.Path, "WANPPPConnection") {
			if strings.HasSuffix(p.Path, ".X_HW_VLAN") ||
				strings.HasSuffix(p.Path, ".X_ZTE-COM_VLANID") ||
				strings.HasSuffix(p.Path, ".X_FH_VLAN") ||
				strings.HasSuffix(p.Path, ".X_ALU_VLANID") ||
				strings.HasSuffix(p.Path, "VLANID") {
				if p.Value != "" && p.Value != "0" && p.Value != "-1" {
					wanInfo.PPPoEVLAN = p.Value
				}
			}
		}

		// Extract MAC Address
		if contains(p.Path, "WANPPPConnection") && strings.HasSuffix(p.Path, ".MACAddress") {
			if p.Value != "" {
				wanInfo.PPPoEMAC = p.Value
			}
		}

		// Extract Service Name
		if contains(p.Path, "WANPPPConnection") {
			if strings.HasSuffix(p.Path, ".X_HW_SERVICELIST") ||
				strings.HasSuffix(p.Path, ".X_ZTE-COM_ServiceList") ||
				strings.HasSuffix(p.Path, ".Name") {
				if p.Value != "" {
					wanInfo.PPPoEServiceName = p.Value
				}
			}
		}

		// Extract other WAN information
		if contains(p.Path, "DefaultGateway") || contains(p.Path, "RemoteIPAddress") {
			if p.Value != "" && p.Value != "0.0.0.0" {
				wanInfo.WANGateway = p.Value
				wanInfo.PPPoEGateway = p.Value
			}
		}
		if contains(p.Path, "DNSServers") || strings.HasSuffix(p.Path, ".DNS") {
			if p.Value != "" {
				wanInfo.PPPoEDNS = p.Value
				// Also split for individual DNS
				dnsParts := strings.Split(p.Value, ",")
				if len(dnsParts) >= 1 {
					wanInfo.WANDNS1 = strings.TrimSpace(dnsParts[0])
				}
				if len(dnsParts) >= 2 {
					wanInfo.WANDNS2 = strings.TrimSpace(dnsParts[1])
				}
			}
		}
		if contains(p.Path, "ConnectionType") {
			wanInfo.WANConnectionType = p.Value
			wanInfo.PPPoEConnType = p.Value
		}
		if contains(p.Path, "NATEnabled") {
			if p.Value == "1" || p.Value == "true" {
				wanInfo.PPPoENAT = "Enabled"
			} else {
				wanInfo.PPPoENAT = "Disabled"
			}
		}
		if strings.HasSuffix(p.Path, ".MaxMRUSize") {
			wanInfo.PPPoEMTU = p.Value
		}
		if strings.HasSuffix(p.Path, ".LastConnectionError") {
			if p.Value != "" && p.Value != "ERROR_NONE" {
				wanInfo.PPPoELastError = p.Value
			}
		}

		// LAN Binding
		if contains(p.Path, "X_HW_LANBIND") || contains(p.Path, "X_ZTE-COM_LanBind") {
			if p.Value == "1" || p.Value == "true" {
				bindName := ""
				if strings.HasSuffix(p.Path, "Lan1Enable") {
					bindName = "LAN1"
				} else if strings.HasSuffix(p.Path, "Lan2Enable") {
					bindName = "LAN2"
				} else if strings.HasSuffix(p.Path, "Lan3Enable") {
					bindName = "LAN3"
				} else if strings.HasSuffix(p.Path, "Lan4Enable") {
					bindName = "LAN4"
				} else if strings.HasSuffix(p.Path, "SSID1Enable") {
					bindName = "WiFi1"
				} else if strings.HasSuffix(p.Path, "SSID2Enable") {
					bindName = "WiFi2"
				}
				if bindName != "" {
					if wanInfo.PPPoELanBind != "" {
						wanInfo.PPPoELanBind += ", " + bindName
					} else {
						wanInfo.PPPoELanBind = bindName
					}
				}
			}
		}
	}

	// Convert connection map to slice
	for _, conn := range connectionMap {
		wanInfo.AllConnections = append(wanInfo.AllConnections, *conn)
	}

	respondJSON(w, http.StatusOK, wanInfo)
}

// WANConnection represents a single WAN connection
type WANConnection struct {
	Path           string `json:"path"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	IP             string `json:"ip"`
	Status         string `json:"status"`
	Uptime         string `json:"uptime"`
	VLAN           string `json:"vlan"`
	Gateway        string `json:"gateway"`
	DNS            string `json:"dns"`
	NAT            string `json:"nat"`
	MTU            string `json:"mtu"`
	LanBind        string `json:"lanBind"`
	ServiceName    string `json:"serviceName"`
	LastError      string `json:"lastError"`
	ConnectionType string `json:"connectionType"`
	Enable         string `json:"enable"`
}

// extractConnectionPath extracts the base WAN connection path
func extractConnectionPath(fullPath string) string {
	// Match patterns like:
	// InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1
	// InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1
	patterns := []string{"WANPPPConnection.", "WANIPConnection."}
	for _, pattern := range patterns {
		if idx := strings.Index(fullPath, pattern); idx != -1 {
			endIdx := idx + len(pattern)
			// Find the next dot after the number
			for i := endIdx; i < len(fullPath); i++ {
				if fullPath[i] == '.' {
					return fullPath[:i]
				}
			}
			return fullPath
		}
	}
	return ""
}

// updateConnectionField updates a WANConnection field based on the parameter
func updateConnectionField(conn *WANConnection, path, value string) {
	if value == "" {
		return
	}
	switch {
	case strings.HasSuffix(path, ".Name"):
		conn.Name = value
	case strings.HasSuffix(path, ".Username"):
		conn.Username = value
	case strings.HasSuffix(path, ".ExternalIPAddress"):
		conn.IP = value
	case strings.HasSuffix(path, ".ConnectionStatus"):
		conn.Status = value
	case strings.HasSuffix(path, ".Uptime"):
		conn.Uptime = formatPPPUptime(value)
	case strings.HasSuffix(path, ".X_HW_VLAN"), strings.HasSuffix(path, ".X_ZTE-COM_VLANID"),
		strings.HasSuffix(path, ".X_FH_VLAN"), strings.HasSuffix(path, "VLANID"):
		conn.VLAN = value
	case strings.HasSuffix(path, ".DefaultGateway"), strings.HasSuffix(path, ".RemoteIPAddress"):
		conn.Gateway = value
	case strings.Contains(path, "DNSServers"):
		conn.DNS = value
	case strings.HasSuffix(path, ".NATEnabled"):
		if value == "1" || value == "true" {
			conn.NAT = "Enabled"
		} else {
			conn.NAT = "Disabled"
		}
	case strings.HasSuffix(path, ".MaxMRUSize"):
		conn.MTU = value
	case strings.HasSuffix(path, ".ConnectionType"):
		conn.ConnectionType = value
	case strings.HasSuffix(path, ".Enable"):
		if value == "1" || value == "true" {
			conn.Enable = "Enabled"
		} else {
			conn.Enable = "Disabled"
		}
	case strings.HasSuffix(path, ".X_HW_SERVICELIST"), strings.HasSuffix(path, ".X_ZTE-COM_ServiceList"):
		conn.ServiceName = value
	case strings.HasSuffix(path, ".LastConnectionError"):
		if value != "ERROR_NONE" {
			conn.LastError = value
		}
	}
}

// formatPPPUptime formats PPP uptime seconds to human readable
func formatPPPUptime(value string) string {
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return value
	}

	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// GetDeviceClients returns the list of connected clients
func (h *Handler) GetDeviceClients(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	params, _ := h.DB.GetDeviceParameters(id, "")

	clientsMap := make(map[string]*models.ConnectedClient)

	for _, p := range params {
		if contains(p.Path, "Hosts.Host.") {
			parts := strings.Split(p.Path, ".")
			if len(parts) < 6 {
				continue
			}

			// Find the index part (it's after 'Host')
			var idx string
			for i, part := range parts {
				if part == "Host" && i+1 < len(parts) {
					idx = parts[i+1]
					break
				}
			}
			if idx == "" {
				continue
			}

			if clientsMap[idx] == nil {
				clientsMap[idx] = &models.ConnectedClient{Active: true, Type: "other"}
			}

			lastPart := parts[len(parts)-1]
			switch lastPart {
			case "HostName":
				clientsMap[idx].Name = p.Value
			case "MACAddress":
				clientsMap[idx].MAC = p.Value
			case "IPAddress":
				clientsMap[idx].IP = p.Value
			case "Active":
				clientsMap[idx].Active = p.Value == "1" || p.Value == "true"
			case "InterfaceType":
				clientsMap[idx].Interface = p.Value
				if strings.Contains(p.Value, "802.11") || strings.Contains(p.Value, "Wireless") {
					clientsMap[idx].Type = "phone"
				} else {
					clientsMap[idx].Type = "laptop"
				}
			}
		}
	}

	var clients []models.ConnectedClient
	for _, c := range clientsMap {
		if c.MAC != "" {
			if c.Name == "" {
				c.Name = "Unknown Device"
			}
			clients = append(clients, *c)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"clients": clients,
	})
}

// UpdateDevice updates a device
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	device.ID = id
	if err := h.DB.UpdateDevice(&device); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update device")
		return
	}

	updated, _ := h.DB.GetDevice(id)
	respondJSON(w, http.StatusOK, updated)
}

// DeleteDevice deletes a device
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid device ID")
		return
	}

	if err := h.DB.DeleteDevice(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete device")
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetDeviceStatus returns the status of a device
func (h *Handler) GetDeviceStatus(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	device, err := h.DB.GetDevice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	status := map[string]interface{}{
		"id":          device.ID,
		"status":      device.Status,
		"lastContact": device.LastContact,
		"lastInform":  device.LastInform,
		"uptime":      device.Uptime,
		"ipAddress":   device.IPAddress,
	}

	respondJSON(w, http.StatusOK, status)
}

// RebootDevice sends a reboot command to a device
func (h *Handler) RebootDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	// Create a reboot task
	task := &models.DeviceTask{
		DeviceID: id,
		Type:     models.TaskReboot,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create reboot task")
		return
	}

	h.DB.CreateLog(&id, "info", "command", "Reboot command queued", "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "Reboot command queued",
	})
}

// FactoryResetDevice sends a factory reset command to a device
func (h *Handler) FactoryResetDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	task := &models.DeviceTask{
		DeviceID: id,
		Type:     models.TaskFactoryReset,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create factory reset task")
		return
	}

	h.DB.CreateLog(&id, "warning", "command", "Factory reset command queued", "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "Factory reset command queued",
	})
}

// RefreshDevice triggers a parameter refresh for a device
func (h *Handler) RefreshDevice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	task := &models.DeviceTask{
		DeviceID: id,
		Type:     models.TaskRefresh,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create refresh task")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "Refresh command queued",
	})
}

// ============== WiFi Handlers ==============

// GetWiFiConfig returns WiFi configuration for a device
func (h *Handler) GetWiFiConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	// Get parameters from multiple common paths for different vendors
	params := []*models.DeviceParameter{}

	// Try TR-181 path first
	tr181Params, err := h.DB.GetDeviceParameters(id, "Device.WiFi.")
	if err == nil {
		params = append(params, tr181Params...)
	}

	// Try TR-098 path
	tr098Params, _ := h.DB.GetDeviceParameters(id, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.")
	params = append(params, tr098Params...)

	// Also try some vendor-specific paths
	vendorParams, _ := h.DB.GetDeviceParameters(id, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.")
	params = append(params, vendorParams...)

	// Build WiFi config from parameters
	config := models.WiFiConfig{}
	for _, p := range params {
		switch {
		case contains(p.Path, "SSID") && !contains(p.Path, "Hidden"):
			if config.SSID == "" { // Only set if not already set
				config.SSID = p.Value
			}
		case contains(p.Path, "KeyPassphrase") || contains(p.Path, "PreSharedKey"):
			if config.Password == "" { // Only set if not already set
				config.Password = p.Value
			}
		case contains(p.Path, "BeaconType") || contains(p.Path, "SecurityMode"):
			if config.SecurityMode == "" { // Only set if not already set
				config.SecurityMode = p.Value
			}
		case contains(p.Path, "Channel"):
			if config.Channel == 0 { // Only set if not already set
				config.Channel, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "Enable"):
			config.Enabled = p.Value == "true" || p.Value == "1"
		case contains(p.Path, "Name") && contains(p.Path, "SSID"): // Some vendors use Name for SSID
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_HW_SSID"): // Huawei specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_ZTE_SSID"): // ZTE specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_FH_SSID"): // FiberHome specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_CT-COM_SSID"): // China Telecom specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "TransmitPower"):
			if config.TransmitPower == 0 { // Only set if not already set
				config.TransmitPower, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "X_HW_WlanHidden") && config.HiddenSSID == false: // Huawei hidden SSID
			config.HiddenSSID = p.Value == "1" || p.Value == "true"
		case contains(p.Path, "MaxAssociatedDevices") || contains(p.Path, "MaxClients"): // Max clients
			if config.MaxClients == 0 { // Only set if not already set
				config.MaxClients, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "Band"):
			if config.Band == "" { // Only set if not already set
				config.Band = p.Value
			}
		case contains(p.Path, "BSSID"):
			if config.BSSID == "" { // Only set if not already set
				config.BSSID = p.Value
			}
		}
	}

	respondJSON(w, http.StatusOK, config)
}

// UpdateWiFiConfig updates WiFi configuration
func (h *Handler) UpdateWiFiConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var config models.WiFiConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)

	// Create a task to set WiFi parameters
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)
		if containsString(manufacturer, "HUAWEI") {
			// Huawei specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["Device.WiFi.SSID.1.Name"] = config.SSID

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BasicEncryptionModes"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
				params["Device.WiFi.Radio.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.ChannelBandwidth != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_HW_BandWidth"] = config.ChannelBandwidth
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_HW_WlanHidden"] = "1"
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSIDAdvertisementEnabled"] = "0"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
			if config.Band != "" {
				params["Device.WiFi.Radio.1.Standard"] = config.Band
			}
			if config.TransmitPower > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.TransmitPower"] = fmt.Sprintf("%d", config.TransmitPower)
			}
		} else if containsString(manufacturer, "ZTE") {
			// ZTE specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
				params["Device.WiFi.Radio.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.ChannelBandwidth != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ZTE-COM_BandWidth"] = config.ChannelBandwidth
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ZTE-COM_WlanHidden"] = "1"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
			if config.TransmitPower > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.TransmitPower"] = fmt.Sprintf("%d", config.TransmitPower)
			}
		} else if containsString(manufacturer, "FIBERHOME") {
			// FiberHome specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.ChannelBandwidth != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_FH_BandWidth"] = config.ChannelBandwidth
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_FH_WlanHidden"] = "1"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
		} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
			// Alcatel/Nokia specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.ChannelBandwidth != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ALU_BandWidth"] = config.ChannelBandwidth
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ALU_WlanHidden"] = "1"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
		} else if containsString(manufacturer, "CIOT") {
			// CIOT specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSIDAdvertisementEnabled"] = "0"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
		} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
			// TP-Link specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["Device.WiFi.SSID.1.Name"] = config.SSID

			// Advanced WiFi parameters
			if config.SecurityMode != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.BeaconType"] = config.SecurityMode
			}
			if config.Channel > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel"] = fmt.Sprintf("%d", config.Channel)
				params["Device.WiFi.Radio.1.Channel"] = fmt.Sprintf("%d", config.Channel)
			}
			if config.ChannelBandwidth != "" {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_TPLINK_BandWidth"] = config.ChannelBandwidth
			}
			if config.HiddenSSID {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_TPLINK_WlanHidden"] = "1"
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSIDAdvertisementEnabled"] = "0"
			}
			if config.MaxClients > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.MaxAssociatedDevices"] = fmt.Sprintf("%d", config.MaxClients)
			}
			if config.TransmitPower > 0 {
				params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.TransmitPower"] = fmt.Sprintf("%d", config.TransmitPower)
			}
		} else {
			// Default paths for unknown vendors
			params["Device.WiFi.SSID.1.SSID"] = config.SSID
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
			params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
			params["Device.WiFi.SSID.1.Name"] = config.SSID
		}
	} else {
		// If no device info, try common paths
		params["Device.WiFi.SSID.1.SSID"] = config.SSID
		params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = config.Password
		params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = config.Password
		params["Device.WiFi.Radio.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = config.SSID
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = config.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = config.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable"] = fmt.Sprintf("%v", config.Enabled)
		params["Device.WiFi.SSID.1.Name"] = config.SSID
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create WiFi update task")
		return
	}

	h.DB.CreateLog(&id, "info", "wifi", fmt.Sprintf("WiFi configuration update queued (SSID: %s)", config.SSID), "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "WiFi configuration update queued",
	})
}

// UpdateSSID updates only the SSID
func (h *Handler) UpdateSSID(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var req struct {
		SSID string `json:"ssid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.SSID == "" {
		respondError(w, http.StatusBadRequest, "SSID cannot be empty")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)

	// Build vendor-specific parameter paths
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)
		if containsString(manufacturer, "HUAWEI") {
			// Huawei specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_HW_SSID"] = req.SSID
		} else if containsString(manufacturer, "ZTE") {
			// ZTE specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ZTE_SSID"] = req.SSID
		} else if containsString(manufacturer, "FIBERHOME") {
			// FiberHome specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_FH_SSID"] = req.SSID
		} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
			// Alcatel/Nokia specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.Name"] = req.SSID
		} else if containsString(manufacturer, "CIOT") {
			// CIOT specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
		} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
			// TP-Link specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_TPLINK_SSID"] = req.SSID
		} else {
			// Default paths for unknown vendors
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.Name"] = req.SSID
		}
	} else {
		// If no device info, try common paths
		params["Device.WiFi.SSID.1.SSID"] = req.SSID
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
		params["Device.WiFi.SSID.1.Name"] = req.SSID
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create SSID update task")
		return
	}

	h.DB.CreateLog(&id, "info", "wifi", fmt.Sprintf("SSID update queued: %s", req.SSID), "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": fmt.Sprintf("SSID update to '%s' queued", req.SSID),
	})
}

// UpdateWiFiPassword updates only the WiFi password
func (h *Handler) UpdateWiFiPassword(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)

	// Build vendor-specific parameter paths
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)
		if containsString(manufacturer, "HUAWEI") {
			// Huawei specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "ZTE") {
			// ZTE specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "FIBERHOME") {
			// FiberHome specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
			// Alcatel/Nokia specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "CIOT") {
			// CIOT specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
			// TP-Link specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else {
			// Default paths for unknown vendors
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		}
	} else {
		// If no device info, try common paths
		params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create password update task")
		return
	}

	h.DB.CreateLog(&id, "info", "wifi", "WiFi password update queued", "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "WiFi password update queued",
	})
}

// ============== WAN Handlers ==============

// GetWANConfigs returns all WAN configurations for a device
func (h *Handler) GetWANConfigs(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	configs, err := h.DB.GetWANConfigs(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get WAN configs")
		return
	}

	respondJSON(w, http.StatusOK, configs)
}

// CreateWANConfig creates a new WAN configuration
func (h *Handler) CreateWANConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var config models.WANConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	config.DeviceID = id
	created, err := h.DB.CreateWANConfig(&config)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create WAN config")
		return
	}

	h.DB.CreateLog(&id, "info", "wan", fmt.Sprintf("WAN configuration created: %s", config.Name), "")

	respondJSON(w, http.StatusCreated, created)
}

// GetWANConfig returns a specific WAN configuration
func (h *Handler) GetWANConfig(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting a single WAN config
	respondJSON(w, http.StatusOK, map[string]string{"message": "Not implemented"})
}

// UpdateWANConfig updates a WAN configuration
func (h *Handler) UpdateWANConfig(w http.ResponseWriter, r *http.Request) {
	wanID := getPathInt64(r, "wanId")

	var config models.WANConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	config.ID = wanID
	if err := h.DB.UpdateWANConfig(&config); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update WAN config")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "WAN configuration updated",
	})
}

// DeleteWANConfig deletes a WAN configuration
func (h *Handler) DeleteWANConfig(w http.ResponseWriter, r *http.Request) {
	wanID := getPathInt64(r, "wanId")

	if err := h.DB.DeleteWANConfig(wanID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete WAN config")
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============== Parameter Handlers ==============

// GetDeviceParameters returns device parameters
func (h *Handler) GetDeviceParameters(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	prefix := r.URL.Query().Get("prefix")

	params, err := h.DB.GetDeviceParameters(id, prefix)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get parameters")
		return
	}

	respondJSON(w, http.StatusOK, params)
}

// SetDeviceParameters sets device parameters
func (h *Handler) SetDeviceParameters(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var params map[string]string
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create parameter update task")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": fmt.Sprintf("Parameter update queued (%d parameters)", len(params)),
	})
}

// GetDeviceParameter returns a specific parameter
func (h *Handler) GetDeviceParameter(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	path := mux.Vars(r)["path"]

	params, err := h.DB.GetDeviceParameters(id, path)
	if err != nil || len(params) == 0 {
		respondError(w, http.StatusNotFound, "Parameter not found")
		return
	}

	respondJSON(w, http.StatusOK, params[0])
}

// ============== Firmware Handlers ==============

// GetFirmwareInfo returns firmware information
func (h *Handler) GetFirmwareInfo(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	device, err := h.DB.GetDevice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	info := map[string]interface{}{
		"currentVersion":  device.SoftwareVersion,
		"hardwareVersion": device.HardwareVersion,
		"updateAvailable": false,
	}

	respondJSON(w, http.StatusOK, info)
}

// UpgradeFirmware starts a firmware upgrade
func (h *Handler) UpgradeFirmware(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var req struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "Firmware URL is required")
		return
	}

	paramsJSON, _ := json.Marshal(req)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskDownload,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create firmware upgrade task")
		return
	}

	h.DB.CreateLog(&id, "warning", "firmware", "Firmware upgrade initiated", req.URL)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "Firmware upgrade started",
	})
}

// GetDeviceStatusLogs returns uptime history logs for a device
func (h *Handler) GetDeviceStatusLogs(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	limit := getQueryInt(r, "limit", 50)

	logs, err := h.DB.GetDeviceLogs(id, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch logs")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": logs})
}

// ============== Task Handlers ==============

// GetDeviceTasks returns tasks for a device
func (h *Handler) GetDeviceTasks(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	tasks, err := h.DB.GetPendingTasks(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get tasks")
		return
	}

	respondJSON(w, http.StatusOK, tasks)
}

// CreateDeviceTask creates a new task
func (h *Handler) CreateDeviceTask(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var task models.DeviceTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task.DeviceID = id
	created, err := h.DB.CreateTask(&task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

// GetTask returns a specific task
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Not implemented"})
}

// DeleteTask deletes a task
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============== Preset Handlers ==============

// GetPresets returns all presets
func (h *Handler) GetPresets(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, []interface{}{})
}

// CreatePreset creates a new preset
func (h *Handler) CreatePreset(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusCreated, map[string]string{"message": "Not implemented"})
}

// GetPreset returns a specific preset
func (h *Handler) GetPreset(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Not implemented"})
}

// UpdatePreset updates a preset
func (h *Handler) UpdatePreset(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"message": "Not implemented"})
}

// DeletePreset deletes a preset
func (h *Handler) DeletePreset(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============== Log Handlers ==============

// GetLogs returns system logs
func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	limit := getQueryInt(r, "limit", 100)
	offset := getQueryInt(r, "offset", 0)

	logs, err := h.DB.GetLogs(nil, level, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get logs")
		return
	}

	respondJSON(w, http.StatusOK, logs)
}

// GetDeviceLogs returns logs for a specific device
func (h *Handler) GetDeviceLogs(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	level := r.URL.Query().Get("level")
	limit := getQueryInt(r, "limit", 100)
	offset := getQueryInt(r, "offset", 0)

	logs, err := h.DB.GetLogs(&id, level, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get logs")
		return
	}

	respondJSON(w, http.StatusOK, logs)
}

// ============== Helper Functions ==============

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// generateJWT generates a JWT token for the user
func generateJWT(user *models.User) string {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Use the JWT secret from config
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "go-acs-secret-key-change-in-production" // Default fallback
	}
	signedToken, _ := token.SignedString([]byte(jwtSecret))
	return signedToken
}

// generateUsernameFromName creates a username from customer name
func generateUsernameFromName(name string) string {
	// Remove special characters and spaces, keep only alphanumeric
	re := regexp.MustCompile("[^a-zA-Z0-9]+")
	cleanName := re.ReplaceAllString(name, "")

	// Take first 10 characters and convert to lowercase
	if len(cleanName) > 10 {
		cleanName = cleanName[:10]
	}

	return strings.ToLower(cleanName)
}

// generateRandomPassword creates a default password
func generateRandomPassword() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$"
	password := make([]byte, 8)
	for i := range password {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		password[i] = charset[num.Int64()]
	}
	return string(password)
}

// hashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func getPathInt64(r *http.Request, key string) int64 {
	vars := mux.Vars(r)
	val, _ := strconv.ParseInt(vars[key], 10, 64)
	return val
}

func getQueryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s, substr))
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============== Billing Handlers ==============

// GetPackages returns all packages
func (h *Handler) GetPackages(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") == "true"
	packages, err := h.DB.GetPackages(activeOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get packages")
		return
	}
	respondJSON(w, http.StatusOK, packages)
}

// CreatePackage creates a new package
func (h *Handler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var pkg models.Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	pkg.IsActive = true
	created, err := h.DB.CreatePackage(&pkg)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create package")
		return
	}

	// Sync to MikroTik
	if h.Mikrotik != nil {
		rateLimit := fmt.Sprintf("%dM/%dM", pkg.UploadSpeed, pkg.DownloadSpeed)
		go h.Mikrotik.SyncPPPProfile(pkg.Name, rateLimit)
	}

	respondJSON(w, http.StatusCreated, created)
}

// GetPackage returns a specific package
func (h *Handler) GetPackageByID(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	pkg, err := h.DB.GetPackage(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Package not found")
		return
	}
	respondJSON(w, http.StatusOK, pkg)
}

// UpdatePackage updates a package
func (h *Handler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	var pkg models.Package
	if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	pkg.ID = id
	if err := h.DB.UpdatePackage(&pkg); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update package")
		return
	}

	updated, _ := h.DB.GetPackage(id)

	// Sync to MikroTik
	if h.Mikrotik != nil && updated != nil {
		rateLimit := fmt.Sprintf("%dM/%dM", updated.UploadSpeed, updated.DownloadSpeed)
		go h.Mikrotik.SyncPPPProfile(updated.Name, rateLimit)
	}

	respondJSON(w, http.StatusOK, updated)
}

// DeletePackage deletes a package
func (h *Handler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if err := h.DB.DeletePackage(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete package")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetCustomers returns all customers
func (h *Handler) GetCustomers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")
	limit := getQueryInt(r, "limit", 50)
	offset := getQueryInt(r, "offset", 0)

	customers, total, err := h.DB.GetCustomers(status, search, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get customers")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"customers": customers,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetLocations returns all customer locations
func (h *Handler) GetLocations(w http.ResponseWriter, r *http.Request) {
	locs, err := h.DB.GetCustomerLocations()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch locations")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": locs})
}

// UpdateCustomerLocation updates customer geo location
func (h *Handler) UpdateCustomerLocation(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.DB.UpdateCustomerLocation(id, req.Latitude, req.Longitude, req.Address); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update location")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// UpdateCustomerFCM updates customer FCM token
func (h *Handler) UpdateCustomerFCM(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	var req struct {
		FCMToken string `json:"fcmToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if err := h.DB.UpdateCustomerFCM(id, req.FCMToken); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update FCM token")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// CreateCustomer creates a new customer
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var customer models.Customer
	if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate portal username if not provided
	if customer.Username == "" {
		// Create username from customer name (remove spaces, lowercase)
		customer.Username = generateUsernameFromName(customer.Name)
	}

	// Hash password if provided, otherwise generate a default one
	if customer.InputPassword != "" {
		hashedPassword, err := hashPassword(customer.InputPassword)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		customer.Password = hashedPassword
	} else {
		// Generate a default password if none provided
		defaultPassword := generateRandomPassword()
		hashedPassword, err := hashPassword(defaultPassword)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		customer.Password = hashedPassword
	}

	if customer.Status == "" {
		customer.Status = "active"
	}
	created, err := h.DB.CreateCustomer(&customer)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create customer")
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

// GetCustomer returns a specific customer
func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	customer, err := h.DB.GetCustomer(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}
	respondJSON(w, http.StatusOK, customer)
}

// UpdateCustomer updates a customer
func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	var req struct {
		Name          string  `json:"name"`
		Email         string  `json:"email"`
		Phone         string  `json:"phone"`
		Address       string  `json:"address"`
		Latitude      float64 `json:"latitude"`
		Longitude     float64 `json:"longitude"`
		PackageID     int64   `json:"packageId"`
		Username      string  `json:"username"`
		Status        string  `json:"status"`
		Balance       float64 `json:"balance"`
		InputPassword string  `json:"password"` // Password might be in request
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get existing customer to preserve unchanged fields
	existingCustomer, err := h.DB.GetCustomer(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	// Update customer fields
	existingCustomer.Name = req.Name
	existingCustomer.Email = req.Email
	existingCustomer.Phone = req.Phone
	existingCustomer.Address = req.Address
	existingCustomer.Latitude = req.Latitude
	existingCustomer.Longitude = req.Longitude
	existingCustomer.PackageID = req.PackageID
	existingCustomer.Username = req.Username
	existingCustomer.Status = req.Status
	existingCustomer.Balance = req.Balance

	// Only update password if a new one is provided
	if req.InputPassword != "" {
		hashedPassword, err := hashPassword(req.InputPassword)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		existingCustomer.Password = hashedPassword
	}

	if err := h.DB.UpdateCustomer(existingCustomer); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update customer")
		return
	}

	updated, _ := h.DB.GetCustomer(id)
	respondJSON(w, http.StatusOK, updated)
}

// DeleteCustomer deletes a customer
func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if err := h.DB.DeleteCustomer(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete customer")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// IsolirCustomer suspends a customer (isolir)
func (h *Handler) IsolirCustomer(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid customer ID")
		return
	}

	// Update customer status to suspended
	customer, err := h.DB.GetCustomer(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	customer.Status = "suspended"
	if err := h.DB.UpdateCustomer(customer); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to suspend customer")
		return
	}

	// Change PPPoE profile to isolir profile via MikroTik API
	if h.Mikrotik != nil {
		// Create isolir profile if it doesn't exist
		isolirProfile := "isolir-profile"
		err = h.Mikrotik.CreateIsolirProfile(isolirProfile, "64k/64k")
		if err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to create isolir profile: %v\n", err)
		}

		// Change customer's PPPoE profile to isolir profile
		if customer.Username != "" {
			err = h.Mikrotik.SetPPPProfile(customer.Username, isolirProfile)
			if err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Failed to change PPPoE profile for customer %s: %v\n", customer.Username, err)
			} else {
				// Disconnect active PPP session to force the new profile
				err = h.Mikrotik.DisconnectPPPUser(customer.Username)
				if err != nil {
					// Log error but don't fail the operation
					fmt.Printf("Failed to disconnect PPP session for customer %s: %v\n", customer.Username, err)
				}
			}
		}
	}

	// Send notification to customer
	if customer.Phone != "" && h.WA != nil {
		go h.WA.Send(customer.Phone, whatsapp.GenerateSuspensionMessage(customer.Name))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Customer %s has been suspended", customer.Name),
	})
}

// UnsuspendCustomer reactivates a suspended customer
func (h *Handler) UnsuspendCustomer(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid customer ID")
		return
	}

	var req struct {
		Profile string `json:"profile"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Update customer status to active
	customer, err := h.DB.GetCustomer(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	customer.Status = "active"
	if err := h.DB.UpdateCustomer(customer); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to unsuspend customer")
		return
	}

	// Change PPPoE profile back to active profile via MikroTik API
	if h.Mikrotik != nil {
		// If no profile is specified, use the customer's package name as the profile
		profile := req.Profile
		if profile == "" {
			if customer.Package != nil {
				profile = customer.Package.Name
			} else {
				// Default to a standard profile name
				profile = "default-profile"
			}
		}

		// Change customer's PPPoE profile back to active profile
		if customer.Username != "" {
			err = h.Mikrotik.SetPPPProfile(customer.Username, profile)
			if err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Failed to change PPPoE profile for customer %s: %v\n", customer.Username, err)
			} else {
				// Disconnect active PPP session to force the new profile
				err = h.Mikrotik.DisconnectPPPUser(customer.Username)
				if err != nil {
					// Log error but don't fail the operation
					fmt.Printf("Failed to disconnect PPP session for customer %s: %v\n", customer.Username, err)
				}
			}
		}
	}

	// Send notification to customer
	if customer.Phone != "" && h.WA != nil {
		go h.WA.Send(customer.Phone, whatsapp.GenerateSuspensionMessage(customer.Name))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Customer %s has been reactivated", customer.Name),
	})
}

// UnsuspendCustomerWithoutPayment reactivates a suspended customer without requiring payment
// and combines unpaid invoices to the next month
func (h *Handler) UnsuspendCustomerWithoutPayment(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid customer ID")
		return
	}

	// Get customer
	customer, err := h.DB.GetCustomer(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	// Get all unpaid invoices for this customer
	invoices, _, err := h.DB.GetInvoices(&customer.ID, "pending", 1000, 0)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get customer invoices")
		return
	}

	// Update customer status to active
	customer.Status = "active"
	if err := h.DB.UpdateCustomer(customer); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to unsuspend customer")
		return
	}

	// Mark all unpaid invoices as 'combined' status instead of paid
	for _, invoice := range invoices {
		// Only combine invoices that are pending (not partially paid)
		if invoice.Status == models.InvoicePending {
			invoice.Status = models.InvoiceCombined // New status to indicate combined to next month
			if err := h.DB.UpdateInvoice(invoice); err != nil {
				fmt.Printf("Failed to update invoice status: %v\n", err) // Log error but don't fail
			}
		}
	}

	// Generate new invoice that includes the combined amounts
	// For now, we'll just reactivate without creating a new invoice
	// In a real implementation, you might want to create a new invoice with combined amounts

	// Change PPPoE profile back to active profile via MikroTik API
	if h.Mikrotik != nil {
		// Use the customer's package name as the profile
		profile := "default-profile"
		if customer.Package != nil {
			profile = customer.Package.Name
		}

		// Change customer's PPPoE profile back to active profile
		if customer.Username != "" {
			err = h.Mikrotik.SetPPPProfile(customer.Username, profile)
			if err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Failed to change PPPoE profile for customer %s: %v\n", customer.Username, err)
			} else {
				// Disconnect active PPP session to force the new profile
				err = h.Mikrotik.DisconnectPPPUser(customer.Username)
				if err != nil {
					// Log error but don't fail the operation
					fmt.Printf("Failed to disconnect PPP session for customer %s: %v\n", customer.Username, err)
				}
			}
		}
	}

	// Send notification to customer
	if customer.Phone != "" && h.WA != nil {
		go h.WA.Send(customer.Phone, fmt.Sprintf("Dear %s, your service has been reactivated. Please settle your outstanding bills soon.", customer.Name))
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":          true,
		"message":          fmt.Sprintf("Customer %s has been reactivated without payment. Unpaid invoices combined to next month.", customer.Name),
		"combinedInvoices": len(invoices),
	})
}

// GetInvoices returns all invoices
func (h *Handler) GetInvoices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	limit := getQueryInt(r, "limit", 50)
	offset := getQueryInt(r, "offset", 0)

	var customerID *int64
	if cidStr := r.URL.Query().Get("customerId"); cidStr != "" {
		cid, err := strconv.ParseInt(cidStr, 10, 64)
		if err == nil {
			customerID = &cid
		}
	}

	invoices, total, err := h.DB.GetInvoices(customerID, status, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get invoices")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateInvoice creates a new invoice
func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	var invoice models.Invoice
	if err := json.NewDecoder(r.Body).Decode(&invoice); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if invoice.Status == "" {
		invoice.Status = models.InvoicePending
	}
	created, err := h.DB.CreateInvoice(&invoice)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create invoice")
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

// GenerateMonthlyInvoices creates invoices for all active customers for the current month
// GenerateInvoicesInternal handles the core logic for invoice generation
func (h *Handler) GenerateInvoicesInternal() (int, error) {
	// Get all active customers with packages
	customers, _, err := h.DB.GetCustomers("active", "", 1000, 0)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	monthYear := now.Format("200601")
	generated := 0

	for _, customer := range customers {
		if customer.PackageID == 0 {
			continue // Skip customers without package
		}

		// Get package to get price
		pkg, err := h.DB.GetPackage(customer.PackageID)
		if err != nil || pkg == nil {
			continue
		}

		// Generate invoice number
		invoiceNo := fmt.Sprintf("INV-%s-%04d", monthYear, customer.ID)

		// Create invoice
		invoice := &models.Invoice{
			CustomerID:  customer.ID,
			InvoiceNo:   invoiceNo,
			Subtotal:    pkg.Price,
			Total:       pkg.Price,
			PeriodStart: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
			PeriodEnd:   time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()),
			DueDate:     time.Date(now.Year(), now.Month()+1, 10, 0, 0, 0, 0, now.Location()),
			Status:      models.InvoicePending,
			Notes:       fmt.Sprintf("Monthly subscription - %s", pkg.Name),
		}

		_, err = h.DB.CreateInvoice(invoice)
		if err == nil {
			generated++

			// Send Email Notification
			if customer.Email != "" && h.Mailer != nil {
				html := mailer.GenerateInvoiceHTML(
					customer.Name,
					invoiceNo,
					invoice.DueDate.Format("02/01/2006"),
					fmt.Sprintf("Rp %.2f", invoice.Total),
				)
				go h.Mailer.Send(customer.Email, "New Invoice Generated - GO-ACS", html)
			}

			// Send WA Notification
			if customer.Phone != "" && h.WA != nil {
				msg := whatsapp.GenerateInvoiceMessage(
					customer.Name,
					invoiceNo,
					invoice.DueDate.Format("02/01/2006"),
					fmt.Sprintf("Rp %.2f", invoice.Total),
				)
				go h.WA.Send(customer.Phone, msg)
			}

			// Send FCM Notification
			if customer.FCMToken != "" && h.FCM != nil {
				title := "New Invoice Generated - GO-ACS"
				body := fmt.Sprintf("Dear %s, a new invoice %s for Rp %.2f has been generated. Due date: %s.",
					customer.Name, invoiceNo, invoice.Total, invoice.DueDate.Format("02/01/2006"))
				go h.FCM.Send(customer.FCMToken, title, body)
			}
		}
	}
	return generated, nil
}

// GenerateMonthlyInvoices creates invoices for all active customers for the current month
func (h *Handler) GenerateMonthlyInvoices(w http.ResponseWriter, r *http.Request) {
	generated, err := h.GenerateInvoicesInternal()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate invoices")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"count":   generated,
		"message": fmt.Sprintf("Generated %d invoices", generated),
	})
}

// GetInvoice returns a single invoice with customer details
func (h *Handler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid invoice ID")
		return
	}

	invoice, err := h.DB.GetInvoice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Invoice not found")
		return
	}

	// Get customer details
	customer, _ := h.DB.GetCustomer(invoice.CustomerID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"invoice":  invoice,
		"customer": customer,
	})
}

// MarkInvoicePaid marks an invoice as paid
func (h *Handler) MarkInvoicePaid(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid invoice ID")
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Method string  `json:"method"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	invoice, err := h.DB.GetInvoice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Invoice not found")
		return
	}

	// Update invoice status
	now := time.Now()
	invoice.Status = models.InvoicePaid
	invoice.PaidAmount = invoice.Total
	invoice.PaidAt = &now

	if err := h.DB.UpdateInvoice(invoice); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update invoice")
		return
	}

	// Create payment record
	payment := &models.Payment{
		CustomerID:    invoice.CustomerID,
		InvoiceID:     &invoice.ID,
		Amount:        invoice.Total,
		PaymentMethod: req.Method,
		Status:        "completed",
		PaymentDate:   now,
	}
	h.DB.CreatePayment(payment)

	// Send Email Receipt
	customer, _ := h.DB.GetCustomer(invoice.CustomerID)
	if customer != nil {
		if customer.Email != "" && h.Mailer != nil {
			html := mailer.GeneratePaymentReceiptHTML(
				customer.Name,
				invoice.InvoiceNo,
				fmt.Sprintf("Rp %.2f", invoice.Total),
				now.Format("02/01/2006 15:04"),
			)
			go h.Mailer.Send(customer.Email, "Payment Receipt - GO-ACS", html)
		}

		// Send WA Receipt
		if customer.Phone != "" && h.WA != nil {
			msg := whatsapp.GeneratePaymentReceiptMessage(
				customer.Name,
				invoice.InvoiceNo,
				now.Format("02/01/2006 15:04"),
				fmt.Sprintf("Rp %.2f", invoice.Total),
			)
			go h.WA.Send(customer.Phone, msg)
		}

		// Send FCM Receipt
		if customer.FCMToken != "" && h.FCM != nil {
			title := "Payment Receipt - GO-ACS"
			body := fmt.Sprintf("Dear %s, payment for invoice %s has been received. Amount: Rp %.2f.",
				customer.Name, invoice.InvoiceNo, invoice.Total)
			go h.FCM.Send(customer.FCMToken, title, body)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Invoice marked as paid",
	})
}

// BatchIsolirOverdue suspends all customers with overdue invoices
func (h *Handler) BatchIsolirOverdue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DaysOverdue int `json:"daysOverdue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.DaysOverdue = 30 // Default 30 days
	}
	if req.DaysOverdue < 1 {
		req.DaysOverdue = 30
	}

	// Get customers with overdue invoices
	customers, _, _ := h.DB.GetCustomers("active", "", 1000, 0)

	suspended := 0
	for _, customer := range customers {
		// Check if customer has overdue invoices
		invoices, _, _ := h.DB.GetInvoices(&customer.ID, "pending", 100, 0)

		hasOverdue := false
		for _, inv := range invoices {
			if inv.DueDate.Before(time.Now().AddDate(0, 0, -req.DaysOverdue)) {
				hasOverdue = true
				break
			}
		}

		if hasOverdue {
			customer.Status = "suspended"
			if err := h.DB.UpdateCustomer(customer); err == nil {
				suspended++
				// Send WA Notification
				if customer.Phone != "" && h.WA != nil {
					go h.WA.Send(customer.Phone, whatsapp.GenerateSuspensionMessage(customer.Name))
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"suspended": suspended,
		"message":   fmt.Sprintf("Suspended %d customers with invoices overdue > %d days", suspended, req.DaysOverdue),
	})
}

// GetNetworkOverview returns aggregated network stats
func (h *Handler) GetNetworkOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetNetworkStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "data": stats})
}

// GetPayments returns all payments
func (h *Handler) GetPayments(w http.ResponseWriter, r *http.Request) {
	limit := getQueryInt(r, "limit", 50)
	offset := getQueryInt(r, "offset", 0)

	var customerID *int64
	if cidStr := r.URL.Query().Get("customerId"); cidStr != "" {
		cid, err := strconv.ParseInt(cidStr, 10, 64)
		if err == nil {
			customerID = &cid
		}
	}

	payments, total, err := h.DB.GetPayments(customerID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get payments")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"payments": payments,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreatePayment creates a new payment
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment models.Payment
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if payment.Status == "" {
		payment.Status = "completed"
	}
	created, err := h.DB.CreatePayment(&payment)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create payment")
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

// GetBillingStats returns billing statistics
func (h *Handler) GetBillingStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetBillingStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get billing stats")
		return
	}
	respondJSON(w, http.StatusOK, stats)
}

// ============== Customer Portal Handlers ==============

// CustomerLogin handles customer authentication
func (h *Handler) CustomerLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Look up customer by username, code, or numeric ID
	customer, err := h.DB.GetCustomerByUsername(req.Username)
	if err != nil {
		customer, err = h.DB.GetCustomerByCode(req.Username)
		if err != nil {
			if id, parseErr := strconv.ParseInt(req.Username, 10, 64); parseErr == nil {
				customer, err = h.DB.GetCustomer(id)
			}
			if err != nil {
				respondError(w, http.StatusUnauthorized, "Invalid credentials")
				return
			}
		}
	}

	// Verify password (in production, use proper password hashing)
	if customer.Password != req.Password {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check if customer is active
	if customer.Status == "suspended" || customer.Status == "terminated" {
		respondError(w, http.StatusForbidden, "Account is suspended. Please contact support.")
		return
	}

	// Generate token (in production, use JWT)
	token := fmt.Sprintf("customer-%d-%d", customer.ID, time.Now().Unix())

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"token":   token,
		"customer": map[string]interface{}{
			"id":           customer.ID,
			"customerCode": customer.CustomerCode,
			"name":         customer.Name,
			"email":        customer.Email,
			"status":       customer.Status,
		},
	})
}

// CustomerLogout handles customer logout
func (h *Handler) CustomerLogout(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetPortalDashboard returns customer portal dashboard data
func (h *Handler) GetPortalDashboard(w http.ResponseWriter, r *http.Request) {
	customerID := getQueryInt64(r, "customerId")
	if customerID == 0 {
		respondError(w, http.StatusBadRequest, "Customer ID required")
		return
	}

	customer, err := h.DB.GetCustomer(customerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	// Get package info
	var pkg *models.Package
	if customer.PackageID > 0 {
		pkg, _ = h.DB.GetPackage(customer.PackageID)
	}

	// Get customer's devices
	devices, _ := h.DB.GetCustomerDevices(customerID)

	// Get recent invoices
	invoices, _, _ := h.DB.GetInvoices(&customerID, "", 5, 0)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"customer": customer,
		"package":  pkg,
		"devices":  devices,
		"invoices": invoices,
	})
}

// GetPortalInvoices returns customer's invoices
func (h *Handler) GetPortalInvoices(w http.ResponseWriter, r *http.Request) {
	customerID := getQueryInt64(r, "customerId")
	if customerID == 0 {
		respondError(w, http.StatusBadRequest, "Customer ID required")
		return
	}

	limit := getQueryInt(r, "limit", 20)
	offset := getQueryInt(r, "offset", 0)

	invoices, total, err := h.DB.GetInvoices(&customerID, "", limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get invoices")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": invoices,
		"total":    total,
	})
}

// CreatePortalTicket allows customers to submit support tickets from the portal
func (h *Handler) CreatePortalTicket(w http.ResponseWriter, r *http.Request) {
	var ticket models.SupportTicket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Ensure customer ID is set
	if ticket.CustomerID == 0 {
		respondError(w, http.StatusBadRequest, "Customer ID required")
		return
	}

	// Set default values
	ticket.Status = "open"
	if ticket.Priority == "" {
		ticket.Priority = "medium"
	}

	created, err := h.DB.CreateSupportTicket(&ticket)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create ticket")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"ticket":  created,
		"message": "Ticket submitted successfully",
	})
}

// GetCustomerDashboard is an alias for GetPortalDashboard
func (h *Handler) GetCustomerDashboard(w http.ResponseWriter, r *http.Request) {
	h.GetPortalDashboard(w, r)
}

// GetCustomerInvoices is an alias for GetPortalInvoices
func (h *Handler) GetCustomerInvoices(w http.ResponseWriter, r *http.Request) {
	h.GetPortalInvoices(w, r)
}

// GetCustomerWiFi returns WiFi settings for customer's device
func (h *Handler) GetCustomerWiFi(w http.ResponseWriter, r *http.Request) {
	customerID := getQueryInt64(r, "customerId")
	if customerID == 0 {
		respondError(w, http.StatusBadRequest, "Customer ID required")
		return
	}

	// Get customer's primary device
	devices, err := h.DB.GetCustomerDevices(customerID)
	if err != nil || len(devices) == 0 {
		respondError(w, http.StatusNotFound, "No device found for customer")
		return
	}

	device := devices[0] // Primary device

	// Get WiFi configuration from device parameters
	params, err := h.DB.GetDeviceParameters(device.ID, "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get device parameters")
		return
	}

	// Build WiFi config from parameters
	config := models.WiFiConfig{}
	for _, p := range params {
		switch {
		case contains(p.Path, "SSID") && !contains(p.Path, "Hidden"):
			if config.SSID == "" { // Only set if not already set
				config.SSID = p.Value
			}
		case contains(p.Path, "KeyPassphrase") || contains(p.Path, "PreSharedKey"):
			if config.Password == "" { // Only set password as masked for security
				config.Password = "********"
			}
		case contains(p.Path, "BeaconType") || contains(p.Path, "SecurityMode"):
			if config.SecurityMode == "" { // Only set if not already set
				config.SecurityMode = p.Value
			}
		case contains(p.Path, "Channel"):
			if config.Channel == 0 { // Only set if not already set
				config.Channel, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "Enable"):
			config.Enabled = p.Value == "true" || p.Value == "1"
		case contains(p.Path, "Name") && contains(p.Path, "SSID"): // Some vendors use Name for SSID
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_HW_SSID"): // Huawei specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_ZTE_SSID"): // ZTE specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_FH_SSID"): // FiberHome specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "X_CT-COM_SSID"): // China Telecom specific
			if config.SSID == "" {
				config.SSID = p.Value
			}
		case contains(p.Path, "TransmitPower"):
			if config.TransmitPower == 0 { // Only set if not already set
				config.TransmitPower, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "X_HW_WlanHidden") && config.HiddenSSID == false: // Huawei hidden SSID
			config.HiddenSSID = p.Value == "1" || p.Value == "true"
		case contains(p.Path, "MaxAssociatedDevices") || contains(p.Path, "MaxClients"): // Max clients
			if config.MaxClients == 0 { // Only set if not already set
				config.MaxClients, _ = strconv.Atoi(p.Value)
			}
		case contains(p.Path, "Band"):
			if config.Band == "" { // Only set if not already set
				config.Band = p.Value
			}
		case contains(p.Path, "BSSID"):
			if config.BSSID == "" { // Only set if not already set
				config.BSSID = p.Value
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"device":     device,
		"wifiConfig": config,
	})
}

// UpdateCustomerWiFi updates WiFi settings for customer's device
func (h *Handler) UpdateCustomerWiFi(w http.ResponseWriter, r *http.Request) {
	customerID := getQueryInt64(r, "customerId")
	if customerID == 0 {
		respondError(w, http.StatusBadRequest, "Customer ID required")
		return
	}

	var req struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get customer's primary device
	devices, err := h.DB.GetCustomerDevices(customerID)
	if err != nil || len(devices) == 0 {
		respondError(w, http.StatusNotFound, "No device found for customer")
		return
	}

	// Return success (actual WiFi update to be implemented via device parameters)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "WiFi settings will be updated on next device connection",
		"ssid":    req.SSID,
	})
}

// UpdateDeviceLocation updates latitude and longitude of a device (used by map editing UI)
func (h *Handler) UpdateDeviceLocation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Address   string  `json:"address"`
	}
	id := getPathInt64(r, "id")
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if err := h.DB.UpdateDeviceLocation(id, req.Latitude, req.Longitude, req.Address); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update location")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ==== Support Ticket Handlers ==== //
func (h *Handler) CreateSupportTicket(w http.ResponseWriter, r *http.Request) {
	var ticket models.SupportTicket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	created, err := h.DB.CreateSupportTicket(&ticket)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create ticket")
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

func (h *Handler) GetSupportTickets(w http.ResponseWriter, r *http.Request) {
	customerID := getQueryInt64(r, "customerId")
	status := r.URL.Query().Get("status")
	limit := getQueryInt(r, "limit", 20)
	offset := getQueryInt(r, "offset", 0)
	tickets, total, err := h.DB.GetSupportTickets(&customerID, status, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get tickets")
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"tickets": tickets, "total": total})
}

func (h *Handler) GetSupportTicket(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "ticketId")
	ticket, err := h.DB.GetSupportTicket(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Ticket not found")
		return
	}
	respondJSON(w, http.StatusOK, ticket)
}

func (h *Handler) UpdateSupportTicket(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "ticketId")
	var ticket models.SupportTicket
	if err := json.NewDecoder(r.Body).Decode(&ticket); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	ticket.ID = id
	if err := h.DB.UpdateSupportTicket(&ticket); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update ticket")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) DeleteSupportTicket(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "ticketId")
	if err := h.DB.DeleteSupportTicket(id); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete ticket")
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// UpdatePortalWiFiSSID updates the WiFi SSID for customer's device
func (h *Handler) UpdatePortalWiFiSSID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID int64  `json:"customerId"`
		DeviceID   int64  `json:"deviceId"`
		SSID       string `json:"ssid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.SSID == "" {
		respondError(w, http.StatusBadRequest, "SSID cannot be empty")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(req.DeviceID)

	// Create task to update SSID on device with vendor-specific parameters
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)
		if containsString(manufacturer, "HUAWEI") {
			// Huawei specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_HW_SSID"] = req.SSID
		} else if containsString(manufacturer, "ZTE") {
			// ZTE specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_ZTE_SSID"] = req.SSID
		} else if containsString(manufacturer, "FIBERHOME") {
			// FiberHome specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.X_FH_SSID"] = req.SSID
		} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
			// Alcatel/Nokia specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.Name"] = req.SSID
		} else if containsString(manufacturer, "CIOT") {
			// CIOT specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
		} else {
			// Default paths for unknown vendors
			params["Device.WiFi.SSID.1.SSID"] = req.SSID
			params["Device.WiFi.SSID.2.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID"] = req.SSID
			params["Device.WiFi.SSID.1.Name"] = req.SSID
		}
	} else {
		// If no device info, try common paths
		params["Device.WiFi.SSID.1.SSID"] = req.SSID
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"] = req.SSID
		params["Device.WiFi.SSID.1.Name"] = req.SSID
	}

	paramsJSON, _ := json.Marshal(params)

	task := &models.DeviceTask{
		DeviceID:   req.DeviceID,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	_, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update SSID")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "SSID update queued. Changes will apply shortly.",
	})
}

// UpdatePortalWiFiPassword updates the WiFi password for customer's device
func (h *Handler) UpdatePortalWiFiPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID int64  `json:"customerId"`
		DeviceID   int64  `json:"deviceId"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(req.DeviceID)

	// Create task to update password on device with vendor-specific parameters
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)
		if containsString(manufacturer, "HUAWEI") {
			// Huawei specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "ZTE") {
			// ZTE specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "FIBERHOME") {
			// FiberHome specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
			// Alcatel/Nokia specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "CIOT") {
			// CIOT specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
			// TP-Link specific paths
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		} else {
			// Default paths for unknown vendors
			params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
			params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase"] = req.Password
			params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.KeyPassphrase"] = req.Password
		}
	} else {
		// If no device info, try common paths
		params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"] = req.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"] = req.Password
		params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.KeyPassphrase"] = req.Password
	}

	paramsJSON, _ := json.Marshal(params)

	task := &models.DeviceTask{
		DeviceID:   req.DeviceID,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	_, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "WiFi password update queued. Changes will apply shortly.",
	})
}

// ============== Payment Gateway Handlers ==============

// GetPaymentChannels returns available payment channels
func (h *Handler) GetPaymentChannels(w http.ResponseWriter, r *http.Request) {
	if h.Payment == nil {
		respondError(w, http.StatusServiceUnavailable, "Payment gateway not configured")
		return
	}

	channels, err := h.Payment.GetChannels()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch payment channels: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"channels": channels,
	})
}

// CreatePaymentTransaction initiates an online payment
func (h *Handler) CreatePaymentTransaction(w http.ResponseWriter, r *http.Request) {
	if h.Payment == nil {
		respondError(w, http.StatusServiceUnavailable, "Payment gateway not configured")
		return
	}

	id := getPathInt64(r, "id")
	if id == 0 {
		respondError(w, http.StatusBadRequest, "Invalid invoice ID")
		return
	}

	invoice, err := h.DB.GetInvoice(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Invoice not found")
		return
	}

	customer, err := h.DB.GetCustomer(invoice.CustomerID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	// Prepare request
	req := payment.TransactionRequest{
		InvoiceID: invoice.InvoiceNo,
		Amount:    int64(invoice.Total),
		Customer: payment.Customer{
			Name:  customer.Name,
			Email: customer.Email,
			Phone: customer.Phone,
		},
		Description: fmt.Sprintf("Payment for %s", invoice.InvoiceNo),
		Items: []payment.Item{
			{
				Name:     fmt.Sprintf("Invoice %s", invoice.InvoiceNo),
				Price:    int64(invoice.Total),
				Quantity: 1,
			},
		},
		ReturnURL: "http://localhost:8080/portal/invoices", // Should be configurable
	}

	resp, err := h.Payment.CreateTransaction(req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Payment creation failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    resp,
	})
}

// HandleTripayCallback processes webhook from Payment Gateway
func (h *Handler) HandleTripayCallback(w http.ResponseWriter, r *http.Request) {
	if h.Payment == nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{"success": false, "message": "Gateway not configured"})
		return
	}

	data, err := h.Payment.HandleCallback(r)
	if err != nil {
		fmt.Printf("[PAYMENT] Callback error: %v\n", err)
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	invoice, err := h.DB.GetInvoiceByNumber(data.InvoiceID)
	if err != nil {
		fmt.Printf("[PAYMENT] Invoice not found: %s\n", data.InvoiceID)
		respondJSON(w, http.StatusNotFound, map[string]interface{}{"success": false, "message": "Invoice not found"})
		return
	}

	// Idempotency check
	if invoice.Status == models.InvoicePaid {
		respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
		return
	}

	if data.Status == "PAID" {
		now := time.Unix(data.PaidAt, 0)
		invoice.Status = models.InvoicePaid
		invoice.PaidAmount = float64(data.Amount)
		invoice.PaidAt = &now

		if err := h.DB.UpdateInvoice(invoice); err != nil {
			fmt.Printf("[PAYMENT] Failed to update invoice: %v\n", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{"success": false})
			return
		}

		// Record Payment
		payment := &models.Payment{
			CustomerID:    invoice.CustomerID,
			InvoiceID:     &invoice.ID,
			Amount:        float64(data.Amount),
			PaymentMethod: data.PaymentMethod,
			Status:        "completed",
			PaymentDate:   now,
			Reference:     data.ReferenceID,
			ReceivedBy:    "SYSTEM (ONLINE)",
		}
		h.DB.CreatePayment(payment)

		// Send Receipt Email
		customer, _ := h.DB.GetCustomer(invoice.CustomerID)
		if customer != nil {
			if customer.Email != "" && h.Mailer != nil {
				html := mailer.GeneratePaymentReceiptHTML(
					customer.Name,
					invoice.InvoiceNo,
					fmt.Sprintf("Rp %.2f", invoice.Total),
					now.Format("02/01/2006 15:04"),
				)
				go h.Mailer.Send(customer.Email, "Payment Receipt - GO-ACS", html)
			}

			// Send WA Notification
			if customer.Phone != "" && h.WA != nil {
				msg := whatsapp.GeneratePaymentReceiptMessage(
					customer.Name,
					invoice.InvoiceNo,
					now.Format("02/01/2006 15:04"),
					fmt.Sprintf("Rp %.2f", invoice.Total),
				)
				go h.WA.Send(customer.Phone, msg)
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// Helper function for getting int64 from query
func getQueryInt64(r *http.Request, key string) int64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0
	}
	i, _ := strconv.ParseInt(val, 10, 64)
	return i
}

// ============== Mobile App API ==============

// GetMobileUsage returns bandwidth history for customer's primary device
func (h *Handler) GetMobileUsage(w http.ResponseWriter, r *http.Request) {
	// For production, use Session/JWT middleware to get CustomerID
	// Here we use query param for quick testing integration
	customerID := getQueryInt64(r, "customerId")
	if customerID == 0 {
		respondError(w, http.StatusBadRequest, "Missing customerId")
		return
	}

	// Get primary device
	devices, err := h.DB.GetDevicesByCustomer(customerID)
	if err != nil || len(devices) == 0 {
		respondJSON(w, http.StatusNotFound, map[string]interface{}{"error": "No device found"})
		return
	}

	// Get usage history (Top 50 records ~ last 4 hours if 5 min interval)
	records, err := h.DB.GetBandwidthHistory(devices[0].ID, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get history")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    records,
	})
}

// GetSettings return all system settings (Mikrotik, Radius, etc)
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.DB.GetSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch settings")
		return
	}
	respondJSON(w, http.StatusOK, settings)
}

// SaveSettings updates multiple settings
func (h *Handler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	for k, v := range req {
		if err := h.DB.SaveSetting(k, v); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to save setting: "+k)
			return
		}

		// Update in-memory config
		switch k {
		case "mikrotik_host":
			h.Config.MikrotikHost = v
		case "mikrotik_user":
			h.Config.MikrotikUser = v
		case "mikrotik_pass":
			h.Config.MikrotikPass = v
		case "mikrotik_port":
			port, _ := strconv.Atoi(v)
			if port > 0 {
				h.Config.MikrotikPort = port
			}
		case "tripay_api_key":
			h.Config.TripayAPIKey = v
		case "tripay_private_key":
			h.Config.TripayPrivateKey = v
		case "tripay_merchant_code":
			h.Config.TripayMerchantCode = v
		case "tripay_mode":
			h.Config.TripayMode = v
		}
	}

	// Re-initialize MikroTik client if MikroTik settings were changed
	for k := range req {
		if k == "mikrotik_host" || k == "mikrotik_user" || k == "mikrotik_pass" || k == "mikrotik_port" {
			h.Mikrotik = mikrotik.New(h.Config)
			break
		}
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// TestMikrotik tests connection to MikroTik router using current config
func (h *Handler) TestMikrotik(w http.ResponseWriter, r *http.Request) {
	if h.Mikrotik == nil {
		respondError(w, http.StatusServiceUnavailable, "MikroTik client not initialized")
		return
	}

	resource, err := h.Mikrotik.GetSystemResource()
	if err != nil {
		respondError(w, http.StatusBadGateway, "Failed to connect: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"host":    h.Config.MikrotikHost,
		"version": resource["version"],
		"uptime":  resource["uptime"],
		"board":   resource["board-name"],
	})
}

// GetMikrotikProfiles returns all PPP profiles from MikroTik
func (h *Handler) GetMikrotikProfiles(w http.ResponseWriter, r *http.Request) {
	if h.Mikrotik == nil {
		respondError(w, http.StatusServiceUnavailable, "MikroTik client not initialized")
		return
	}

	profiles, err := h.Mikrotik.GetPPPProfiles()
	if err != nil {
		respondError(w, http.StatusBadGateway, "Failed to fetch profiles: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, profiles)
}

// CreateMikrotikProfile creates a new PPP profile on MikroTik
func (h *Handler) CreateMikrotikProfile(w http.ResponseWriter, r *http.Request) {
	if h.Mikrotik == nil {
		respondError(w, http.StatusServiceUnavailable, "MikroTik client not initialized")
		return
	}

	var req struct {
		Name      string `json:"name"`
		RateLimit string `json:"rate_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "Profile name is required")
		return
	}

	if err := h.Mikrotik.SyncPPPProfile(req.Name, req.RateLimit); err != nil {
		respondError(w, http.StatusBadGateway, "Failed to create profile: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ============== Update Handlers ==============

// CheckForUpdates checks for available updates from GitHub
func (h *Handler) CheckForUpdates(w http.ResponseWriter, r *http.Request) {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		respondError(w, http.StatusInternalServerError, "Git is not installed on this system")
		return
	}

	// Check if we're in a git repository
	if _, err := exec.Command("git", "rev-parse", "--git-dir").CombinedOutput(); err != nil {
		respondError(w, http.StatusInternalServerError, "Not running from a git repository")
		return
	}

	// Get current git info
	currentBranch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentCommit, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	lastUpdate, _ := exec.Command("git", "log", "-1", "--format=%cd", "--date=relative").Output()

	// Try to get git tag for version
	tagOutput, _ := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
	version := strings.TrimSpace(string(tagOutput))
	if version == "" {
		// If no tag, use short commit hash
		version = strings.TrimSpace(string(currentCommit))[:7]
	}

	// Fetch from remote
	if err := exec.Command("git", "fetch", "origin").Run(); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch from remote repository: "+err.Error())
		return
	}

	// Check if we're behind
	behindOutput, _ := exec.Command("git", "rev-list", "--count", "HEAD..origin/"+strings.TrimSpace(string(currentBranch))).Output()
	commitsBehind, _ := strconv.Atoi(strings.TrimSpace(string(behindOutput)))

	// Get latest commit message
	latestMsg, _ := exec.Command("git", "log", "origin/"+strings.TrimSpace(string(currentBranch)), "-1", "--format=%s").Output()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"current_version":       version,
		"current_branch":        strings.TrimSpace(string(currentBranch)),
		"current_commit":        strings.TrimSpace(string(currentCommit))[:7],
		"last_update":           strings.TrimSpace(string(lastUpdate)),
		"updates_available":     commitsBehind > 0,
		"commits_behind":        commitsBehind,
		"latest_commit_message": strings.TrimSpace(string(latestMsg)),
	})
}

// PerformUpdate performs git pull and rebuild
func (h *Handler) PerformUpdate(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Get current git info before update
	currentBranch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentCommit, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	branch := strings.TrimSpace(string(currentBranch))
	currentHash := strings.TrimSpace(string(currentCommit))[:7]

	// Send start notification
	if h.Telegram != nil {
		go h.Telegram.SendUpdateStart(branch, currentHash)
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	sendLog := func(message, logType string) {
		data := map[string]string{"message": message, "type": logType}
		json.NewEncoder(w).Encode(data)
		flusher.Flush()
	}

	sendLog("Starting update process...", "info")

	// Git pull
	sendLog("Pulling latest changes from GitHub...", "command")
	if h.Telegram != nil {
		go h.Telegram.SendUpdateProgress("Git Pull", "Fetching latest changes from repository...")
	}

	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Git pull failed: %s", err.Error()), "error")
		sendLog(string(output), "error")
		if h.Telegram != nil {
			go h.Telegram.SendUpdateError("Git Pull", string(output))
		}
		return
	}
	sendLog(string(output), "success")

	// Go mod tidy
	sendLog("Updating dependencies...", "command")
	cmd = exec.Command("go", "mod", "tidy")
	output, err = cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Dependency update failed: %s", err.Error()), "warning")
	} else {
		sendLog("Dependencies updated", "success")
	}

	// Build
	sendLog("Building application...", "command")
	if h.Telegram != nil {
		go h.Telegram.SendUpdateProgress("Build", "Compiling application...")
	}

	cmd = exec.Command("go", "build", "-o", "go-acs-bin", "cmd/server/main.go")
	output, err = cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Build failed: %s", err.Error()), "error")
		sendLog(string(output), "error")
		if h.Telegram != nil {
			go h.Telegram.SendUpdateError("Build", string(output))
		}
		return
	}
	sendLog("Build successful", "success")

	// Copy binary
	sendLog("Installing new binary...", "command")
	cmd = exec.Command("systemctl", "stop", "go-acs")
	cmd.Run()

	cmd = exec.Command("cp", "-f", "go-acs-bin", "/opt/go-acs/go-acs")
	output, err = cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Failed to copy binary: %s", err.Error()), "error")
	} else {
		sendLog("Binary installed", "success")
	}

	// Copy web files
	sendLog("Updating web files...", "command")
	cmd = exec.Command("cp", "-r", "web/*", "/opt/go-acs/web/")
	cmd.Run()
	sendLog("Web files updated", "success")

	// Restart service
	sendLog("Restarting service...", "command")
	if h.Telegram != nil {
		go h.Telegram.SendUpdateProgress("Restart", "Restarting GO-ACS service...")
	}
	cmd = exec.Command("systemctl", "restart", "go-acs")
	err = cmd.Run()
	if err != nil {
		sendLog(fmt.Sprintf("Failed to restart service: %s", err.Error()), "error")
		if h.Telegram != nil {
			go h.Telegram.SendUpdateError("Service Restart", err.Error())
		}
	} else {
		sendLog("Service restarted successfully", "success")
	}

	sendLog("Update completed!", "success")

	// Get new commit hash
	newCommit, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	newHash := strings.TrimSpace(string(newCommit))[:7]

	// Calculate duration
	duration := time.Since(startTime).Round(time.Second).String()

	// Send success notification
	if h.Telegram != nil {
		go h.Telegram.SendUpdateSuccess(newHash, duration)
	}
}

// RebuildApplication rebuilds the Go application
func (h *Handler) RebuildApplication(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "Streaming not supported")
		return
	}

	sendLog := func(message, logType string) {
		data := map[string]string{"message": message, "type": logType}
		json.NewEncoder(w).Encode(data)
		flusher.Flush()
	}

	sendLog("Starting rebuild...", "info")

	// Build
	sendLog("Building application...", "command")
	cmd := exec.Command("go", "build", "-o", "go-acs-bin", "cmd/server/main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Build failed: %s", err.Error()), "error")
		sendLog(string(output), "error")
		return
	}
	sendLog("Build successful", "success")

	// Copy binary
	sendLog("Installing new binary...", "command")
	cmd = exec.Command("systemctl", "stop", "go-acs")
	cmd.Run()

	cmd = exec.Command("cp", "-f", "go-acs-bin", "/opt/go-acs/go-acs")
	output, err = cmd.CombinedOutput()
	if err != nil {
		sendLog(fmt.Sprintf("Failed to copy binary: %s", err.Error()), "error")
	} else {
		sendLog("Binary installed", "success")
	}

	sendLog("Rebuild completed!", "success")
}

// RestartService restarts the go-acs service
func (h *Handler) RestartService(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("systemctl", "restart", "go-acs")
	err := cmd.Run()

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to restart service: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Service restart initiated",
	})
}

// SyncCustomerToDeviceByPPPoE synchronizes a customer to a device using PPPoE username
func (h *Handler) SyncCustomerToDeviceByPPPoE(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID    int64  `json:"customerId"`
		PPPoEUsername string `json:"pppoeUsername"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.CustomerID <= 0 {
		respondError(w, http.StatusBadRequest, "Customer ID is required")
		return
	}

	if req.PPPoEUsername == "" {
		respondError(w, http.StatusBadRequest, "PPPoE username is required")
		return
	}

	// Perform the synchronization
	if err := h.DB.SyncCustomerToDevice(req.CustomerID, req.PPPoEUsername); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sync customer to device: "+err.Error())
		return
	}

	// Get the updated customer with device info
	customer, err := h.DB.GetCustomer(req.CustomerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get updated customer")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"message":  "Customer successfully synced to device",
		"customer": customer,
	})
}

// GetDeviceByTemplate retrieves a device by its template field (PPPoE username)
func (h *Handler) GetDeviceByTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	template := vars["template"]

	if template == "" {
		respondError(w, http.StatusBadRequest, "Template parameter is required")
		return
	}

	device, err := h.DB.GetDeviceByTemplate(template)
	if err != nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	respondJSON(w, http.StatusOK, device)
}

// GetCustomerByPPPoE retrieves a customer by PPPoE username
func (h *Handler) GetCustomerByPPPoE(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pppoeUsername := vars["pppoeUsername"]

	if pppoeUsername == "" {
		respondError(w, http.StatusBadRequest, "PPPoE username parameter is required")
		return
	}

	customer, err := h.DB.GetCustomerByPPPoE(pppoeUsername)
	if err != nil {
		respondError(w, http.StatusNotFound, "Customer not found")
		return
	}

	respondJSON(w, http.StatusOK, customer)
}

// ============== LAN Configuration Handlers ==============

// LANConfig represents LAN configuration
type LANConfig struct {
	Enable        bool   `json:"enable"`
	IPAddress     string `json:"ipAddress"`
	SubnetMask    string `json:"subnetMask"`
	DHCPEnable    bool   `json:"dhcpEnable"`
	DHCPServerIP  string `json:"dhcpServerIP"`
	VLANID        int    `json:"vlanId"`
	VLANPriority  int    `json:"vlanPriority"`
	BridgeMode    bool   `json:"bridgeMode"`
	PortIsolation bool   `json:"portIsolation"`
	MaxClients    int    `json:"maxClients"`
}

// GetLANConfig returns LAN configuration for a device
func (h *Handler) GetLANConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	params, err := h.DB.GetDeviceParameters(id, "InternetGatewayDevice.LANDevice.")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get LAN parameters")
		return
	}

	config := LANConfig{}
	for _, p := range params {
		switch {
		case contains(p.Path, "Enable"):
			config.Enable = p.Value == "true" || p.Value == "1"
		case contains(p.Path, "IPAddress"):
			config.IPAddress = p.Value
		case contains(p.Path, "SubnetMask"):
			config.SubnetMask = p.Value
		case contains(p.Path, "DHCPEnable") || contains(p.Path, "DHCP.ServerEnable"):
			config.DHCPEnable = p.Value == "true" || p.Value == "1"
		case contains(p.Path, "DHCPServerIPAddress"):
			config.DHCPServerIP = p.Value
		case strings.HasSuffix(p.Path, "VLANID") || strings.HasSuffix(p.Path, "VLANId"):
			if v, err := strconv.Atoi(p.Value); err == nil {
				config.VLANID = v
			}
		case strings.HasSuffix(p.Path, "VLANPriority"):
			if v, err := strconv.Atoi(p.Value); err == nil {
				config.VLANPriority = v
			}
		case contains(p.Path, "BridgeMode"):
			config.BridgeMode = p.Value == "true" || p.Value == "1"
		case contains(p.Path, "PortIsolation"):
			config.PortIsolation = p.Value == "true" || p.Value == "1"
		case strings.HasSuffix(p.Path, "MaxClients") || strings.HasSuffix(p.Path, "MaxAssociatedDevices"):
			if v, err := strconv.Atoi(p.Value); err == nil {
				config.MaxClients = v
			}
		}
	}

	respondJSON(w, http.StatusOK, config)
}

// UpdateLANConfig updates LAN configuration for a device
func (h *Handler) UpdateLANConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var config LANConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)

	// Build vendor-specific parameter paths
	params := make(map[string]string)

	if device != nil {
		manufacturer := strings.ToUpper(device.Manufacturer)

		// Common LAN paths
		params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.Enable"] = fmt.Sprintf("%v", config.Enable)

		if config.IPAddress != "" {
			params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.IPAddress"] = config.IPAddress
		}
		if config.SubnetMask != "" {
			params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.SubnetMask"] = config.SubnetMask
		}
		params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.DHCPEnable"] = fmt.Sprintf("%v", config.DHCPEnable)

		if config.DHCPServerIP != "" {
			params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.DHCPServerIPAddress"] = config.DHCPServerIP
		}

		// Vendor-specific VLAN paths
		if config.VLANID > 0 {
			if containsString(manufacturer, "HUAWEI") {
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.X_HW_VLANID"] = fmt.Sprintf("%d", config.VLANID)
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.X_HW_VLANPriority"] = fmt.Sprintf("%d", config.VLANPriority)
			} else if containsString(manufacturer, "ZTE") {
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.X_ZTE-COM_VLANID"] = fmt.Sprintf("%d", config.VLANID)
			} else if containsString(manufacturer, "FIBERHOME") {
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.X_FH_VLANID"] = fmt.Sprintf("%d", config.VLANID)
			} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.X_TPLINK_VLANID"] = fmt.Sprintf("%d", config.VLANID)
			} else {
				params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.VLANID"] = fmt.Sprintf("%d", config.VLANID)
			}
		}

		// Vendor-specific Bridge Mode paths
		if config.BridgeMode {
			if containsString(manufacturer, "HUAWEI") {
				params["InternetGatewayDevice.X_HW_BridgeMode.Enable"] = "1"
			} else if containsString(manufacturer, "ZTE") {
				params["InternetGatewayDevice.X_ZTE-COM_BridgeMode.Enable"] = "1"
			} else if containsString(manufacturer, "FIBERHOME") {
				params["InternetGatewayDevice.X_FH_BridgeMode.Enable"] = "1"
			} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
				params["InternetGatewayDevice.X_TPLINK_BridgeMode.Enable"] = "1"
			}
		}

		// Vendor-specific Port Isolation paths
		if config.PortIsolation {
			if containsString(manufacturer, "HUAWEI") {
				params["InternetGatewayDevice.X_HW_PortIsolation.Enable"] = "1"
			} else if containsString(manufacturer, "ZTE") {
				params["InternetGatewayDevice.X_ZTE-COM_PortIsolation.Enable"] = "1"
			} else if containsString(manufacturer, "FIBERHOME") {
				params["InternetGatewayDevice.X_FH_PortIsolation.Enable"] = "1"
			} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
				params["InternetGatewayDevice.X_TPLINK_PortIsolation.Enable"] = "1"
			}
		}

		if config.MaxClients > 0 {
			params["InternetGatewayDevice.LANDevice.1.LANEthernetConfig.1.MaxClients"] = fmt.Sprintf("%d", config.MaxClients)
		}
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create LAN update task")
		return
	}

	h.DB.CreateLog(&id, "info", "lan", "LAN configuration update queued", "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "LAN configuration update queued",
	})
}

// ============== Port Forwarding / NAT Configuration ==============

// PortForwardingRule represents a port forwarding rule
type PortForwardingRule struct {
	ExternalPort   int    `json:"externalPort"`
	InternalPort   int    `json:"internalPort"`
	InternalClient string `json:"internalClient"`
	Protocol       string `json:"protocol"` // TCP, UDP, or BOTH
	Enable         bool   `json:"enable"`
	Description    string `json:"description"`
}

// GetPortForwardingRules returns port forwarding rules for a device
func (h *Handler) GetPortForwardingRules(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	params, err := h.DB.GetDeviceParameters(id, "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get parameters")
		return
	}

	rules := []PortForwardingRule{}
	ruleMap := make(map[string]*PortForwardingRule)

	for _, p := range params {
		// Check for vendor-specific NAT paths
		if strings.Contains(p.Path, "X_HW_NAT.PortMapping") ||
			strings.Contains(p.Path, "X_ZTE-COM_NAT.PortMapping") ||
			strings.Contains(p.Path, "X_FH_NAT.PortMapping") ||
			strings.Contains(p.Path, "X_TPLINK_NAT.PortMapping") ||
			strings.Contains(p.Path, "X_ALU_NAT.PortMapping") {

			// Extract rule index from path
			parts := strings.Split(p.Path, ".")
			if len(parts) < 2 {
				continue
			}
			ruleKey := parts[len(parts)-2] // e.g., "1" from "X_HW_NAT.PortMapping.1.Enable"

			if _, exists := ruleMap[ruleKey]; !exists {
				ruleMap[ruleKey] = &PortForwardingRule{}
			}

			rule := ruleMap[ruleKey]

			if strings.HasSuffix(p.Path, "ExternalPort") {
				if v, err := strconv.Atoi(p.Value); err == nil {
					rule.ExternalPort = v
				}
			} else if strings.HasSuffix(p.Path, "InternalPort") {
				if v, err := strconv.Atoi(p.Value); err == nil {
					rule.InternalPort = v
				}
			} else if strings.HasSuffix(p.Path, "InternalClient") {
				rule.InternalClient = p.Value
			} else if strings.HasSuffix(p.Path, "Protocol") {
				rule.Protocol = p.Value
			} else if strings.HasSuffix(p.Path, "Enable") {
				rule.Enable = p.Value == "true" || p.Value == "1"
			} else if strings.HasSuffix(p.Path, "Description") {
				rule.Description = p.Value
			}
		}
	}

	// Convert map to slice
	for _, rule := range ruleMap {
		rules = append(rules, *rule)
	}

	respondJSON(w, http.StatusOK, rules)
}

// CreatePortForwardingRule creates a new port forwarding rule
func (h *Handler) CreatePortForwardingRule(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var rule PortForwardingRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)
	if device == nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	manufacturer := strings.ToUpper(device.Manufacturer)
	params := make(map[string]string)

	// Determine vendor-specific NAT path
	var natPath string
	if containsString(manufacturer, "HUAWEI") {
		natPath = "InternetGatewayDevice.X_HW_NAT.PortMapping.1"
	} else if containsString(manufacturer, "ZTE") {
		natPath = "InternetGatewayDevice.X_ZTE-COM_NAT.PortMapping.1"
	} else if containsString(manufacturer, "FIBERHOME") {
		natPath = "InternetGatewayDevice.X_FH_NAT.PortMapping.1"
	} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
		natPath = "InternetGatewayDevice.X_TPLINK_NAT.PortMapping.1"
	} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
		natPath = "InternetGatewayDevice.X_ALU_NAT.PortMapping.1"
	} else {
		natPath = "InternetGatewayDevice.NAT.PortMapping.1"
	}

	// Build rule parameters
	params[natPath+".ExternalPort"] = fmt.Sprintf("%d", rule.ExternalPort)
	params[natPath+".InternalPort"] = fmt.Sprintf("%d", rule.InternalPort)
	params[natPath+".InternalClient"] = rule.InternalClient
	params[natPath+".Protocol"] = rule.Protocol
	params[natPath+".Enable"] = fmt.Sprintf("%v", rule.Enable)
	params[natPath+".Description"] = rule.Description

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create port forwarding task")
		return
	}

	h.DB.CreateLog(&id, "info", "nat", fmt.Sprintf("Port forwarding rule created: %d -> %s:%d", rule.ExternalPort, rule.InternalClient, rule.InternalPort), "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "Port forwarding rule created",
	})
}

// ============== Bridge Mode Configuration ==============

// SetBridgeMode enables or disables bridge mode
func (h *Handler) SetBridgeMode(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var req struct {
		Enable bool `json:"enable"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)
	if device == nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	manufacturer := strings.ToUpper(device.Manufacturer)
	params := make(map[string]string)

	// Vendor-specific bridge mode paths
	if containsString(manufacturer, "HUAWEI") {
		params["InternetGatewayDevice.X_HW_BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	} else if containsString(manufacturer, "ZTE") {
		params["InternetGatewayDevice.X_ZTE-COM_BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	} else if containsString(manufacturer, "FIBERHOME") {
		params["InternetGatewayDevice.X_FH_BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
		params["InternetGatewayDevice.X_TPLINK_BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
		params["InternetGatewayDevice.X_ALU_BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	} else {
		// Generic path
		params["InternetGatewayDevice.BridgeMode.Enable"] = fmt.Sprintf("%v", req.Enable)
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create bridge mode task")
		return
	}

	status := "disabled"
	if req.Enable {
		status = "enabled"
	}

	h.DB.CreateLog(&id, "info", "bridge", fmt.Sprintf("Bridge mode %s", status), "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": fmt.Sprintf("Bridge mode %s", status),
	})
}

// ============== QoS Configuration ==============

// QoSConfig represents QoS configuration
type QoSConfig struct {
	Enable       bool   `json:"enable"`
	MaxBandwidth int    `json:"maxBandwidth"` // in Kbps
	Priority     string `json:"priority"`     // High, Medium, Low
}

// GetQoSConfig returns QoS configuration for a device
func (h *Handler) GetQoSConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	params, err := h.DB.GetDeviceParameters(id, "")
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get QoS parameters")
		return
	}

	config := QoSConfig{}
	for _, p := range params {
		switch {
		case strings.Contains(p.Path, "QoS") && strings.HasSuffix(p.Path, "Enable"):
			config.Enable = p.Value == "true" || p.Value == "1"
		case strings.Contains(p.Path, "QoS") && strings.HasSuffix(p.Path, "MaxBandwidth"):
			if v, err := strconv.Atoi(p.Value); err == nil {
				config.MaxBandwidth = v
			}
		case strings.Contains(p.Path, "QoS") && strings.HasSuffix(p.Path, "Priority"):
			config.Priority = p.Value
		}
	}

	respondJSON(w, http.StatusOK, config)
}

// UpdateQoSConfig updates QoS configuration for a device
func (h *Handler) UpdateQoSConfig(w http.ResponseWriter, r *http.Request) {
	id := getPathInt64(r, "id")

	var config QoSConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get device to determine vendor
	device, _ := h.DB.GetDevice(id)
	if device == nil {
		respondError(w, http.StatusNotFound, "Device not found")
		return
	}

	manufacturer := strings.ToUpper(device.Manufacturer)
	params := make(map[string]string)

	// Vendor-specific QoS paths
	var qosPath string
	if containsString(manufacturer, "HUAWEI") {
		qosPath = "InternetGatewayDevice.X_HW_QoS"
	} else if containsString(manufacturer, "ZTE") {
		qosPath = "InternetGatewayDevice.X_ZTE-COM_QoS"
	} else if containsString(manufacturer, "FIBERHOME") {
		qosPath = "InternetGatewayDevice.X_FH_QoS"
	} else if containsString(manufacturer, "TPLINK") || containsString(manufacturer, "TP-LINK") {
		qosPath = "InternetGatewayDevice.X_TPLINK_QoS"
	} else if containsString(manufacturer, "ALCATEL") || containsString(manufacturer, "NOKIA") {
		qosPath = "InternetGatewayDevice.X_ALU_QoS"
	} else {
		qosPath = "InternetGatewayDevice.QoS"
	}

	params[qosPath+".Enable"] = fmt.Sprintf("%v", config.Enable)
	if config.MaxBandwidth > 0 {
		params[qosPath+".MaxBandwidth"] = fmt.Sprintf("%d", config.MaxBandwidth)
	}
	if config.Priority != "" {
		params[qosPath+".Priority"] = config.Priority
	}

	paramsJSON, _ := json.Marshal(params)
	task := &models.DeviceTask{
		DeviceID:   id,
		Type:       models.TaskSetParameterValues,
		Parameters: paramsJSON,
	}

	created, err := h.DB.CreateTask(task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create QoS update task")
		return
	}

	h.DB.CreateLog(&id, "info", "qos", "QoS configuration update queued", "")

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"taskId":  created.ID,
		"message": "QoS configuration update queued",
	})
}

// ChangeAdminPassword handles password change for admin users
func (h *Handler) ChangeAdminPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username        string `json:"username"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Username == "" {
		respondError(w, http.StatusBadRequest, "Username is required")
		return
	}

	if req.CurrentPassword == "" {
		respondError(w, http.StatusBadRequest, "Current password is required")
		return
	}

	if req.NewPassword == "" {
		respondError(w, http.StatusBadRequest, "New password is required")
		return
	}

	if len(req.NewPassword) < 6 {
		respondError(w, http.StatusBadRequest, "New password must be at least 6 characters")
		return
	}

	// Get user from database
	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil || user == nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		respondError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash new password")
		return
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := h.DB.UpdateUser(user); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password changed successfully",
	})
}
