package models

import (
	"encoding/json"
	"time"
)

// Device represents an ONU/CPE device
type Device struct {
	ID                int64        `json:"id"`
	SerialNumber      string       `json:"serialNumber"`
	OUI               string       `json:"oui"` // Manufacturer OUI
	ProductClass      string       `json:"productClass"`
	Manufacturer      string       `json:"manufacturer"`
	ModelName         string       `json:"modelName"`
	HardwareVersion   string       `json:"hardwareVersion"`
	SoftwareVersion   string       `json:"softwareVersion"`
	ConnectionRequest string       `json:"connectionRequestUrl"`
	Status            DeviceStatus `json:"status"`
	LastInform        *time.Time   `json:"lastInform"`
	LastContact       *time.Time   `json:"lastContact"`
	IPAddress         string       `json:"ipAddress"`
	MACAddress        string       `json:"macAddress"`
	Uptime            int64        `json:"uptime"`
	RXPower           float64      `json:"rxPower"`
	TXPower           float64      `json:"txPower,omitempty"`
	OpticalTemperature float64     `json:"opticalTemperature,omitempty"`
	OpticalVoltage     float64     `json:"opticalVoltage,omitempty"`
	OpticalCurrent     float64     `json:"opticalCurrent,omitempty"`
	Distance           float64     `json:"distance,omitempty"`
	ClientCount       int          `json:"clientCount"`
	Template          string       `json:"template"`
	// PPPoE Information
	PPPoEUsername     string `json:"pppoeUsername,omitempty"`
	PPPoEIP           string `json:"pppoeIP,omitempty"`
	WANIP             string `json:"wanIP,omitempty"`
	WANConnectionType string `json:"wanConnectionType,omitempty"`
	// Additional ONU information
	Temperature float64 `json:"temperature,omitempty"`
	// Location fields for map
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
	// Customer relation
	CustomerID *int64            `json:"customerId,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Notes      string            `json:"notes"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// DeviceStatus represents the online/offline status
type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "online"
	StatusOffline DeviceStatus = "offline"
	StatusUnknown DeviceStatus = "unknown"
)

