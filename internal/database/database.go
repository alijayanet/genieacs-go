package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-acs/internal/models"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the database connection
type DB struct {
	*sql.DB
}

// InitDB initializes the database connection and creates tables
func InitDB(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	wrapper := &DB{db}

	// Create tables
	if err := wrapper.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	// Auto-migrations
	wrapper.checkAndMigrateDevicesTable()
	wrapper.checkAndMigrateCustomersTable()

	// Ensure default admin user exists
	wrapper.EnsureDefaultAdmin("admin", "admin123")

	return wrapper, nil
}

func (db *DB) checkAndMigrateDevicesTable() {
	var count int

	// Column: customer_id
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='customer_id'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding customer_id")
		db.Exec("ALTER TABLE devices ADD COLUMN customer_id INTEGER REFERENCES customers(id) ON DELETE SET NULL")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_customer ON devices(customer_id)")
	}

	// Column: rx_power
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='rx_power'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding rx_power")
		db.Exec("ALTER TABLE devices ADD COLUMN rx_power REAL DEFAULT 0")
	}

	// Column: client_count
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='client_count'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding client_count")
		db.Exec("ALTER TABLE devices ADD COLUMN client_count INTEGER DEFAULT 0")
	}

	// Column: template
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='template'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding template")
		db.Exec("ALTER TABLE devices ADD COLUMN template TEXT")
	}

	// Column: latitude
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='latitude'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding latitude")
		db.Exec("ALTER TABLE devices ADD COLUMN latitude REAL DEFAULT 0")
	}

	// Column: longitude
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='longitude'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding longitude")
		db.Exec("ALTER TABLE devices ADD COLUMN longitude REAL DEFAULT 0")
	}

	// Column: address
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='address'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding address")
		db.Exec("ALTER TABLE devices ADD COLUMN address TEXT")
	}

	// Column: temperature
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('devices') WHERE name='temperature'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating: adding temperature")
		db.Exec("ALTER TABLE devices ADD COLUMN temperature REAL DEFAULT 0")
	}
}

func (db *DB) checkAndMigrateCustomersTable() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('customers') WHERE name='fcm_token'").Scan(&count)
	if count == 0 {
		fmt.Println("[DB] Migrating customers table: adding fcm_token column")
		if _, err := db.Exec("ALTER TABLE customers ADD COLUMN fcm_token TEXT"); err != nil {
			fmt.Printf("[DB] Error adding fcm_token column: %v\n", err)
		}
	}
}

