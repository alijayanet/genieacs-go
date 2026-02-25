package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"go-acs/internal/config"
	"go-acs/internal/database"
	"go-acs/internal/handlers"
	"go-acs/internal/mailer"
	"go-acs/internal/middleware"
	"go-acs/internal/mikrotik"
	"go-acs/internal/notification/fcm"
	"go-acs/internal/notification/telegram"
	"go-acs/internal/notification/whatsapp"
	"go-acs/internal/payment/tripay"
	"go-acs/internal/scheduler"
	"go-acs/internal/tr069"
	"go-acs/internal/websocket"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	// Print banner
	printBanner()

	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Database initialized successfully")

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	log.Println("✓ WebSocket hub started")

	// Initialize TR-069 server
	tr069Server := tr069.NewServer(cfg.TR069Port, db, wsHub)
	go tr069Server.Start()

	log.Printf("✓ TR-069 server started on port %d", cfg.TR069Port)

	// Initialize Mailer (Mock for now, can be configured via env)
	mailConfig := mailer.Config{
		Host:     "", // Empty host triggers mock mode
		Port:     587,
		Username: "user",
		Password: "password",
		From:     "noreply@go-acs.local",
	}
	mailService := mailer.New(mailConfig)

	// Load settings from database
	settings, err := db.GetSettings()
	if err == nil {
		if v, ok := settings["mikrotik_host"]; ok && v != "" {
			cfg.MikrotikHost = v
		}
		if v, ok := settings["mikrotik_user"]; ok && v != "" {
			cfg.MikrotikUser = v
		}
		if v, ok := settings["mikrotik_pass"]; ok && v != "" {
			cfg.MikrotikPass = v
		}
		if v, ok := settings["mikrotik_port"]; ok && v != "" {
			if port, err := strconv.Atoi(v); err == nil {
				cfg.MikrotikPort = port
			}
		}
		if v, ok := settings["tripay_api_key"]; ok && v != "" {
			cfg.TripayAPIKey = v
		}
	}

	// Initialize MikroTik Client
	mikrotikClient := mikrotik.New(cfg)

	// Initialize Payment Gateway (Tripay)
	tripayGateway := tripay.New(cfg)

	// Initialize WhatsApp Client
	waClient := whatsapp.New(cfg)

	// Initialize FCM Client
	fcmClient := fcm.New(cfg)

	// Initialize Telegram Client
	telegramClient := telegram.New(cfg.TelegramToken, cfg.TelegramChatID)

	// Initialize HTTP handlers
	h := handlers.NewHandler(db, wsHub, mailService, mikrotikClient, tripayGateway, waClient, fcmClient, telegramClient, cfg)

	// Initialize Scheduler
	sched := scheduler.New(h)
	sched.Start()
	log.Println("✓ Scheduler started")

	// Setup router
	router := setupRouter(h, wsHub)

	// Setup CORS with more restrictive settings
	allowedOrigins := []string{
		"http://localhost:8080",
		"http://localhost:3000",
	}
	
	// Add additional origins from environment if needed
	if origin := os.Getenv("ALLOWED_ORIGINS"); origin != "" {
		allowedOrigins = append(allowedOrigins, strings.Split(origin, ",")...)
	}
	
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})

	// Apply authentication middleware
	authMiddleware := middleware.AuthMiddleware(cfg.JWTSecret)
	handler := c.Handler(authMiddleware(router))

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("✓ HTTP server starting on port %d", cfg.ServerPort)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🌐 Web UI: http://localhost:%d", cfg.ServerPort)
	log.Printf("🔧 API: http://localhost:%d/api", cfg.ServerPort)
	log.Printf("📡 TR-069: http://localhost:%d", cfg.TR069Port)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("\n🛑 Shutting down server...")
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(addr, handler))
}