// DeviceParameter represents a TR-069 parameter
type DeviceParameter struct {
	ID        int64     `json:"id"`
	DeviceID  int64     `json:"deviceId"`
	Path      string    `json:"path"`
	Value     string    `json:"value"`
	Type      string    `json:"type"`
	Writable  bool      `json:"writable"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// WiFiConfig represents WiFi configuration
type WiFiConfig struct {
	SSID             string `json:"ssid"`
	Password         string `json:"password"`
	SecurityMode     string `json:"securityMode"` // WPA2-PSK, WPA3, etc.
	Channel          int    `json:"channel"`
	ChannelBandwidth string `json:"channelBandwidth"` // 20MHz, 40MHz, 80MHz
	Enabled          bool   `json:"enabled"`
	HiddenSSID       bool   `json:"hiddenSSID"`
	MaxClients       int    `json:"maxClients"`
	Band             string `json:"band"` // 2.4GHz, 5GHz
	TransmitPower    int    `json:"transmitPower"`
	BSSID            string `json:"bssid"`
	ConnectedClients int    `json:"connectedClients"`
}

// WANConfig represents WAN connection configuration
type WANConfig struct {
	ID             int64     `json:"id"`
	DeviceID       int64     `json:"deviceId"`
	Name           string    `json:"name"`
	ConnectionType string    `json:"connectionType"` // PPPoE, DHCP, Static
	VLAN           int       `json:"vlan"`
	Username       string    `json:"username,omitempty"`
	Password       string    `json:"password,omitempty"`
	IPAddress      string    `json:"ipAddress"`
	SubnetMask     string    `json:"subnetMask"`
	Gateway        string    `json:"gateway"`
	DNS1           string    `json:"dns1"`
	DNS2           string    `json:"dns2"`
	MTU            int       `json:"mtu"`
	Enabled        bool      `json:"enabled"`
	NATEnabled     bool      `json:"natEnabled"`
	Status         string    `json:"status"`
	Uptime         int64     `json:"uptime"`
	BytesSent      int64     `json:"bytesSent"`
	BytesReceived  int64     `json:"bytesReceived"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// LANConfig represents LAN configuration
type LANConfig struct {
	IPAddress   string `json:"ipAddress"`
	SubnetMask  string `json:"subnetMask"`
	DHCPEnabled bool   `json:"dhcpEnabled"`
	DHCPStart   string `json:"dhcpStart"`
	DHCPEnd     string `json:"dhcpEnd"`
	LeaseTime   int    `json:"leaseTime"`
}

// DeviceTask represents a pending task for a device
type DeviceTask struct {
	ID          int64           `json:"id"`
	DeviceID    int64           `json:"deviceId"`
	Type        TaskType        `json:"type"`
	Status      TaskStatus      `json:"status"`
	Parameters  json.RawMessage `json:"parameters"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
}

// TaskType represents the type of task
type TaskType string

const (
	TaskGetParameterValues TaskType = "getParameterValues"
	TaskSetParameterValues TaskType = "setParameterValues"
	TaskReboot             TaskType = "reboot"
	TaskFactoryReset       TaskType = "factoryReset"
	TaskDownload           TaskType = "download"
	TaskRefresh            TaskType = "refresh"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// Preset represents a provision/preset configuration
type Preset struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Filter      json.RawMessage `json:"filter"`     // Device filter criteria
	Provisions  json.RawMessage `json:"provisions"` // Actions to perform
	Weight      int             `json:"weight"`     // Priority (lower = higher priority)
	Enabled     bool            `json:"enabled"`
	Events      []string        `json:"events"` // Trigger events
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// Log represents a system log entry
type Log struct {
	ID        int64     `json:"id"`
	DeviceID  *int64    `json:"deviceId,omitempty"`
	Level     string    `json:"level"` // info, warning, error
	Category  string    `json:"category"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalDevices   int64            `json:"totalDevices"`
	OnlineDevices  int64            `json:"onlineDevices"`
	OfflineDevices int64            `json:"offlineDevices"`
	PendingTasks   int64            `json:"pendingTasks"`
	ActiveSessions int64            `json:"activeSessions"`
	DevicesByModel map[string]int64 `json:"devicesByModel"`
	RecentActivity []ActivityItem   `json:"recentActivity"`
}

// ActivityItem represents a recent activity
type ActivityItem struct {
	Type      string    `json:"type"`
	DeviceID  int64     `json:"deviceId"`
	DeviceSN  string    `json:"deviceSerialNumber"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// InformRequest represents TR-069 Inform request
type InformRequest struct {
	DeviceID      DeviceIDStruct         `json:"deviceId"`
	Event         []EventStruct          `json:"event"`
	ParameterList []ParameterValueStruct `json:"parameterList"`
	MaxEnvelopes  int                    `json:"maxEnvelopes"`
	CurrentTime   time.Time              `json:"currentTime"`
	RetryCount    int                    `json:"retryCount"`
}

// DeviceIDStruct represents device identification in TR-069
type DeviceIDStruct struct {
	Manufacturer string `json:"manufacturer"`
	OUI          string `json:"oui"`
	ProductClass string `json:"productClass"`
	SerialNumber string `json:"serialNumber"`
}

// EventStruct represents an event in TR-069 Inform
type EventStruct struct {
	EventCode  string `json:"eventCode"`
	CommandKey string `json:"commandKey"`
}

// ParameterValueStruct represents a parameter value
type ParameterValueStruct struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// User represents a system user
type User struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Password  string     `json:"-"` // Never expose password
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"lastLogin"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

// ============== BILLING & CUSTOMER MODELS ==============

// Customer represents an ISP customer
type Customer struct {
	ID           int64  `json:"id"`
	CustomerCode string `json:"customerCode"` // e.g., CUST-0001
	Name         string `json:"name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	// Location for map
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// Package subscription
	PackageID int64    `json:"packageId"`
	Package   *Package `json:"package,omitempty"`
	// Portal login credentials
	Username string `json:"username"`
	Password string `json:"-"` // Never expose
	// For input purposes (when creating/updating)
	InputPassword string `json:"password"`
	// Status
	Status   string    `json:"status"` // active, suspended, terminated
	FCMToken string    `json:"fcmToken"`
	JoinDate time.Time `json:"joinDate"`
	// Balance
	Balance float64 `json:"balance"` // Prepaid balance or outstanding
	// Devices assigned
	Devices   []*Device `json:"devices,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Package represents an internet package/plan
type Package struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"` // e.g., "Home 20 Mbps"
	Description   string    `json:"description"`
	DownloadSpeed int       `json:"downloadSpeed"` // in Mbps
	UploadSpeed   int       `json:"uploadSpeed"`   // in Mbps
	Quota         int64     `json:"quota"`         // in bytes, 0 = unlimited
	Price         float64   `json:"price"`         // Monthly price
	SetupFee      float64   `json:"setupFee"`      // One-time fee
	IsActive      bool      `json:"isActive"`
	Subscribers   int       `json:"subscribers"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type DeviceLog struct {
	ID        int64     `json:"id"`
	DeviceID  int64     `json:"deviceId"`
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changedAt"`
}

// Invoice represents a monthly bill
type Invoice struct {
	ID         int64     `json:"id"`
	InvoiceNo  string    `json:"invoiceNo"` // e.g., INV-202601-0001
	CustomerID int64     `json:"customerId"`
	Customer   *Customer `json:"customer,omitempty"`
	// Billing period
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	DueDate     time.Time `json:"dueDate"`
	// Amounts
	Subtotal float64 `json:"subtotal"`
	Tax      float64 `json:"tax"`
	Discount float64 `json:"discount"`
	Total    float64 `json:"total"`
	// Status
	Status     InvoiceStatus `json:"status"`
	PaidAmount float64       `json:"paidAmount"`
	PaidAt     *time.Time    `json:"paidAt,omitempty"`
	// Items
	Items     []InvoiceItem `json:"items"`
	Notes     string        `json:"notes"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// InvoiceStatus represents invoice payment status
type InvoiceStatus string

const (
	InvoicePending   InvoiceStatus = "pending"
	InvoicePaid      InvoiceStatus = "paid"
	InvoicePartial   InvoiceStatus = "partial"
	InvoiceOverdue   InvoiceStatus = "overdue"
	InvoiceCancelled InvoiceStatus = "cancelled"
	InvoiceCombined  InvoiceStatus = "combined"
)

// InvoiceItem represents a line item in an invoice
type InvoiceItem struct {
	ID          int64   `json:"id"`
	InvoiceID   int64   `json:"invoiceId"`
	Description string  `json:"description"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Amount      float64 `json:"amount"`
}

// Payment represents a payment record
type Payment struct {
	ID            int64     `json:"id"`
	PaymentNo     string    `json:"paymentNo"` // e.g., PAY-202601-0001
	CustomerID    int64     `json:"customerId"`
	InvoiceID     *int64    `json:"invoiceId,omitempty"` // Can be for specific invoice or general
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"paymentMethod"` // cash, transfer, qris, etc.
	Reference     string    `json:"reference"`     // Bank ref, receipt no, etc.
	Status        string    `json:"status"`        // pending, completed, failed, refunded
	Notes         string    `json:"notes"`
	ReceivedBy    string    `json:"receivedBy"` // Staff name
	PaymentDate   time.Time `json:"paymentDate"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// SupportTicket represents a customer support ticket
type SupportTicket struct {
	ID          int64      `json:"id"`
	TicketNo    string     `json:"ticketNo"`
	CustomerID  int64      `json:"customerId"`
	Customer    *Customer  `json:"customer,omitempty"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Category    string     `json:"category"` // billing, technical, general
	Priority    string     `json:"priority"` // low, medium, high
	Status      string     `json:"status"`   // open, in_progress, resolved, closed
	AssignedTo  *int64     `json:"assignedTo,omitempty"`
	Resolution  string     `json:"resolution"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ClosedAt    *time.Time `json:"closedAt,omitempty"`
}

// BillingStats represents billing dashboard statistics
type BillingStats struct {
	TotalCustomers     int64   `json:"totalCustomers"`
	ActiveCustomers    int64   `json:"activeCustomers"`
	SuspendedCustomers int64   `json:"suspendedCustomers"`
	MonthlyRevenue     float64 `json:"monthlyRevenue"`
	PendingInvoices    int64   `json:"pendingInvoices"`
	OverdueAmount      float64 `json:"overdueAmount"`
	TodayPayments      float64 `json:"todayPayments"`
}

// BandwidthRecord represents a bandwidth usage snapshot
type BandwidthRecord struct {
	Timestamp     time.Time `json:"timestamp"`
	BytesSent     int64     `json:"bytesSent"`
	BytesReceived int64     `json:"bytesReceived"`
}

// NetworkStats represents aggregated network statistics
type NetworkStats struct {
	TotalDownload int64       `json:"totalDownload"`
	TotalUpload   int64       `json:"totalUpload"`
	TopUsers      []UsageStat `json:"topUsers"`
	TrafficChart  []UsageStat `json:"trafficChart"` // Hourly data
}

type UsageStat struct {
	Label         string `json:"label"` // Customer Name or Hour
	BytesReceived int64  `json:"bytesReceived"`
	BytesSent     int64  `json:"bytesSent"`
}

type CustomerLocation struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Status       string  `json:"status"`       // Customer status
	DeviceStatus string  `json:"deviceStatus"` // Online/Offline (from primary device)
	Address      string  `json:"address"`
}

// ConnectedClient represents a device connected to the ONU
type ConnectedClient struct {
	Name      string `json:"name"`
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Type      string `json:"type"` // phone, laptop, tv, other
	RSSI      int    `json:"rssi"`
	Active    bool   `json:"active"`
	Interface string `json:"interface"`
}

// PONStats represents optical signal statistics
type PONStats struct {
	RXPower     float64 `json:"rxPower"`
	TXPower     float64 `json:"txPower"`
	Temperature float64 `json:"temperature"`
	Voltage     float64 `json:"voltage"`
	BiasCurrent float64 `json:"biasCurrent"`
	ONU_ID      string  `json:"onuId"`
	Distance    string  `json:"distance"`
	PONMode     string  `json:"ponMode"`
}