func (db *DB) createTables() error {
	tables := []string{
		// Devices table
		`CREATE TABLE IF NOT EXISTS devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			serial_number TEXT UNIQUE NOT NULL,
			oui TEXT,
			product_class TEXT,
			manufacturer TEXT,
			model_name TEXT,
			hardware_version TEXT,
			software_version TEXT,
			connection_request TEXT,
			status TEXT DEFAULT 'offline',
			last_inform DATETIME,
			last_contact DATETIME,
			ip_address TEXT,
			mac_address TEXT,
			uptime INTEGER DEFAULT 0,
			rx_power REAL DEFAULT 0,
			client_count INTEGER DEFAULT 0,
			template TEXT,
			parameters TEXT,
			tags TEXT,
			notes TEXT,
			latitude REAL DEFAULT 0,
			longitude REAL DEFAULT 0,
			address TEXT,
			customer_id INTEGER,
			temperature REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Device parameters table
		`CREATE TABLE IF NOT EXISTS device_parameters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			path TEXT NOT NULL,
			value TEXT,
			type TEXT DEFAULT 'string',
			writable BOOLEAN DEFAULT 1,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			UNIQUE(device_id, path)
		)`,

		// WAN configurations table
		`CREATE TABLE IF NOT EXISTS wan_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			name TEXT,
			connection_type TEXT,
			vlan INTEGER DEFAULT 0,
			username TEXT,
			password TEXT,
			ip_address TEXT,
			subnet_mask TEXT,
			gateway TEXT,
			dns1 TEXT,
			dns2 TEXT,
			mtu INTEGER DEFAULT 1500,
			enabled BOOLEAN DEFAULT 1,
			nat_enabled BOOLEAN DEFAULT 1,
			status TEXT DEFAULT 'disconnected',
			uptime INTEGER DEFAULT 0,
			bytes_sent INTEGER DEFAULT 0,
			bytes_received INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,

		// Tasks table
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			parameters TEXT,
			result TEXT,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			completed_at DATETIME,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,

		// Presets table
		`CREATE TABLE IF NOT EXISTS presets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			filter TEXT,
			provisions TEXT,
			weight INTEGER DEFAULT 0,
			enabled BOOLEAN DEFAULT 1,
			events TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Logs table
		`CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER,
			level TEXT DEFAULT 'info',
			category TEXT,
			message TEXT,
			details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE SET NULL
		)`,

		// Bandwidth Usage table
		`CREATE TABLE IF NOT EXISTS bandwidth_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			bytes_sent BIGINT DEFAULT 0,
			bytes_received BIGINT DEFAULT 0,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bandwidth_device_time ON bandwidth_usage(device_id, timestamp)`,

		// Device Logs table (Uptime Tracking)
		`CREATE TABLE IF NOT EXISTS device_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			status TEXT NOT NULL,
			changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(device_id) REFERENCES devices(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_device_logs_device ON device_logs(device_id)`,

		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user',
			last_login DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// ============== BILLING SYSTEM TABLES ==============

		// Packages table
		`CREATE TABLE IF NOT EXISTS packages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			download_speed INTEGER DEFAULT 0,
			upload_speed INTEGER DEFAULT 0,
			quota INTEGER DEFAULT 0,
			price REAL DEFAULT 0,
			setup_fee REAL DEFAULT 0,
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Customers table
		`CREATE TABLE IF NOT EXISTS customers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			customer_code TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			address TEXT,
			latitude REAL DEFAULT 0,
			longitude REAL DEFAULT 0,
			package_id INTEGER,
			username TEXT UNIQUE,
			password TEXT,
			status TEXT DEFAULT 'active',
			join_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			balance REAL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE SET NULL
		)`,

		// Invoices table
		`CREATE TABLE IF NOT EXISTS invoices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invoice_no TEXT UNIQUE NOT NULL,
			customer_id INTEGER NOT NULL,
			period_start DATETIME,
			period_end DATETIME,
			due_date DATETIME,
			subtotal REAL DEFAULT 0,
			tax REAL DEFAULT 0,
			discount REAL DEFAULT 0,
			total REAL DEFAULT 0,
			status TEXT DEFAULT 'pending',
			paid_amount REAL DEFAULT 0,
			paid_at DATETIME,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
		)`,

		// Invoice items table
		`CREATE TABLE IF NOT EXISTS invoice_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			invoice_id INTEGER NOT NULL,
			description TEXT,
			quantity INTEGER DEFAULT 1,
			unit_price REAL DEFAULT 0,
			amount REAL DEFAULT 0,
			FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
		)`,

		// Payments table
		`CREATE TABLE IF NOT EXISTS payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			payment_no TEXT UNIQUE NOT NULL,
			customer_id INTEGER NOT NULL,
			invoice_id INTEGER,
			amount REAL DEFAULT 0,
			payment_method TEXT DEFAULT 'cash',
			reference TEXT,
			status TEXT DEFAULT 'completed',
			notes TEXT,
			received_by TEXT,
			payment_date DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE SET NULL
		)`,

		// Support tickets table
		`CREATE TABLE IF NOT EXISTS support_tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_no TEXT UNIQUE NOT NULL,
			customer_id INTEGER NOT NULL,
			subject TEXT,
			description TEXT,
			category TEXT DEFAULT 'general',
			priority TEXT DEFAULT 'medium',
			status TEXT DEFAULT 'open',
			assigned_to INTEGER,
			resolution TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			closed_at DATETIME,
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE,
			FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL
		)`,

		// Update devices table to include location and customer fields
		`CREATE TABLE IF NOT EXISTS device_customer_map (
			device_id INTEGER NOT NULL,
			customer_id INTEGER NOT NULL,
			PRIMARY KEY (device_id, customer_id),
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE CASCADE
		)`,

		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_devices_serial ON devices(serial_number)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_last_contact ON devices(last_contact)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_created_at ON devices(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_ip_address ON devices(ip_address)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_mac_address ON devices(mac_address)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_status_last_contact ON devices(status, last_contact)`,
		`CREATE INDEX IF NOT EXISTS idx_device_parameters_device ON device_parameters(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_device_parameters_path ON device_parameters(path)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_device ON tasks(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_device ON logs(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_created ON logs(created_at)`,

		// Billing indexes
		`CREATE INDEX IF NOT EXISTS idx_customers_code ON customers(customer_code)`,
		`CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_customer ON invoices(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_customer ON payments(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_date ON payments(payment_date)`,

		// Settings table for application config
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v\nSQL: %s", err, table)
		}
	}

	return nil
}

// ============== Device Operations ==============

// GetDevices retrieves all devices with optional filtering
func (db *DB) GetDevices(status string, search string, limit, offset int) ([]*models.Device, int64, error) {
	var conditions []string
	var args []interface{}

	if status != "" && status != "all" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	if search != "" {
		conditions = append(conditions, "(serial_number LIKE ? OR manufacturer LIKE ? OR model_name LIKE ?)")
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM devices " + whereClause
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get devices
	query := fmt.Sprintf(`
		SELECT id, serial_number, oui, product_class, manufacturer, model_name,
			   hardware_version, software_version, connection_request, status,
			   last_inform, last_contact, ip_address, mac_address, uptime,
			   rx_power, client_count, template,
			   parameters, tags, notes, created_at, updated_at, latitude, longitude, address, temperature, customer_id
		FROM devices %s
		ORDER BY last_contact DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, 0, err
		}
		devices = append(devices, device)
	}

	return devices, total, nil
}

// GetDevicesByCustomer retrieves all devices belonging to a customer
func (db *DB) GetDevicesByCustomer(customerID int64) ([]*models.Device, error) {
	query := `
		SELECT id, serial_number, oui, product_class, manufacturer, model_name,
			   hardware_version, software_version, connection_request, status,
			   last_inform, last_contact, ip_address, mac_address, uptime,
			   rx_power, client_count, template,
			   parameters, tags, notes, created_at, updated_at, latitude, longitude, address, temperature, customer_id
		FROM devices WHERE customer_id = ?
		ORDER BY last_contact DESC
	`
	rows, err := db.Query(query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
}

// GetDevice retrieves a device by ID
func (db *DB) GetDevice(id int64) (*models.Device, error) {
	query := `
		SELECT id, serial_number, oui, product_class, manufacturer, model_name,
			   hardware_version, software_version, connection_request, status,
			   last_inform, last_contact, ip_address, mac_address, uptime,
			   rx_power, client_count, template,
			   parameters, tags, notes, created_at, updated_at, latitude, longitude, address, temperature, customer_id
		FROM devices WHERE id = ?
	`
	row := db.QueryRow(query, id)
	return scanDeviceRow(row)
}

// GetDeviceBySerial retrieves a device by serial number
func (db *DB) GetDeviceBySerial(serialNumber string) (*models.Device, error) {
	query := `
		SELECT id, serial_number, oui, product_class, manufacturer, model_name,
			   hardware_version, software_version, connection_request, status,
			   last_inform, last_contact, ip_address, mac_address, uptime,
			   rx_power, client_count, template,
			   parameters, tags, notes, created_at, updated_at, latitude, longitude, address, temperature, customer_id
		FROM devices WHERE serial_number = ?
	`
	row := db.QueryRow(query, serialNumber)
	return scanDeviceRow(row)
}

// CreateDevice creates a new device
func (db *DB) CreateDevice(device *models.Device) (*models.Device, error) {
	paramsJSON, _ := json.Marshal(device.Parameters)
	tagsJSON, _ := json.Marshal(device.Tags)

	result, err := db.Exec(`
		INSERT INTO devices (serial_number, oui, product_class, manufacturer, model_name,
							 hardware_version, software_version, connection_request, status,
							 ip_address, mac_address, uptime, rx_power, client_count, template,
							 parameters, tags, notes, temperature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		device.SerialNumber, device.OUI, device.ProductClass, device.Manufacturer,
		device.ModelName, device.HardwareVersion, device.SoftwareVersion,
		device.ConnectionRequest, device.Status, device.IPAddress, device.MACAddress,
		device.Uptime, device.RXPower, device.ClientCount, device.Template,
		string(paramsJSON), string(tagsJSON), device.Notes, device.Temperature,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return db.GetDevice(id)
}

// UpdateDevice updates an existing device
func (db *DB) UpdateDevice(device *models.Device) error {
	paramsJSON, _ := json.Marshal(device.Parameters)
	tagsJSON, _ := json.Marshal(device.Tags)

	_, err := db.Exec(`
		UPDATE devices SET
			oui = ?, product_class = ?, manufacturer = ?, model_name = ?,
			hardware_version = ?, software_version = ?, connection_request = ?,
			status = ?, last_inform = ?, last_contact = ?, ip_address = ?,
			mac_address = ?, uptime = ?, rx_power = ?, client_count = ?, template = ?,
			parameters = ?, tags = ?, notes = ?, temperature = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		device.OUI, device.ProductClass, device.Manufacturer, device.ModelName,
		device.HardwareVersion, device.SoftwareVersion, device.ConnectionRequest,
		device.Status, device.LastInform, device.LastContact, device.IPAddress,
		device.MACAddress, device.Uptime, device.RXPower, device.ClientCount, device.Template,
		string(paramsJSON), string(tagsJSON), device.Notes, device.Temperature, device.ID,
	)
	return err
}

// DeleteDevice deletes a device
func (db *DB) DeleteDevice(id int64) error {
	_, err := db.Exec("DELETE FROM devices WHERE id = ?", id)
	return err
}

// UpdateDeviceStatus updates the status and last contact time
func (db *DB) UpdateDeviceStatus(id int64, newStatus models.DeviceStatus) error {
	// 1. Get current status
	var oldStatus string
	err := db.QueryRow("SELECT COALESCE(status, 'offline') FROM devices WHERE id = ?", id).Scan(&oldStatus)
	if err != nil {
		return err
	}

	// 2. If changed, insert log
	if oldStatus != string(newStatus) {
		_, err = db.Exec("INSERT INTO device_logs (device_id, status, changed_at) VALUES (?, ?, CURRENT_TIMESTAMP)", id, newStatus)
		if err != nil {
			fmt.Printf("Failed to log status change for device %d: %v\n", id, err)
		}
	}

	// 3. Update device
	_, err = db.Exec(`
		UPDATE devices SET status = ?, last_contact = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newStatus, id)
	return err
}

// GetDeviceLogs retrieves uptime logs for a device
func (db *DB) GetDeviceLogs(deviceID int64, limit int) ([]models.DeviceLog, error) {
	rows, err := db.Query("SELECT id, device_id, status, changed_at FROM device_logs WHERE device_id = ? ORDER BY changed_at DESC LIMIT ?", deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.DeviceLog
	for rows.Next() {
		var l models.DeviceLog
		if err := rows.Scan(&l.ID, &l.DeviceID, &l.Status, &l.ChangedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ============== Device Parameters Operations ==============

// GetDeviceParameters retrieves all parameters for a device
func (db *DB) GetDeviceParameters(deviceID int64, pathPrefix string) ([]*models.DeviceParameter, error) {
	var rows *sql.Rows
	var err error

	if pathPrefix != "" {
		rows, err = db.Query(`
			SELECT id, device_id, path, value, type, writable, updated_at
			FROM device_parameters
			WHERE device_id = ? AND path LIKE ?
			ORDER BY path
		`, deviceID, pathPrefix+"%")
	} else {
		rows, err = db.Query(`
			SELECT id, device_id, path, value, type, writable, updated_at
			FROM device_parameters
			WHERE device_id = ?
			ORDER BY path
		`, deviceID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []*models.DeviceParameter
	for rows.Next() {
		var p models.DeviceParameter
		err := rows.Scan(&p.ID, &p.DeviceID, &p.Path, &p.Value, &p.Type, &p.Writable, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		params = append(params, &p)
	}

	return params, nil
}

// SetDeviceParameter sets or updates a device parameter
func (db *DB) SetDeviceParameter(deviceID int64, path, value, paramType string, writable bool) error {
	_, err := db.Exec(`
		INSERT INTO device_parameters (device_id, path, value, type, writable, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(device_id, path) DO UPDATE SET
			value = excluded.value,
			type = excluded.type,
			writable = excluded.writable,
			updated_at = CURRENT_TIMESTAMP
	`, deviceID, path, value, paramType, writable)
	return err
}

// ============== WAN Config Operations ==============

// GetWANConfigs retrieves all WAN configurations for a device
func (db *DB) GetWANConfigs(deviceID int64) ([]*models.WANConfig, error) {
	rows, err := db.Query(`
		SELECT id, device_id, name, connection_type, vlan, username, password,
			   ip_address, subnet_mask, gateway, dns1, dns2, mtu, enabled,
			   nat_enabled, status, uptime, bytes_sent, bytes_received,
			   created_at, updated_at
		FROM wan_configs
		WHERE device_id = ?
		ORDER BY id
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.WANConfig
	for rows.Next() {
		var c models.WANConfig
		err := rows.Scan(
			&c.ID, &c.DeviceID, &c.Name, &c.ConnectionType, &c.VLAN,
			&c.Username, &c.Password, &c.IPAddress, &c.SubnetMask, &c.Gateway,
			&c.DNS1, &c.DNS2, &c.MTU, &c.Enabled, &c.NATEnabled, &c.Status,
			&c.Uptime, &c.BytesSent, &c.BytesReceived, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &c)
	}

	return configs, nil
}

// CreateWANConfig creates a new WAN configuration
func (db *DB) CreateWANConfig(config *models.WANConfig) (*models.WANConfig, error) {
	result, err := db.Exec(`
		INSERT INTO wan_configs (device_id, name, connection_type, vlan, username, password,
								 ip_address, subnet_mask, gateway, dns1, dns2, mtu, enabled, nat_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		config.DeviceID, config.Name, config.ConnectionType, config.VLAN,
		config.Username, config.Password, config.IPAddress, config.SubnetMask,
		config.Gateway, config.DNS1, config.DNS2, config.MTU, config.Enabled, config.NATEnabled,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	config.ID = id
	return config, nil
}

// UpdateWANConfig updates a WAN configuration
func (db *DB) UpdateWANConfig(config *models.WANConfig) error {
	_, err := db.Exec(`
		UPDATE wan_configs SET
			name = ?, connection_type = ?, vlan = ?, username = ?, password = ?,
			ip_address = ?, subnet_mask = ?, gateway = ?, dns1 = ?, dns2 = ?,
			mtu = ?, enabled = ?, nat_enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		config.Name, config.ConnectionType, config.VLAN, config.Username, config.Password,
		config.IPAddress, config.SubnetMask, config.Gateway, config.DNS1, config.DNS2,
		config.MTU, config.Enabled, config.NATEnabled, config.ID,
	)
	return err
}

// DeleteWANConfig deletes a WAN configuration
func (db *DB) DeleteWANConfig(id int64) error {
	_, err := db.Exec("DELETE FROM wan_configs WHERE id = ?", id)
	return err
}

// ============== Task Operations ==============

// GetPendingTasks retrieves pending tasks for a device
func (db *DB) GetPendingTasks(deviceID int64) ([]*models.DeviceTask, error) {
	rows, err := db.Query(`
		SELECT id, device_id, type, status, parameters, result, error,
			   created_at, started_at, completed_at
		FROM tasks
		WHERE device_id = ? AND status = 'pending'
		ORDER BY created_at ASC
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*models.DeviceTask
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// CreateTask creates a new task
func (db *DB) CreateTask(task *models.DeviceTask) (*models.DeviceTask, error) {
	result, err := db.Exec(`
		INSERT INTO tasks (device_id, type, status, parameters)
		VALUES (?, ?, ?, ?)
	`, task.DeviceID, task.Type, models.TaskPending, string(task.Parameters))
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	task.ID = id
	task.Status = models.TaskPending
	return task, nil
}

// UpdateTask updates a task in the database
func (db *DB) UpdateTask(task *models.DeviceTask) error {
	paramsJSON, _ := json.Marshal(task.Parameters)
	resultJSON, _ := json.Marshal(task.Result)

	_, err := db.Exec(`
		UPDATE tasks SET
			status = ?,
			parameters = ?,
			result = ?,
			error = ?,
			started_at = ?,
			completed_at = ?
		WHERE id = ?
	`, task.Status, string(paramsJSON), string(resultJSON), task.Error, task.StartedAt, task.CompletedAt, task.ID)
	return err
}

// UpdateTaskStatus updates a task's status
func (db *DB) UpdateTaskStatus(id int64, status models.TaskStatus, result json.RawMessage, errMsg string) error {
	_, err := db.Exec(`
		UPDATE tasks SET
			status = ?,
			result = ?,
			error = ?,
			started_at = CASE WHEN ? = 'running' AND started_at IS NULL THEN CURRENT_TIMESTAMP ELSE started_at END,
			completed_at = CASE WHEN ? IN ('completed', 'failed') THEN CURRENT_TIMESTAMP ELSE completed_at END
		WHERE id = ?
	`, status, string(result), errMsg, status, status, id)
	return err
}

// ============== Dashboard Operations ==============

// GetDashboardStats retrieves dashboard statistics
func (db *DB) GetDashboardStats() (*models.DashboardStats, error) {
	stats := &models.DashboardStats{
		DevicesByModel: make(map[string]int64),
	}

	// Total devices
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&stats.TotalDevices)

	// Online devices
	db.QueryRow("SELECT COUNT(*) FROM devices WHERE status = 'online'").Scan(&stats.OnlineDevices)

	// Offline devices
	stats.OfflineDevices = stats.TotalDevices - stats.OnlineDevices

	// Pending tasks
	db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'pending'").Scan(&stats.PendingTasks)

	// Devices by model
	rows, err := db.Query(`
		SELECT COALESCE(model_name, 'Unknown'), COUNT(*)
		FROM devices
		GROUP BY model_name
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var count int64
			if rows.Scan(&model, &count) == nil {
				stats.DevicesByModel[model] = count
			}
		}
	}

	// Recent activity
	activityRows, err := db.Query(`
		SELECT l.category, l.message, l.created_at, d.id, d.serial_number
		FROM logs l
		LEFT JOIN devices d ON l.device_id = d.id
		ORDER BY l.created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer activityRows.Close()
		for activityRows.Next() {
			var activity models.ActivityItem
			var deviceID sql.NullInt64
			var deviceSN sql.NullString
			if activityRows.Scan(&activity.Type, &activity.Message, &activity.Timestamp, &deviceID, &deviceSN) == nil {
				if deviceID.Valid {
					activity.DeviceID = deviceID.Int64
				}
				if deviceSN.Valid {
					activity.DeviceSN = deviceSN.String
				}
				stats.RecentActivity = append(stats.RecentActivity, activity)
			}
		}
	}

	return stats, nil
}

// ============== Log Operations ==============

// CreateLog creates a new log entry
func (db *DB) CreateLog(deviceID *int64, level, category, message, details string) error {
	_, err := db.Exec(`
		INSERT INTO logs (device_id, level, category, message, details)
		VALUES (?, ?, ?, ?, ?)
	`, deviceID, level, category, message, details)
	return err
}

// GetLogs retrieves logs with filtering
func (db *DB) GetLogs(deviceID *int64, level string, limit, offset int) ([]*models.Log, error) {
	var conditions []string
	var args []interface{}

	if deviceID != nil {
		conditions = append(conditions, "device_id = ?")
		args = append(args, *deviceID)
	}

	if level != "" && level != "all" {
		conditions = append(conditions, "level = ?")
		args = append(args, level)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, device_id, level, category, message, details, created_at
		FROM logs %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.Log
	for rows.Next() {
		var l models.Log
		var deviceID sql.NullInt64
		err := rows.Scan(&l.ID, &deviceID, &l.Level, &l.Category, &l.Message, &l.Details, &l.CreatedAt)
		if err != nil {
			return nil, err
		}
		if deviceID.Valid {
			l.DeviceID = &deviceID.Int64
		}
		logs = append(logs, &l)
	}

	return logs, nil
}

// ============== Helper Functions ==============

func scanDevice(rows *sql.Rows) (*models.Device, error) {
	var d models.Device
	var lastInform, lastContact sql.NullTime
	var paramsStr, tagsStr, notes, address, templateStr sql.NullString
	var lat, long, temp sql.NullFloat64
	var rxPower sql.NullFloat64
	var clientCount sql.NullInt64
	var customerID sql.NullInt64

	err := rows.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer,
		&d.ModelName, &d.HardwareVersion, &d.SoftwareVersion, &d.ConnectionRequest,
		&d.Status, &lastInform, &lastContact, &d.IPAddress, &d.MACAddress,
		&d.Uptime, &rxPower, &clientCount, &templateStr,
		&paramsStr, &tagsStr, &notes, &d.CreatedAt, &d.UpdatedAt,
		&lat, &long, &address, &temp, &customerID,
	)
	if err != nil {
		return nil, err
	}

	d.RXPower = rxPower.Float64
	d.ClientCount = int(clientCount.Int64)
	d.Template = templateStr.String
	d.Latitude = lat.Float64
	d.Longitude = long.Float64
	d.Temperature = temp.Float64
	d.Address = address.String
	if customerID.Valid {
		d.CustomerID = &customerID.Int64
	}

	if lastInform.Valid {
		d.LastInform = &lastInform.Time
	}
	if lastContact.Valid {
		d.LastContact = &lastContact.Time
	}
	if notes.Valid {
		d.Notes = notes.String
	}

	// Parse parameters JSON
	d.Parameters = make(map[string]string)
	if paramsStr.Valid && paramsStr.String != "" {
		json.Unmarshal([]byte(paramsStr.String), &d.Parameters)
	}

	// Parse tags JSON
	if tagsStr.Valid && tagsStr.String != "" {
		json.Unmarshal([]byte(tagsStr.String), &d.Tags)
	}

	return &d, nil
}

func scanDeviceRow(row *sql.Row) (*models.Device, error) {
	var d models.Device
	var lastInform, lastContact sql.NullTime
	var paramsStr, tagsStr, notes, address, templateStr sql.NullString
	var lat, long, temp sql.NullFloat64
	var rxPower sql.NullFloat64
	var clientCount sql.NullInt64
	var customerID sql.NullInt64

	err := row.Scan(
		&d.ID, &d.SerialNumber, &d.OUI, &d.ProductClass, &d.Manufacturer,
		&d.ModelName, &d.HardwareVersion, &d.SoftwareVersion, &d.ConnectionRequest,
		&d.Status, &lastInform, &lastContact, &d.IPAddress, &d.MACAddress,
		&d.Uptime, &rxPower, &clientCount, &templateStr,
		&paramsStr, &tagsStr, &notes, &d.CreatedAt, &d.UpdatedAt,
		&lat, &long, &address, &temp, &customerID,
	)
	if err != nil {
		return nil, err
	}

	d.RXPower = rxPower.Float64
	d.ClientCount = int(clientCount.Int64)
	d.Template = templateStr.String
	d.Latitude = lat.Float64
	d.Longitude = long.Float64
	d.Temperature = temp.Float64
	d.Address = address.String
	if customerID.Valid {
		d.CustomerID = &customerID.Int64
	}

	if lastInform.Valid {
		d.LastInform = &lastInform.Time
	}
	if lastContact.Valid {
		d.LastContact = &lastContact.Time
	}
	if notes.Valid {
		d.Notes = notes.String
	}

	d.Parameters = make(map[string]string)
	if paramsStr.Valid && paramsStr.String != "" {
		json.Unmarshal([]byte(paramsStr.String), &d.Parameters)
	}

	if tagsStr.Valid && tagsStr.String != "" {
		json.Unmarshal([]byte(tagsStr.String), &d.Tags)
	}

	return &d, nil
}

func scanTask(rows *sql.Rows) (*models.DeviceTask, error) {
	var t models.DeviceTask
	var params, result sql.NullString
	var errMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := rows.Scan(
		&t.ID, &t.DeviceID, &t.Type, &t.Status, &params, &result,
		&errMsg, &t.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}

	if params.Valid {
		t.Parameters = json.RawMessage(params.String)
	}
	if result.Valid {
		t.Result = json.RawMessage(result.String)
	}
	if errMsg.Valid {
		t.Error = errMsg.String
	}
	if startedAt.Valid {
		t.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}

	return &t, nil
}

// ============== Package Operations ==============

// GetPackages retrieves all packages
func (db *DB) GetPackages(activeOnly bool) ([]*models.Package, error) {
	query := `
		SELECT p.id, p.name, p.description, p.download_speed, p.upload_speed, p.quota, p.price, p.setup_fee, p.is_active, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM customers WHERE package_id = p.id) as subscribers
		FROM packages p
	`
	if activeOnly {
		query += " WHERE p.is_active = 1"
	}
	query += " ORDER BY p.price ASC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*models.Package
	for rows.Next() {
		var p models.Package
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &desc, &p.DownloadSpeed, &p.UploadSpeed, &p.Quota, &p.Price, &p.SetupFee, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.Subscribers)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			p.Description = desc.String
		}
		packages = append(packages, &p)
	}
	return packages, nil
}

// GetPackage retrieves a package by ID
func (db *DB) GetPackage(id int64) (*models.Package, error) {
	var p models.Package
	var desc sql.NullString
	err := db.QueryRow(`
		SELECT id, name, description, download_speed, upload_speed, quota, price, setup_fee, is_active, created_at, updated_at,
		       (SELECT COUNT(*) FROM customers WHERE package_id = id) as subscribers
		FROM packages WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &desc, &p.DownloadSpeed, &p.UploadSpeed, &p.Quota, &p.Price, &p.SetupFee, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.Subscribers)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		p.Description = desc.String
	}
	return &p, nil
}

// CreatePackage creates a new package
func (db *DB) CreatePackage(pkg *models.Package) (*models.Package, error) {
	result, err := db.Exec(`
		INSERT INTO packages (name, description, download_speed, upload_speed, quota, price, setup_fee, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, pkg.Name, pkg.Description, pkg.DownloadSpeed, pkg.UploadSpeed, pkg.Quota, pkg.Price, pkg.SetupFee, pkg.IsActive)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return db.GetPackage(id)
}

// UpdatePackage updates a package
func (db *DB) UpdatePackage(pkg *models.Package) error {
	_, err := db.Exec(`
		UPDATE packages SET name = ?, description = ?, download_speed = ?, upload_speed = ?, quota = ?, 
		price = ?, setup_fee = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, pkg.Name, pkg.Description, pkg.DownloadSpeed, pkg.UploadSpeed, pkg.Quota, pkg.Price, pkg.SetupFee, pkg.IsActive, pkg.ID)
	return err
}

// DeletePackage deletes a package
func (db *DB) DeletePackage(id int64) error {
	_, err := db.Exec("DELETE FROM packages WHERE id = ?", id)
	return err
}

// ============== Customer Operations ==============

// GetCustomers retrieves all customers with optional filtering
func (db *DB) GetCustomers(status string, search string, limit, offset int) ([]*models.Customer, int64, error) {
	var conditions []string
	var args []interface{}

	if status != "" && status != "all" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	if search != "" {
		conditions = append(conditions, "(customer_code LIKE ? OR name LIKE ? OR phone LIKE ?)")
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM customers " + whereClause
	db.QueryRow(countQuery, args...).Scan(&total)

	// Get customers
	query := fmt.Sprintf(`
		SELECT c.id, c.customer_code, c.name, c.email, c.phone, c.address, c.latitude, c.longitude,
		       c.package_id, c.username, c.status, c.join_date, c.balance, c.created_at, c.updated_at, c.fcm_token,
		       p.name, p.price, p.download_speed, p.upload_speed
		FROM customers c 
		LEFT JOIN packages p ON c.package_id = p.id
		%s
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		var email, phone, address, username, fcmToken sql.NullString
		var packageID sql.NullInt64
		var pkgName sql.NullString
		var pkgPrice sql.NullFloat64
		var pkgDown, pkgUp sql.NullInt64

		err := rows.Scan(&c.ID, &c.CustomerCode, &c.Name, &email, &phone, &address, &c.Latitude, &c.Longitude,
			&packageID, &username, &c.Status, &c.JoinDate, &c.Balance, &c.CreatedAt, &c.UpdatedAt, &fcmToken,
			&pkgName, &pkgPrice, &pkgDown, &pkgUp)
		if err != nil {
			return nil, 0, err
		}
		if email.Valid {
			c.Email = email.String
		}
		if phone.Valid {
			c.Phone = phone.String
		}
		if address.Valid {
			c.Address = address.String
		}
		if packageID.Valid {
			c.PackageID = packageID.Int64
		}
		if username.Valid {
			c.Username = username.String
		}
		if fcmToken.Valid {
			c.FCMToken = fcmToken.String
		}

		if pkgName.Valid {
			c.Package = &models.Package{
				ID:            packageID.Int64,
				Name:          pkgName.String,
				Price:         pkgPrice.Float64,
				DownloadSpeed: int(pkgDown.Int64),
				UploadSpeed:   int(pkgUp.Int64),
			}
		}

		customers = append(customers, &c)
	}
	return customers, total, nil
}

// GetCustomerLocations retrieves customer locations for mapping
func (db *DB) GetCustomerLocations() ([]models.CustomerLocation, error) {
	query := `
        SELECT c.id, c.name, COALESCE(c.latitude, 0), COALESCE(c.longitude, 0), c.status, c.address,
               COALESCE(d.status, 'offline') as device_status
        FROM customers c
        LEFT JOIN devices d ON d.customer_id = c.id
        GROUP BY c.id
    `
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locs []models.CustomerLocation
	for rows.Next() {
		var l models.CustomerLocation
		var addr sql.NullString
		if err := rows.Scan(&l.ID, &l.Name, &l.Latitude, &l.Longitude, &l.Status, &addr, &l.DeviceStatus); err != nil {
			continue
		}
		l.Address = addr.String
		locs = append(locs, l)
	}
	return locs, nil
}

// UpdateCustomerLocation updates the geolocation of a customer
func (db *DB) UpdateCustomerLocation(id int64, lat, long float64, address string) error {
	_, err := db.Exec("UPDATE customers SET latitude=?, longitude=?, address=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", lat, long, address, id)
	return err
}

// UpdateCustomerFCM updates the FCM token for a customer
func (db *DB) UpdateCustomerFCM(id int64, token string) error {
	_, err := db.Exec("UPDATE customers SET fcm_token=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", token, id)
	return err
}

// GetCustomer retrieves a customer by ID
func (db *DB) GetCustomer(id int64) (*models.Customer, error) {
	var c models.Customer
	var email, phone, address, username, fcmToken sql.NullString
	var packageID sql.NullInt64
	var pkgName sql.NullString
	var pkgPrice sql.NullFloat64
	var pkgDown, pkgUp sql.NullInt64

	err := db.QueryRow(`
		SELECT c.id, c.customer_code, c.name, c.email, c.phone, c.address, c.latitude, c.longitude,
		       c.package_id, c.username, c.status, c.join_date, c.balance, c.created_at, c.updated_at, c.fcm_token,
		       p.name, p.price, p.download_speed, p.upload_speed
		FROM customers c
		LEFT JOIN packages p ON c.package_id = p.id
		WHERE c.id = ?
	`, id).Scan(&c.ID, &c.CustomerCode, &c.Name, &email, &phone, &address, &c.Latitude, &c.Longitude,
		&packageID, &username, &c.Status, &c.JoinDate, &c.Balance, &c.CreatedAt, &c.UpdatedAt, &fcmToken,
		&pkgName, &pkgPrice, &pkgDown, &pkgUp)
	if err != nil {
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if address.Valid {
		c.Address = address.String
	}
	if packageID.Valid {
		c.PackageID = packageID.Int64
	}
	if username.Valid {
		c.Username = username.String
	}

	if fcmToken.Valid {
		c.FCMToken = fcmToken.String
	}

	if pkgName.Valid {
		c.Package = &models.Package{
			ID:            packageID.Int64,
			Name:          pkgName.String,
			Price:         pkgPrice.Float64,
			DownloadSpeed: int(pkgDown.Int64),
			UploadSpeed:   int(pkgUp.Int64),
		}
	}

	return &c, nil
}

// CreateCustomer creates a new customer
func (db *DB) CreateCustomer(customer *models.Customer) (*models.Customer, error) {
	// Generate customer code if not provided
	if customer.CustomerCode == "" {
		var count int64
		db.QueryRow("SELECT COUNT(*) FROM customers").Scan(&count)
		customer.CustomerCode = fmt.Sprintf("CUST-%04d", count+1)
	}

	result, err := db.Exec(`
		INSERT INTO customers (customer_code, name, email, phone, address, latitude, longitude, package_id, username, password, status, balance)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, customer.CustomerCode, customer.Name, customer.Email, customer.Phone, customer.Address,
		customer.Latitude, customer.Longitude, customer.PackageID, customer.Username, customer.Password, customer.Status, customer.Balance)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	return db.GetCustomer(id)
}

// UpdateCustomer updates a customer
func (db *DB) UpdateCustomer(customer *models.Customer) error {
	_, err := db.Exec(`
		UPDATE customers SET name = ?, email = ?, phone = ?, address = ?, latitude = ?, longitude = ?,
		package_id = ?, username = ?, password = ?, status = ?, balance = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, customer.Name, customer.Email, customer.Phone, customer.Address, customer.Latitude, customer.Longitude,
		customer.PackageID, customer.Username, customer.Password, customer.Status, customer.Balance, customer.ID)
	return err
}

// DeleteCustomer deletes a customer
func (db *DB) DeleteCustomer(id int64) error {
	_, err := db.Exec("DELETE FROM customers WHERE id = ?", id)
	return err
}

// ============== Invoice Operations ==============

// GetInvoices retrieves invoices with optional filtering
func (db *DB) GetInvoices(customerID *int64, status string, limit, offset int) ([]*models.Invoice, int64, error) {
	var conditions []string
	var args []interface{}

	if customerID != nil {
		conditions = append(conditions, "customer_id = ?")
		args = append(args, *customerID)
	}
	if status != "" && status != "all" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM invoices "+whereClause, args...).Scan(&total)

	query := fmt.Sprintf(`
		SELECT id, invoice_no, customer_id, period_start, period_end, due_date, 
		       subtotal, tax, discount, total, status, paid_amount, paid_at, notes, created_at, updated_at
		FROM invoices %s ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invoices []*models.Invoice
	for rows.Next() {
		var inv models.Invoice
		var periodStart, periodEnd, dueDate, paidAt sql.NullTime
		var notes sql.NullString
		err := rows.Scan(&inv.ID, &inv.InvoiceNo, &inv.CustomerID, &periodStart, &periodEnd, &dueDate,
			&inv.Subtotal, &inv.Tax, &inv.Discount, &inv.Total, &inv.Status, &inv.PaidAmount, &paidAt, &notes, &inv.CreatedAt, &inv.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		if periodStart.Valid {
			inv.PeriodStart = periodStart.Time
		}
		if periodEnd.Valid {
			inv.PeriodEnd = periodEnd.Time
		}
		if dueDate.Valid {
			inv.DueDate = dueDate.Time
		}
		if paidAt.Valid {
			inv.PaidAt = &paidAt.Time
		}
		if notes.Valid {
			inv.Notes = notes.String
		}
		invoices = append(invoices, &inv)
	}
	return invoices, total, nil
}

// CreateInvoice creates a new invoice
func (db *DB) CreateInvoice(inv *models.Invoice) (*models.Invoice, error) {
	// Generate invoice number
	if inv.InvoiceNo == "" {
		var count int64
		db.QueryRow("SELECT COUNT(*) FROM invoices WHERE strftime('%Y%m', created_at) = strftime('%Y%m', 'now')").Scan(&count)
		inv.InvoiceNo = fmt.Sprintf("INV-%s-%04d", time.Now().Format("200601"), count+1)
	}

	result, err := db.Exec(`
		INSERT INTO invoices (invoice_no, customer_id, period_start, period_end, due_date, subtotal, tax, discount, total, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.InvoiceNo, inv.CustomerID, inv.PeriodStart, inv.PeriodEnd, inv.DueDate, inv.Subtotal, inv.Tax, inv.Discount, inv.Total, inv.Status, inv.Notes)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	inv.ID = id
	return inv, nil
}

// GetInvoice retrieves a single invoice by ID
func (db *DB) GetInvoice(id int64) (*models.Invoice, error) {
	var inv models.Invoice
	var periodStart, periodEnd, dueDate, paidAt sql.NullTime
	var notes sql.NullString
	err := db.QueryRow(`
		SELECT id, invoice_no, customer_id, period_start, period_end, due_date, 
		       subtotal, tax, discount, total, status, paid_amount, paid_at, notes, created_at, updated_at
		FROM invoices WHERE id = ?
	`, id).Scan(&inv.ID, &inv.InvoiceNo, &inv.CustomerID, &periodStart, &periodEnd, &dueDate,
		&inv.Subtotal, &inv.Tax, &inv.Discount, &inv.Total, &inv.Status, &inv.PaidAmount, &paidAt, &notes, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if periodStart.Valid {
		inv.PeriodStart = periodStart.Time
	}
	if periodEnd.Valid {
		inv.PeriodEnd = periodEnd.Time
	}
	if dueDate.Valid {
		inv.DueDate = dueDate.Time
	}
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}
	if notes.Valid {
		inv.Notes = notes.String
	}
	return &inv, nil
}

// GetInvoiceByNumber retrieves a single invoice by invoice number
func (db *DB) GetInvoiceByNumber(invoiceNo string) (*models.Invoice, error) {
	var inv models.Invoice
	var periodStart, periodEnd, dueDate, paidAt sql.NullTime
	var notes sql.NullString
	err := db.QueryRow(`
		SELECT id, invoice_no, customer_id, period_start, period_end, due_date, 
		       subtotal, tax, discount, total, status, paid_amount, paid_at, notes, created_at, updated_at
		FROM invoices WHERE invoice_no = ?
	`, invoiceNo).Scan(&inv.ID, &inv.InvoiceNo, &inv.CustomerID, &periodStart, &periodEnd, &dueDate,
		&inv.Subtotal, &inv.Tax, &inv.Discount, &inv.Total, &inv.Status, &inv.PaidAmount, &paidAt, &notes, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if periodStart.Valid {
		inv.PeriodStart = periodStart.Time
	}
	if periodEnd.Valid {
		inv.PeriodEnd = periodEnd.Time
	}
	if dueDate.Valid {
		inv.DueDate = dueDate.Time
	}
	if paidAt.Valid {
		inv.PaidAt = &paidAt.Time
	}
	if notes.Valid {
		inv.Notes = notes.String
	}
	return &inv, nil
}

// UpdateInvoice updates an invoice
func (db *DB) UpdateInvoice(inv *models.Invoice) error {
	_, err := db.Exec(`
		UPDATE invoices SET status = ?, paid_amount = ?, paid_at = ?, notes = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, inv.Status, inv.PaidAmount, inv.PaidAt, inv.Notes, inv.ID)
	return err
}

// UpdateInvoiceStatus updates invoice status and paid amount
func (db *DB) UpdateInvoiceStatus(id int64, status models.InvoiceStatus, paidAmount float64) error {
	var paidAt interface{}
	if status == models.InvoicePaid {
		paidAt = time.Now()
	}
	_, err := db.Exec(`
		UPDATE invoices SET status = ?, paid_amount = ?, paid_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, status, paidAmount, paidAt, id)
	return err
}

// ============== Payment Operations ==============

// GetPayments retrieves payments
func (db *DB) GetPayments(customerID *int64, limit, offset int) ([]*models.Payment, int64, error) {
	whereClause := ""
	var args []interface{}
	if customerID != nil {
		whereClause = "WHERE customer_id = ?"
		args = append(args, *customerID)
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM payments "+whereClause, args...).Scan(&total)

	query := fmt.Sprintf(`
		SELECT id, payment_no, customer_id, invoice_id, amount, payment_method, reference, status, notes, received_by, payment_date, created_at, updated_at
		FROM payments %s ORDER BY payment_date DESC LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var payments []*models.Payment
	for rows.Next() {
		var p models.Payment
		var invoiceID sql.NullInt64
		var reference, notes, receivedBy sql.NullString
		err := rows.Scan(&p.ID, &p.PaymentNo, &p.CustomerID, &invoiceID, &p.Amount, &p.PaymentMethod, &reference, &p.Status, &notes, &receivedBy, &p.PaymentDate, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		if invoiceID.Valid {
			p.InvoiceID = &invoiceID.Int64
		}
		if reference.Valid {
			p.Reference = reference.String
		}
		if notes.Valid {
			p.Notes = notes.String
		}
		if receivedBy.Valid {
			p.ReceivedBy = receivedBy.String
		}
		payments = append(payments, &p)
	}
	return payments, total, nil
}

// CreatePayment creates a new payment
func (db *DB) CreatePayment(payment *models.Payment) (*models.Payment, error) {
	// Generate payment number
	if payment.PaymentNo == "" {
		var count int64
		db.QueryRow("SELECT COUNT(*) FROM payments WHERE strftime('%Y%m', created_at) = strftime('%Y%m', 'now')").Scan(&count)
		payment.PaymentNo = fmt.Sprintf("PAY-%s-%04d", time.Now().Format("200601"), count+1)
	}

	result, err := db.Exec(`
		INSERT INTO payments (payment_no, customer_id, invoice_id, amount, payment_method, reference, status, notes, received_by, payment_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, payment.PaymentNo, payment.CustomerID, payment.InvoiceID, payment.Amount, payment.PaymentMethod, payment.Reference, payment.Status, payment.Notes, payment.ReceivedBy, payment.PaymentDate)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	payment.ID = id
	return payment, nil
}

// ============== Billing Stats ==============

// GetBillingStats retrieves billing dashboard statistics
func (db *DB) GetBillingStats() (*models.BillingStats, error) {
	stats := &models.BillingStats{}

	// Total customers
	db.QueryRow("SELECT COUNT(*) FROM customers").Scan(&stats.TotalCustomers)

	// Active customers
	db.QueryRow("SELECT COUNT(*) FROM customers WHERE status = 'active'").Scan(&stats.ActiveCustomers)

	// Suspended customers
	db.QueryRow("SELECT COUNT(*) FROM customers WHERE status = 'suspended'").Scan(&stats.SuspendedCustomers)

	// Monthly revenue (this month's paid invoices)
	db.QueryRow(`
		SELECT COALESCE(SUM(paid_amount), 0) FROM invoices 
		WHERE status = 'paid' AND strftime('%Y%m', paid_at) = strftime('%Y%m', 'now')
	`).Scan(&stats.MonthlyRevenue)

	// Pending invoices
	db.QueryRow("SELECT COUNT(*) FROM invoices WHERE status = 'pending'").Scan(&stats.PendingInvoices)

	// Overdue amount
	db.QueryRow(`
		SELECT COALESCE(SUM(total - paid_amount), 0) FROM invoices 
		WHERE status IN ('pending', 'overdue') AND due_date < date('now')
	`).Scan(&stats.OverdueAmount)

	// Today's payments
	db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) FROM payments 
		WHERE date(payment_date) = date('now') AND status = 'completed'
	`).Scan(&stats.TodayPayments)

	return stats, nil
}

// ============== Customer Portal Operations ==============

// GetCustomerByUsername retrieves a customer by username
func (db *DB) GetCustomerByUsername(username string) (*models.Customer, error) {
	var c models.Customer
	var email, phone, address, pwd sql.NullString
	var packageID sql.NullInt64
	err := db.QueryRow(`
		SELECT id, customer_code, name, email, phone, address, latitude, longitude,
		       package_id, username, password, status, join_date, balance, created_at, updated_at
		FROM customers WHERE username = ?
	`, username).Scan(&c.ID, &c.CustomerCode, &c.Name, &email, &phone, &address, &c.Latitude, &c.Longitude,
		&packageID, &c.Username, &pwd, &c.Status, &c.JoinDate, &c.Balance, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if address.Valid {
		c.Address = address.String
	}
	if packageID.Valid {
		c.PackageID = packageID.Int64
	}
	if pwd.Valid {
		c.Password = pwd.String
	}
	return &c, nil
}

// GetCustomerByCode retrieves a customer by customer code
func (db *DB) GetCustomerByCode(code string) (*models.Customer, error) {
	var c models.Customer
	var email, phone, address, username, pwd sql.NullString
	var packageID sql.NullInt64
	err := db.QueryRow(`
		SELECT id, customer_code, name, email, phone, address, latitude, longitude,
		       package_id, username, password, status, join_date, balance, created_at, updated_at
		FROM customers WHERE customer_code = ?
	`, code).Scan(&c.ID, &c.CustomerCode, &c.Name, &email, &phone, &address, &c.Latitude, &c.Longitude,
		&packageID, &username, &pwd, &c.Status, &c.JoinDate, &c.Balance, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if address.Valid {
		c.Address = address.String
	}
	if packageID.Valid {
		c.PackageID = packageID.Int64
	}
	if username.Valid {
		c.Username = username.String
	}
	if pwd.Valid {
		c.Password = pwd.String
	}
	return &c, nil
}

// GetDeviceByTemplate retrieves a device by its template field which contains the PPPoE username
func (db *DB) GetDeviceByTemplate(template string) (*models.Device, error) {
	query := `
		SELECT id, serial_number, oui, product_class, manufacturer, model_name,
		       hardware_version, software_version, connection_request, status,
		       last_inform, last_contact, ip_address, mac_address, uptime,
		       rx_power, client_count, template,
		       parameters, tags, notes, created_at, updated_at, latitude, longitude, address, temperature, customer_id
		FROM devices WHERE template = ?
	`
	row := db.QueryRow(query, template)
	return scanDeviceRow(row)
}

// GetCustomerDevices retrieves all devices assigned to a customer
func (db *DB) GetCustomerDevices(customerID int64) ([]*models.Device, error) {
	rows, err := db.Query(`
		SELECT d.id, d.serial_number, d.oui, d.product_class, d.manufacturer, d.model_name,
		       d.hardware_version, d.software_version, d.connection_request, d.status,
		       d.last_inform, d.last_contact, d.ip_address, d.mac_address, d.uptime,
		       d.parameters, d.tags, d.notes, d.created_at, d.updated_at
		FROM devices d
		INNER JOIN device_customer_map dcm ON d.id = dcm.device_id
		WHERE dcm.customer_id = ?
		ORDER BY d.last_contact DESC
	`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
}

// GetCustomerByPPPoE retrieves a customer by PPPoE username (searching through device template)
func (db *DB) GetCustomerByPPPoE(pppoeUsername string) (*models.Customer, error) {
	query := `
		SELECT c.id, c.customer_code, c.name, c.email, c.phone, c.address, c.latitude, c.longitude,
		       c.package_id, c.username, c.password, c.status, c.join_date, c.balance, c.created_at, c.updated_at
		FROM customers c
		INNER JOIN device_customer_map dcm ON c.id = dcm.customer_id
		INNER JOIN devices d ON d.id = dcm.device_id
		WHERE d.template = ?
	`
	var c models.Customer
	var email, phone, address, username, pwd sql.NullString
	var packageID sql.NullInt64
	err := db.QueryRow(query, pppoeUsername).Scan(&c.ID, &c.CustomerCode, &c.Name, &email, &phone, &address, &c.Latitude, &c.Longitude,
		&packageID, &username, &pwd, &c.Status, &c.JoinDate, &c.Balance, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	if phone.Valid {
		c.Phone = phone.String
	}
	if address.Valid {
		c.Address = address.String
	}
	if packageID.Valid {
		c.PackageID = packageID.Int64
	}
	if username.Valid {
		c.Username = username.String
	}
	if pwd.Valid {
		c.Password = pwd.String
	}
	return &c, nil
}

// AssignDeviceToCustomer assigns a device to a customer
func (db *DB) AssignDeviceToCustomer(deviceID, customerID int64) error {
	_, err := db.Exec(`
		INSERT OR REPLACE INTO device_customer_map (device_id, customer_id)
		VALUES (?, ?)
	`, deviceID, customerID)
	return err
}

// UnassignDeviceFromCustomer removes device-customer assignment
func (db *DB) UnassignDeviceFromCustomer(deviceID, customerID int64) error {
	_, err := db.Exec(`
		DELETE FROM device_customer_map WHERE device_id = ? AND customer_id = ?
	`, deviceID, customerID)
	return err
}

// SyncCustomerToDevice synchronizes customer to device using PPPoE username for matching
func (db *DB) SyncCustomerToDevice(customerID int64, pppoeUsername string) error {
	// First get the customer to ensure they exist
	customer, err := db.GetCustomer(customerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %v", err)
	}

	// Get the device by PPPoE username (stored in template field)
	device, err := db.GetDeviceByTemplate(pppoeUsername)
	if err != nil {
		return fmt.Errorf("failed to get device by PPPoE username: %v", err)
	}

	// Assign the device to the customer
	if err := db.AssignDeviceToCustomer(device.ID, customer.ID); err != nil {
		return fmt.Errorf("failed to assign device to customer: %v", err)
	}

	// Update the device's customer_id field directly as well
	_, err = db.Exec(`UPDATE devices SET customer_id = ? WHERE id = ?`, customer.ID, device.ID)
	if err != nil {
		return fmt.Errorf("failed to update device customer_id: %v", err)
	}

	return nil
}

// UpdateDeviceLocation updates device location coordinates and address
func (db *DB) UpdateDeviceLocation(deviceID int64, latitude, longitude float64, address string) error {
	_, err := db.Exec(`
		UPDATE devices SET latitude = ?, longitude = ?, address = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, latitude, longitude, address, deviceID)
	return err
}

// CreateSupportTicket creates a new support ticket
func (db *DB) CreateSupportTicket(ticket *models.SupportTicket) (*models.SupportTicket, error) {
	// Generate ticket number
	if ticket.TicketNo == "" {
		var count int64
		db.QueryRow("SELECT COUNT(*) FROM support_tickets WHERE strftime('%Y%m', created_at) = strftime('%Y%m', 'now')").Scan(&count)
		ticket.TicketNo = fmt.Sprintf("TCK-%s-%04d", time.Now().Format("200601"), count+1)
	}

	result, err := db.Exec(`
		INSERT INTO support_tickets (ticket_no, customer_id, subject, description, category, priority, status, assigned_to)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ticket.TicketNo, ticket.CustomerID, ticket.Subject, ticket.Description, ticket.Category, ticket.Priority, ticket.Status, ticket.AssignedTo)
	if err != nil {
		return nil, err
	}
	id, _ := result.LastInsertId()
	ticket.ID = id
	return ticket, nil
}

// GetSupportTickets retrieves support tickets with optional filtering
func (db *DB) GetSupportTickets(customerID *int64, status string, limit, offset int) ([]*models.SupportTicket, int64, error) {
	var conditions []string
	var args []interface{}

	if customerID != nil {
		conditions = append(conditions, "customer_id = ?")
		args = append(args, *customerID)
	}
	if status != "" && status != "all" {
		conditions = append(conditions, "status = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int64
	db.QueryRow("SELECT COUNT(*) FROM support_tickets "+whereClause, args...).Scan(&total)

	query := fmt.Sprintf(`
		SELECT id, ticket_no, customer_id, subject, description, category, priority, status, assigned_to, resolution, created_at, updated_at, closed_at
		FROM support_tickets %s ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []*models.SupportTicket
	for rows.Next() {
		var t models.SupportTicket
		var assignedTo sql.NullInt64
		var resolution sql.NullString
		var closedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.TicketNo, &t.CustomerID, &t.Subject, &t.Description, &t.Category, &t.Priority, &t.Status, &assignedTo, &resolution, &t.CreatedAt, &t.UpdatedAt, &closedAt)
		if err != nil {
			return nil, 0, err
		}
		if assignedTo.Valid {
			t.AssignedTo = &assignedTo.Int64
		}
		if resolution.Valid {
			t.Resolution = resolution.String
		}
		if closedAt.Valid {
			t.ClosedAt = &closedAt.Time
		}
		tickets = append(tickets, &t)
	}
	return tickets, total, nil
}

// GetSupportTicket retrieves a support ticket by ID
func (db *DB) GetSupportTicket(id int64) (*models.SupportTicket, error) {
	var t models.SupportTicket
	var assignedTo sql.NullInt64
	var resolution sql.NullString
	var closedAt sql.NullTime
	err := db.QueryRow(`
		SELECT id, ticket_no, customer_id, subject, description, category, priority, status, assigned_to, resolution, created_at, updated_at, closed_at
		FROM support_tickets WHERE id = ?
	`, id).Scan(&t.ID, &t.TicketNo, &t.CustomerID, &t.Subject, &t.Description, &t.Category, &t.Priority, &t.Status, &assignedTo, &resolution, &t.CreatedAt, &t.UpdatedAt, &closedAt)
	if err != nil {
		return nil, err
	}
	if assignedTo.Valid {
		t.AssignedTo = &assignedTo.Int64
	}
	if resolution.Valid {
		t.Resolution = resolution.String
	}
	if closedAt.Valid {
		t.ClosedAt = &closedAt.Time
	}
	return &t, nil
}

// UpdateSupportTicket updates a support ticket
func (db *DB) UpdateSupportTicket(ticket *models.SupportTicket) error {
	var assignedTo interface{}
	if ticket.AssignedTo != nil {
		assignedTo = *ticket.AssignedTo
	} else {
		assignedTo = nil
	}

	_, err := db.Exec(`
		UPDATE support_tickets SET subject = ?, description = ?, category = ?, priority = ?, status = ?, assigned_to = ?, resolution = ?, updated_at = CURRENT_TIMESTAMP, closed_at = CASE WHEN ? IN ('resolved', 'closed') THEN CURRENT_TIMESTAMP ELSE closed_at END
		WHERE id = ?
	`, ticket.Subject, ticket.Description, ticket.Category, ticket.Priority, ticket.Status, assignedTo, ticket.Resolution, ticket.Status, ticket.ID)
	return err
}

// DeleteSupportTicket deletes a support ticket
func (db *DB) DeleteSupportTicket(id int64) error {
	_, err := db.Exec("DELETE FROM support_tickets WHERE id = ?", id)
	return err
}

// RecordBandwidthUsage records bandwidth usage snapshot
func (db *DB) RecordBandwidthUsage(deviceID int64, sent, received int64) error {
	_, err := db.Exec("INSERT INTO bandwidth_usage (device_id, bytes_sent, bytes_received) VALUES (?, ?, ?)", deviceID, sent, received)
	return err
}

// GetBandwidthHistory retrieves bandwidth usage history for a device
func (db *DB) GetBandwidthHistory(deviceID int64, limit int) ([]models.BandwidthRecord, error) {
	rows, err := db.Query("SELECT timestamp, bytes_sent, bytes_received FROM bandwidth_usage WHERE device_id = ? ORDER BY timestamp DESC LIMIT ?", deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.BandwidthRecord
	for rows.Next() {
		var r models.BandwidthRecord
		if err := rows.Scan(&r.Timestamp, &r.BytesSent, &r.BytesReceived); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, nil
}

// GetNetworkStats retrieves aggregated network statistics for today
func (db *DB) GetNetworkStats() (*models.NetworkStats, error) {
	stats := &models.NetworkStats{
		TopUsers:     []models.UsageStat{},
		TrafficChart: []models.UsageStat{},
	}

	// 1. Total Usage Today (Sum of usage per device)
	// We calculate specific usage as MAX - MIN for today for each device
	queryTotal := `
		SELECT 
			SUM(max_rx - min_rx) as total_dl,
			SUM(max_tx - min_tx) as total_ul
		FROM (
			SELECT 
				MAX(bytes_received) as max_rx, MIN(bytes_received) as min_rx,
				MAX(bytes_sent) as max_tx, MIN(bytes_sent) as min_tx
			FROM bandwidth_usage
			WHERE timestamp >= date('now', 'start of day')
			GROUP BY device_id
		)
	`
	var totalDl, totalUl sql.NullInt64
	db.QueryRow(queryTotal).Scan(&totalDl, &totalUl)
	stats.TotalDownload = totalDl.Int64
	stats.TotalUpload = totalUl.Int64

	// 2. Top Users
	queryTop := `
		SELECT c.name, (MAX(b.bytes_received) - MIN(b.bytes_received)) as usage
		FROM bandwidth_usage b
		JOIN devices d ON b.device_id = d.id
		JOIN customers c ON d.customer_id = c.id
		WHERE b.timestamp >= date('now', 'start of day')
		GROUP BY c.id
		ORDER BY usage DESC
		LIMIT 5
	`
	rows, err := db.Query(queryTop)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s models.UsageStat
			var usage sql.NullInt64
			rows.Scan(&s.Label, &usage)
			s.BytesReceived = usage.Int64 // Just use RX for ranking
			stats.TopUsers = append(stats.TopUsers, s)
		}
	}

	// 3. Hourly Chart (Simplified: taking max of each hour - min of each hour? No, that's tricky)
	// Let's just take the MAX counter value at each hour? No.
	// We need Sum of Deltas per hour. Very complex in one query.
	// Simple approach: Count number of records? No.
	// Alternative: Just show Total Bytes Recorded (if we change scheduler to record Delta).

	// Since we record COUNTERS, charting "Traffic Rate" is hard without processing.
	// Fallback: Just return empty chart or mock for now, or use Latest Speed if we had it.
	// Actually, let's skip chart data for now or return 0 to avoid wrong data.
	// We will fill chart labels 00-23.
	for i := 0; i < 24; i++ {
		stats.TrafficChart = append(stats.TrafficChart, models.UsageStat{
			Label:         fmt.Sprintf("%02d:00", i),
			BytesReceived: 0,
			BytesSent:     0,
		})
	}

	return stats, nil
}

// GetSetting retrieves a configuration value by key
func (db *DB) GetSetting(key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SaveSetting saves or updates a configuration value
func (db *DB) SaveSetting(key, value string) error {
	_, err := db.Exec(`
		INSERT INTO settings (key, value, updated_at) 
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET 
			value = excluded.value,
			updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetSettings retrieves all settings
func (db *DB) GetSettings() (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}

// GetUserByUsername retrieves a user by username
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, password, email, role, last_login, created_at, updated_at FROM users WHERE username = ?`
	var user models.User
	var lastLogin sql.NullTime
	var email sql.NullString

	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &email, &user.Role, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if email.Valid {
		user.Email = email.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

// UpdateUser updates a user's information
func (db *DB) UpdateUser(user *models.User) error {
	_, err := db.Exec(`
		UPDATE users SET 
			password = ?, 
			email = ?, 
			role = ?, 
			last_login = ?, 
			updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`,
		user.Password, user.Email, user.Role, user.LastLogin, user.ID,
	)
	return err
}

// CreateUser creates a new user
func (db *DB) CreateUser(user *models.User) error {
	// Hash the password before storing
	if user.Password != "" {
		hashedPassword, err := db.HashPassword(user.Password)
		if err != nil {
			return err
		}
		user.Password = hashedPassword
	}

	_, err := db.Exec(`
		INSERT INTO users (username, password, email, role, created_at, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		user.Username, user.Password, user.Email, user.Role,
	)
	return err
}

// HashPassword hashes a password using bcrypt
func (db *DB) HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// GetUserByID retrieves a user by ID
func (db *DB) GetUserByID(userID int64) (*models.User, error) {
	query := `SELECT id, username, password, email, role, last_login, created_at, updated_at FROM users WHERE id = ?`
	var user models.User
	var lastLogin sql.NullTime
	var email sql.NullString

	err := db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Password, &email, &user.Role, &lastLogin, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}

	if email.Valid {
		user.Email = email.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

// EnsureDefaultAdmin ensures that a default admin user exists
// This is called during database initialization
func (db *DB) EnsureDefaultAdmin(username, password string) error {
	// Check if admin user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %v", err)
	}

	// If user already exists, no need to create
	if count > 0 {
		return nil
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Create the admin user
	_, err = db.Exec(`
		INSERT INTO users (username, password, email, role, created_at, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		username, string(hashedPassword), "admin@go-acs.local", "admin",
	)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %v", err)
	}

	fmt.Printf("✓ Default admin user '%s' created successfully\n", username)
	return nil
}