func setupRouter(h *handlers.Handler, wsHub *websocket.Hub) *mux.Router {
	router := mux.NewRouter()

	// Serve static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Serve favicon
	router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		// Return a simple 16x16 transparent PNG as favicon to avoid 404 errors
		w.Header().Set("Content-Type", "image/x-icon")
		w.WriteHeader(http.StatusOK)
		// Small 16x16 transparent PNG (minimal valid PNG)
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0xF3, 0xFF, 0x61, 0x00, 0x00, 0x00, 0x04, 0x73, 0x42, 0x49, 0x54, 0x08, 0x08, 0x08, 0x08, 0x7C, 0x08, 0x64, 0x88, 0x00, 0x00, 0x00, 0x09, 0x70, 0x48, 0x59, 0x73, 0x00, 0x00, 0x0B, 0x13, 0x00, 0x00, 0x0B, 0x13, 0x01, 0x00, 0x9A, 0x9C, 0x18, 0x00, 0x00, 0x00, 0x1D, 0x49, 0x44, 0x41, 0x54, 0x78, 0xDA, 0xEC, 0xC1, 0x01, 0x0D, 0x00, 0x00, 0x00, 0xC2, 0xA0, 0xF7, 0x4F, 0x6D, 0x0E, 0x37, 0xA0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xBE, 0x0D, 0x21, 0x00, 0x00, 0x01, 0xD4, 0x97, 0xE0, 0xE3, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82})
	}).Methods("GET")

	// Serve web UI
	router.HandleFunc("/", h.ServeIndex).Methods("GET")
	router.HandleFunc("/dashboard", h.ServeDashboard).Methods("GET")
	router.HandleFunc("/devices", h.ServeDevices).Methods("GET")
	router.HandleFunc("/device/{id}", h.ServeDeviceDetail).Methods("GET")
	router.HandleFunc("/provisions", h.ServeProvisions).Methods("GET")
	router.HandleFunc("/packages", h.ServePackages).Methods("GET")
	router.HandleFunc("/customers", h.ServeCustomers).Methods("GET")
	router.HandleFunc("/billing", h.ServeBilling).Methods("GET")
	router.HandleFunc("/map", h.ServeMap).Methods("GET")
	router.HandleFunc("/portal", h.ServePortal).Methods("GET")
	router.HandleFunc("/portal/login", h.ServePortalLogin).Methods("GET")
	router.HandleFunc("/tasks", h.ServeTasks).Methods("GET")
	router.HandleFunc("/tickets", h.ServeTickets).Methods("GET")
	router.HandleFunc("/settings", h.ServeSettings).Methods("GET")
	router.HandleFunc("/logs", h.ServeLogs).Methods("GET")
	router.HandleFunc("/update", h.ServeUpdate).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Admin Authentication
	api.HandleFunc("/auth/login", h.Login).Methods("POST")
	api.HandleFunc("/auth/logout", h.Logout).Methods("POST")

	// Customer Portal Authentication
	api.HandleFunc("/portal/auth/login", h.CustomerLogin).Methods("POST")
	api.HandleFunc("/portal/auth/logout", h.CustomerLogout).Methods("POST")

	// Customer Portal API
	api.HandleFunc("/portal/dashboard", h.GetPortalDashboard).Methods("GET")
	api.HandleFunc("/portal/invoices", h.GetPortalInvoices).Methods("GET")
	api.HandleFunc("/portal/wifi/ssid", h.UpdatePortalWiFiSSID).Methods("PUT")
	api.HandleFunc("/portal/wifi/password", h.UpdatePortalWiFiPassword).Methods("PUT")
	api.HandleFunc("/portal/tickets", h.CreatePortalTicket).Methods("POST")

	// Dashboard
	api.HandleFunc("/dashboard/stats", h.GetDashboardStats).Methods("GET")

	// Device/ONU management
	api.HandleFunc("/devices", h.GetDevices).Methods("GET")
	api.HandleFunc("/devices", h.CreateDevice).Methods("POST")
	api.HandleFunc("/devices/{id}", h.GetDevice).Methods("GET")
	api.HandleFunc("/devices/{id}", h.UpdateDevice).Methods("PUT")
	api.HandleFunc("/devices/{id}", h.DeleteDevice).Methods("DELETE")
	api.HandleFunc("/devices/{id}/status", h.GetDeviceStatus).Methods("GET")
	api.HandleFunc("/devices/{id}/logs", h.GetDeviceLogs).Methods("GET")
	api.HandleFunc("/devices/{id}/status-logs", h.GetDeviceStatusLogs).Methods("GET")
	api.HandleFunc("/devices/{id}/pon", h.GetDevicePON).Methods("GET")
	api.HandleFunc("/devices/{id}/clients", h.GetDeviceClients).Methods("GET")
	api.HandleFunc("/devices/{id}/reboot", h.RebootDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/factory-reset", h.FactoryResetDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/refresh", h.RefreshDevice).Methods("POST")
	api.HandleFunc("/devices/{id}/parameters", h.GetDeviceParameters).Methods("GET")

	// WiFi configuration
	api.HandleFunc("/devices/{id}/wifi", h.GetWiFiConfig).Methods("GET")
	api.HandleFunc("/devices/{id}/wifi", h.UpdateWiFiConfig).Methods("PUT")
	api.HandleFunc("/devices/{id}/wifi/ssid", h.UpdateSSID).Methods("PUT")
	api.HandleFunc("/devices/{id}/wifi/password", h.UpdateWiFiPassword).Methods("PUT")

	// WAN configuration
	api.HandleFunc("/devices/{id}/wan", h.GetWANConfigs).Methods("GET")
	api.HandleFunc("/devices/{id}/wan", h.CreateWANConfig).Methods("POST")
	api.HandleFunc("/devices/{id}/wan/{wanId}", h.GetWANConfig).Methods("GET")
	api.HandleFunc("/devices/{id}/wan/{wanId}", h.UpdateWANConfig).Methods("PUT")
	api.HandleFunc("/devices/{id}/wan/{wanId}", h.DeleteWANConfig).Methods("DELETE")
	// WAN/PPPoE details
	api.HandleFunc("/devices/{id}/wan-details", h.GetDeviceWAN).Methods("GET")

	// LAN configuration
	api.HandleFunc("/devices/{id}/lan", h.GetLANConfig).Methods("GET")
	api.HandleFunc("/devices/{id}/lan", h.UpdateLANConfig).Methods("PUT")

	// Device parameters
	api.HandleFunc("/devices/{id}/parameters", h.GetDeviceParameters).Methods("GET")
	api.HandleFunc("/devices/{id}/parameters", h.SetDeviceParameters).Methods("POST")
	api.HandleFunc("/devices/{id}/parameters/{path}", h.GetDeviceParameter).Methods("GET")
	api.HandleFunc("/devices/template/{template}", h.GetDeviceByTemplate).Methods("GET")
	api.HandleFunc("/customers/pppoe/{pppoeUsername}", h.GetCustomerByPPPoE).Methods("GET")

	// Firmware management
	api.HandleFunc("/devices/{id}/firmware", h.GetFirmwareInfo).Methods("GET")
	api.HandleFunc("/devices/{id}/firmware/upgrade", h.UpgradeFirmware).Methods("POST")

	// Tasks/Commands
	api.HandleFunc("/devices/{id}/tasks", h.GetDeviceTasks).Methods("GET")
	api.HandleFunc("/devices/{id}/tasks", h.CreateDeviceTask).Methods("POST")
	api.HandleFunc("/tasks/{taskId}", h.GetTask).Methods("GET")
	api.HandleFunc("/tasks/{taskId}", h.DeleteTask).Methods("DELETE")

	// Presets/Provisions
	api.HandleFunc("/presets", h.GetPresets).Methods("GET")
	api.HandleFunc("/presets", h.CreatePreset).Methods("POST")
	api.HandleFunc("/presets/{id}", h.GetPreset).Methods("GET")
	api.HandleFunc("/presets/{id}", h.UpdatePreset).Methods("PUT")
	api.HandleFunc("/presets/{id}", h.DeletePreset).Methods("DELETE")

	// Logs
	api.HandleFunc("/logs", h.GetLogs).Methods("GET")
	api.HandleFunc("/devices/{id}/logs", h.GetDeviceLogs).Methods("GET")

	// ============== Billing API Routes ==============

	// Packages
	api.HandleFunc("/packages", h.GetPackages).Methods("GET")
	api.HandleFunc("/packages", h.CreatePackage).Methods("POST")
	api.HandleFunc("/packages/{id}", h.GetPackageByID).Methods("GET")
	api.HandleFunc("/packages/{id}", h.UpdatePackage).Methods("PUT")
	api.HandleFunc("/packages/{id}", h.DeletePackage).Methods("DELETE")

	//Customers
	api.HandleFunc("/customers", h.GetCustomers).Methods("GET")
	api.HandleFunc("/customers", h.CreateCustomer).Methods("POST")
	api.HandleFunc("/customers/{id}", h.GetCustomer).Methods("GET")
	api.HandleFunc("/customers/{id}", h.UpdateCustomer).Methods("PUT")
	api.HandleFunc("/customers/{id}", h.DeleteCustomer).Methods("DELETE")
	api.HandleFunc("/customers/{id}/isolir", h.IsolirCustomer).Methods("POST")
	api.HandleFunc("/customers/{id}/unsuspend", h.UnsuspendCustomer).Methods("POST")
	api.HandleFunc("/customers/{id}/unsuspend-without-payment", h.UnsuspendCustomerWithoutPayment).Methods("POST")
	api.HandleFunc("/customers/{id}/location", h.UpdateCustomerLocation).Methods("PUT")
	api.HandleFunc("/customers/{id}/fcm", h.UpdateCustomerFCM).Methods("POST")
	api.HandleFunc("/customers/{id}/sync-device", h.SyncCustomerToDeviceByPPPoE).Methods("POST")
	api.HandleFunc("/locations", h.GetLocations).Methods("GET")

	// Invoices
	api.HandleFunc("/invoices", h.GetInvoices).Methods("GET")
	api.HandleFunc("/invoices", h.CreateInvoice).Methods("POST")
	api.HandleFunc("/invoices/generate", h.GenerateMonthlyInvoices).Methods("POST")
	api.HandleFunc("/invoices/{id}", h.GetInvoice).Methods("GET")
	api.HandleFunc("/invoices/{id}/pay", h.MarkInvoicePaid).Methods("POST")

	// Payments
	api.HandleFunc("/payments", h.GetPayments).Methods("GET")
	api.HandleFunc("/payments", h.CreatePayment).Methods("POST")
	api.HandleFunc("/payment/channels", h.GetPaymentChannels).Methods("GET")
	api.HandleFunc("/invoices/{id}/pay/online", h.CreatePaymentTransaction).Methods("POST")

	// Callbacks (Public)
	api.HandleFunc("/callbacks/tripay", h.HandleTripayCallback).Methods("POST")

	// Billing Stats & Actions
	api.HandleFunc("/billing/stats", h.GetBillingStats).Methods("GET")
	api.HandleFunc("/network/stats", h.GetNetworkOverview).Methods("GET")
	api.HandleFunc("/billing/batch-isolir", h.BatchIsolirOverdue).Methods("POST")

	// Customer Portal API
	api.HandleFunc("/portal/auth/login", h.CustomerLogin).Methods("POST")
	api.HandleFunc("/portal/dashboard", h.GetCustomerDashboard).Methods("GET")
	api.HandleFunc("/portal/invoices", h.GetCustomerInvoices).Methods("GET")
	api.HandleFunc("/portal/wifi", h.GetCustomerWiFi).Methods("GET")
	api.HandleFunc("/portal/wifi", h.UpdateCustomerWiFi).Methods("PUT")

	// Mobile API
	api.HandleFunc("/mobile/usage", h.GetMobileUsage).Methods("GET")

	// Support Tickets
	api.HandleFunc("/tickets", h.GetSupportTickets).Methods("GET")
	api.HandleFunc("/tickets", h.CreateSupportTicket).Methods("POST")
	api.HandleFunc("/tickets/{id}", h.GetSupportTicket).Methods("GET")
	api.HandleFunc("/tickets/{id}", h.UpdateSupportTicket).Methods("PUT")
	api.HandleFunc("/tickets/{id}", h.DeleteSupportTicket).Methods("DELETE")

	// Device Location (for map)
	api.HandleFunc("/devices/{id}/location", h.UpdateDeviceLocation).Methods("PUT")

	// System Settings
	api.HandleFunc("/settings", h.GetSettings).Methods("GET")
	api.HandleFunc("/settings", h.SaveSettings).Methods("POST")
	api.HandleFunc("/settings/password", h.ChangeAdminPassword).Methods("POST")
	api.HandleFunc("/mikrotik/test", h.TestMikrotik).Methods("GET")
	api.HandleFunc("/mikrotik/profiles", h.GetMikrotikProfiles).Methods("GET")
	api.HandleFunc("/mikrotik/profiles", h.CreateMikrotikProfile).Methods("POST")

	// Update API
	api.HandleFunc("/update/check", h.CheckForUpdates).Methods("GET")
	api.HandleFunc("/update/perform", h.PerformUpdate).Methods("POST")
	api.HandleFunc("/update/rebuild", h.RebuildApplication).Methods("POST")
	api.HandleFunc("/update/restart", h.RestartService).Methods("POST")

	// LAN Configuration
	api.HandleFunc("/devices/{id}/lan", h.GetLANConfig).Methods("GET")
	api.HandleFunc("/devices/{id}/lan", h.UpdateLANConfig).Methods("PUT")

	// Port Forwarding / NAT
	api.HandleFunc("/devices/{id}/port-forwarding", h.GetPortForwardingRules).Methods("GET")
	api.HandleFunc("/devices/{id}/port-forwarding", h.CreatePortForwardingRule).Methods("POST")

	// Bridge Mode
	api.HandleFunc("/devices/{id}/bridge-mode", h.SetBridgeMode).Methods("PUT")

	// QoS
	api.HandleFunc("/devices/{id}/qos", h.GetQoSConfig).Methods("GET")
	api.HandleFunc("/devices/{id}/qos", h.UpdateQoSConfig).Methods("PUT")

	// WebSocket
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleWebSocket(wsHub, w, r)
	})

	return router
}

func printBanner() {
	banner := `
   ██████╗  ██████╗       █████╗  ██████╗███████╗
  ██╔════╝ ██╔═══██╗     ██╔══██╗██╔════╝██╔════╝
  ██║  ███╗██║   ██║     ███████║██║     ███████╗
  ██║   ██║██║   ██║     ██╔══██║██║     ╚════██║
  ╚██████╔╝╚██████╔╝     ██║  ██║╚██████╗███████║
   ╚═════╝  ╚═════╝      ╚═╝  ╚═╝ ╚═════╝╚══════╝
  
  Go-based Auto Configuration Server for ONU Management
  Version: 1.0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`
	fmt.Println(banner)
}
