package tr069

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-acs/internal/database"
	"go-acs/internal/models"
	"go-acs/internal/websocket"
)

// Server represents the TR-069 ACS server
type Server struct {
	Port     int
	DB       *database.DB
	WSHub    *websocket.Hub
	sessions sync.Map // Map of session ID to session data
}

// Session represents a TR-069 session
type Session struct {
	ID           string
	DeviceID     int64
	SerialNumber string
	StartTime    time.Time
	LastActivity time.Time
	PendingTasks []*models.DeviceTask
	CurrentTask  *models.DeviceTask
}

// NewServer creates a new TR-069 server
func NewServer(port int, db *database.DB, wsHub *websocket.Hub) *Server {
	return &Server{
		Port:  port,
		DB:    db,
		WSHub: wsHub,
	}
}

// Start starts the TR-069 server
func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	mux.HandleFunc("/tr069", s.handleRequest)
	mux.HandleFunc("/acs", s.handleRequest)

	// Health check endpoints for testing connectivity
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Health check request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"go-acs-tr069","port":7547}`))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Status check request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"go-acs-tr069","port":7547}`))
	})

	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("✓ TR-069 ACS server listening on %s", addr)
	log.Printf("  Endpoints: /, /tr069, /acs, /health, /status")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("TR-069 server error: %v", err)
	}
}

// handleRequest handles incoming TR-069 requests
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Log the request with more details for debugging
	log.Printf("╔══════════════════════════════════════════════════════════════")
	log.Printf("║ TR-069 Request: %s %s", r.Method, r.URL.Path)
	log.Printf("║ From: %s", r.RemoteAddr)
	log.Printf("║ User-Agent: %s", r.UserAgent())
	log.Printf("║ Content-Length: %d", r.ContentLength)
	log.Printf("║ Content-Type: %s", r.Header.Get("Content-Type"))
	log.Printf("╚══════════════════════════════════════════════════════════════")

	// Set common headers
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("SOAPAction", "")
	w.Header().Set("Connection", "keep-alive")

	// Handle empty POST (session initiation) or GET requests
	if r.Method == "GET" || r.ContentLength == 0 {
		log.Printf("→ Empty request, sending 204 No Content")
		// Send empty response to indicate no pending commands
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		// Empty request - CPE is asking for commands
		s.handleEmptyRequest(w, r)
		return
	}

	// Log the request body for debugging
	log.Printf("TR-069 Request Body:\n%s", string(body))

	// Parse the SOAP envelope
	envelope, err := parseSOAPEnvelope(body)
	if err != nil {
		log.Printf("Error parsing SOAP envelope: %v", err)
		http.Error(w, "Invalid SOAP request", http.StatusBadRequest)
		return
	}

	// Handle the request based on the method
	response := s.handleSOAPRequest(envelope, r)

	// Send response
	if response != nil {
		responseBytes, err := xml.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Printf("Error marshaling response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Add XML declaration
		fullResponse := []byte(xml.Header + string(responseBytes))
		log.Printf("TR-069 Response:\n%s", string(fullResponse))

		w.WriteHeader(http.StatusOK)
		w.Write(fullResponse)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) handleEmptyRequest(w http.ResponseWriter, r *http.Request) {
	// CPE is asking for pending commands
	clientIP := strings.Split(r.RemoteAddr, ":")[0]

	// Find the device for this session
	var deviceID int64
	if sessionData, ok := s.sessions.Load(clientIP); ok {
		session := sessionData.(*Session)
		deviceID = session.DeviceID
	} else {
		// Try to find device by IP directly if session lost
		devices, _, _ := s.DB.GetDevices("online", "", 500, 0)
		for _, d := range devices {
			if d.IPAddress == clientIP {
				deviceID = d.ID
				break
			}
		}
	}

	if deviceID == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Fetch pending tasks
	tasks, err := s.DB.GetPendingTasks(deviceID)
	if err != nil || len(tasks) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Send the first task
	task := tasks[0]
	s.sendTask(w, task)
}

func (s *Server) sendTask(w http.ResponseWriter, task *models.DeviceTask) {
	log.Printf("Sending task %d (Type: %s) to device %d", task.ID, task.Type, task.DeviceID)

	var response []byte
	id := fmt.Sprintf("task-%d", task.ID)

	switch task.Type {
	case models.TaskGetParameterValues:
		var paths []string
		json.Unmarshal(task.Parameters, &paths)
		response = CreateGetParameterValues(id, paths)
	case models.TaskSetParameterValues:
		var params map[string]interface{}
		json.Unmarshal(task.Parameters, &params)
		response = CreateSetParameterValues(id, params)
	case models.TaskReboot:
		response = CreateReboot(id, id)
	case models.TaskFactoryReset:
		response = CreateFactoryReset(id)
	case models.TaskDownload:
		var download struct {
			URL      string `json:"url"`
			FileType string `json:"fileType"`
			FileSize int64  `json:"fileSize"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		json.Unmarshal(task.Parameters, &download)
		if download.FileType == "" {
			download.FileType = "1 Firmware Upgrade Image"
		}
		response = CreateDownload(id, download.FileType, download.URL, download.FileSize, download.Username, download.Password)
	case models.TaskRefresh:
		// Build comprehensive parameter list using vendor-aware resolver
		device, _ := s.DB.GetDevice(task.DeviceID)
		standardParams := GetStandardONUParameters()
		commonParams := GetONUCommonParameters()
		vendorResolver := NewVendorSpecificParameterResolver(device.Manufacturer, device.ModelName)
		vendorParams := vendorResolver.GetVendorSpecificParameters()

		allPaths := make([]string, 0)
		for _, category := range standardParams {
			for _, param := range category.Parameters {
				allPaths = append(allPaths, param.Path)
			}
		}
		allPaths = append(allPaths, commonParams...)
		allPaths = append(allPaths, vendorParams...)

		legacyPaths := []string{
			"InternetGatewayDevice.LANDevice.1.WLANConfiguration.",
			"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.",
			"InternetGatewayDevice.LANDevice.1.Hosts.",
			"InternetGatewayDevice.DeviceInfo.",
		}
		allPaths = append(allPaths, legacyPaths...)
		if strings.Contains(strings.ToUpper(device.Manufacturer), "HUAWEI") {
			allPaths = append(allPaths, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.X_HW_VenderClassID")
		}
		response = CreateGetParameterValues(id, allPaths)
	default:
		log.Printf("Unsupported task type: %s", task.Type)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Update task status to running
	now := time.Now()
	task.Status = models.TaskRunning
	task.StartedAt = &now
	s.DB.UpdateTask(task)

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func (s *Server) handleSOAPRequest(envelope *SOAPEnvelope, r *http.Request) *SOAPEnvelope {
	// Determine the CWMP method being called
	body := envelope.Body.InnerXML

	switch {
	case strings.Contains(string(body), "Inform"):
		return s.handleInform(envelope, r)
	case strings.Contains(string(body), "GetRPCMethodsResponse"):
		return s.handleGetRPCMethodsResponse(envelope)
	case strings.Contains(string(body), "TransferComplete"):
		return s.handleTransferComplete(envelope)
	case strings.Contains(string(body), "GetParameterValuesResponse"):
		s.handleGetParameterValuesResponse(envelope, r)
		return nil // We'll send next task in handleRequest/empty post
	case strings.Contains(string(body), "SetParameterValuesResponse"):
		s.handleSetParameterValuesResponse(envelope, r)
		return nil
	case strings.Contains(string(body), "RebootResponse"):
		s.handleRebootResponse(envelope, r)
		return nil
	case strings.Contains(string(body), "FactoryResetResponse"):
		s.handleFactoryResetResponse(envelope, r)
		return nil
	case strings.Contains(string(body), "Fault"):
		s.handleFault(envelope, r)
		return nil
	default:
		log.Printf("Unknown CWMP method in body: %s", string(body)[:min(200, len(body))])
		return nil
	}
}

func (s *Server) handleFault(envelope *SOAPEnvelope, _ *http.Request) {
	log.Printf("Fault received from device: %s", string(envelope.Body.InnerXML))
	// Try to identify task from Envelope ID
	if envelope.Header != nil && strings.HasPrefix(envelope.Header.ID, "task-") {
		taskIDStr := strings.TrimPrefix(envelope.Header.ID, "task-")
		if taskID, err := strconv.ParseInt(taskIDStr, 10, 64); err == nil {
			now := time.Now()
			task := &models.DeviceTask{
				ID:          taskID,
				Status:      models.TaskFailed,
				CompletedAt: &now,
				Error:       "CWMP Fault: " + string(envelope.Body.InnerXML),
			}
			s.DB.UpdateTask(task)
		}
	}
}

// handleInform handles the Inform RPC from CPE
func (s *Server) handleInform(envelope *SOAPEnvelope, r *http.Request) *SOAPEnvelope {
	// Parse the Inform message
	inform, err := parseInform(envelope.Body.InnerXML)
	if err != nil {
		log.Printf("Error parsing Inform: %v", err)
		return nil
	}

	log.Printf("Inform received from device: %s (SN: %s)",
		inform.DeviceId.Manufacturer, inform.DeviceId.SerialNumber)

	// Decode Serial Number (Logic from GenieACS)
	// GPON serials often start with 4-byte manufacturer code in hex
	rawSN := inform.DeviceId.SerialNumber
	sn := decodeSerialNumber(rawSN)

	// Find or create the device in the database
	device, err := s.DB.GetDeviceBySerial(sn)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			// New device - initialize the object
			device = &models.Device{
				SerialNumber: sn,
				Status:       models.StatusOnline,
				Manufacturer: inform.DeviceId.Manufacturer,
				OUI:          inform.DeviceId.OUI,
				ProductClass: inform.DeviceId.ProductClass,
			}

			device, err = s.DB.CreateDevice(device)
			if err != nil {
				log.Printf("Error creating device %s: %v", inform.DeviceId.SerialNumber, err)
			} else {
				log.Printf("New device registered: %s", device.SerialNumber)
				s.DB.CreateLog(&device.ID, "info", "device",
					fmt.Sprintf("New device registered: %s", device.SerialNumber), "")
			}
		} else {
			// Database error (missing columns, etc)
			log.Printf("Database error fetching device %s: %v", inform.DeviceId.SerialNumber, err)
			// Don't return, try to proceed with minimal info or log it clearly
		}
	}

	if device != nil {
		// Update existing device
		now := time.Now()
		device.Status = models.StatusOnline
		device.LastInform = &now
		device.LastContact = &now
		device.IPAddress = strings.Split(r.RemoteAddr, ":")[0]
		device.ClientCount = 0 // Reset for summation

		// Update device info from Inform using new parameter parser
		parser := NewDeviceParameterParser(device, device.Manufacturer, device.ModelName)
		for _, param := range inform.ParameterList.ParameterValueStruct {
			parser.ParseParameter(param.Name, param.Value)
		}

		// Get parsed device data and update main device
		parsedDevice := parser.GetDeviceData()
		if parsedDevice != nil {
			// Update device fields that were parsed from Inform
			if parsedDevice.RXPower != 0 {
				device.RXPower = parsedDevice.RXPower
			}
			if parsedDevice.TXPower != 0 {
				device.TXPower = parsedDevice.TXPower
			}
			if parsedDevice.OpticalTemperature != 0 {
				device.OpticalTemperature = parsedDevice.OpticalTemperature
			}
			if parsedDevice.OpticalVoltage != 0 {
				device.OpticalVoltage = parsedDevice.OpticalVoltage
			}
			if parsedDevice.OpticalCurrent != 0 {
				device.OpticalCurrent = parsedDevice.OpticalCurrent
			}
			if parsedDevice.Distance != 0 {
				device.Distance = parsedDevice.Distance
			}
		}

		s.DB.UpdateDevice(device)
		log.Printf("Device updated: %s (Status: online, RX: %.2f dBm, TX: %.2f dBm)", device.SerialNumber, device.RXPower, device.TXPower)

		// Store session for this IP address so we can identify device in subsequent responses
		clientIP := strings.Split(r.RemoteAddr, ":")[0]
		s.sessions.Store(clientIP, &Session{
			DeviceID:     device.ID,
			SerialNumber: device.SerialNumber,
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		})
	}

	// Store parameters from Inform
	if device != nil {
		for _, param := range inform.ParameterList.ParameterValueStruct {
			s.DB.SetDeviceParameter(device.ID, param.Name, param.Value, "string", true)
		}
	}

	// Notify via WebSocket
	if s.WSHub != nil && device != nil {
		s.WSHub.Broadcast(websocket.Message{
			Type:     "device_update",
			DeviceID: device.ID,
			Data: map[string]interface{}{
				"status":      "online",
				"lastContact": time.Now(),
				"event":       "inform",
			},
		})
	}

	// Log the Inform event
	if device != nil {
		eventCodes := ""
		for _, event := range inform.Event.EventStruct {
			eventCodes += event.EventCode + " "
		}
		s.DB.CreateLog(&device.ID, "info", "inform",
			fmt.Sprintf("Inform received: %s", strings.TrimSpace(eventCodes)), "")

		// Run provisioning/bootstrap logic (Logic from Provision script)
		s.bootstrapDevice(device)
	}

	// Return InformResponse
	return createInformResponse(envelope.Header)
}

func (s *Server) handleGetRPCMethodsResponse(_ *SOAPEnvelope) *SOAPEnvelope {
	log.Println("GetRPCMethodsResponse received")
	return nil
}

func (s *Server) handleTransferComplete(envelope *SOAPEnvelope) *SOAPEnvelope {
	log.Println("TransferComplete received")
	return createTransferCompleteResponse(envelope.Header)
}

func (s *Server) handleGetParameterValuesResponse(envelope *SOAPEnvelope, r *http.Request) *SOAPEnvelope {
	log.Println("GetParameterValuesResponse received")

	// Parse the response to extract parameters
	parsed, err := ParseGetParameterValuesResponse(envelope.Body.InnerXML)
	if err != nil {
		log.Printf("Error parsing GetParameterValuesResponse: %v", err)
		return nil
	}

	log.Printf("Parsed %d parameters from GetParameterValuesResponse", len(parsed.ParameterList))

	// Try to find device by IP address from request
	clientIP := strings.Split(r.RemoteAddr, ":")[0]

	// First, look for the device by IP in database
	var device *models.Device
	devices, _, _ := s.DB.GetDevices("online", "", 500, 0)
	for _, d := range devices {
		if d.IPAddress == clientIP {
			device = d
			break
		}
	}

	// Also check in parameters for SerialNumber
	if device == nil {
		var serialNumber string
		for _, p := range parsed.ParameterList {
			if strings.HasSuffix(p.Name, ".SerialNumber") || strings.HasSuffix(p.Name, ".X_HW_SerialNumber") {
				serialNumber = p.Value
				break
			}
		}
		if serialNumber != "" {
			device, _ = s.DB.GetDeviceBySerial(serialNumber)
		}
	}

	// Also try to find from session map
	if device == nil {
		if sessionData, ok := s.sessions.Load(clientIP); ok {
			session := sessionData.(*Session)
			device, _ = s.DB.GetDevice(session.DeviceID)
		}
	}

	if device != nil {
		// Use new parameter parser for better data extraction
		parser := NewDeviceParameterParser(device, device.Manufacturer, device.ModelName)

		// Parse all parameters
		for _, p := range parsed.ParameterList {
			parser.ParseParameter(p.Name, p.Value)
		}

		// Store each parameter
		storedCount := 0
		for _, p := range parsed.ParameterList {
			err := s.DB.SetDeviceParameter(device.ID, p.Name, p.Value, p.Type, true)
			if err != nil {
				log.Printf("Error storing parameter %s: %v", p.Name, err)
			} else {
				storedCount++
			}
		}

		// Update device with parsed data
		parsedDevice := parser.GetDeviceData()
		if parsedDevice != nil && device != nil {
			// Update device fields that were parsed
			if parsedDevice.RXPower != 0 {
				device.RXPower = parsedDevice.RXPower
			}
			if parsedDevice.TXPower != 0 {
				device.TXPower = parsedDevice.TXPower
			}
			if parsedDevice.OpticalTemperature != 0 {
				device.OpticalTemperature = parsedDevice.OpticalTemperature
			}
			if parsedDevice.OpticalVoltage != 0 {
				device.OpticalVoltage = parsedDevice.OpticalVoltage
			}
			if parsedDevice.OpticalCurrent != 0 {
				device.OpticalCurrent = parsedDevice.OpticalCurrent
			}
			if parsedDevice.Distance != 0 {
				device.Distance = parsedDevice.Distance
			}

			// Save updated device to database
			if err := s.DB.UpdateDevice(device); err != nil {
				log.Printf("Error updating device with parsed parameters: %v", err)
			} else {
				log.Printf("Updated device %s with parsed optical parameters: RX=%.2f dBm, TX=%.2f dBm, Temp=%.2f°C",
					device.SerialNumber, device.RXPower, device.TXPower, device.OpticalTemperature)
			}
		}

		log.Printf("Stored %d parameters for device %s (IP: %s)", storedCount, device.SerialNumber, clientIP)

		// Mark task as completed
		if envelope.Header != nil && strings.HasPrefix(envelope.Header.ID, "task-") {
			taskIDStr := strings.TrimPrefix(envelope.Header.ID, "task-")
			if taskID, err := strconv.ParseInt(taskIDStr, 10, 64); err == nil {
				now := time.Now()
				resJSON, _ := json.Marshal(map[string]interface{}{"count": storedCount})
				task := &models.DeviceTask{
					ID:          taskID,
					Status:      models.TaskCompleted,
					CompletedAt: &now,
					Result:      resJSON,
				}
				s.DB.UpdateTask(task)
			}
		}
	} else if len(parsed.ParameterList) > 0 {
		log.Printf("No device identified for IP %s, skipping parameter storage for %d params", clientIP, len(parsed.ParameterList))
	}

	return nil
}

func (s *Server) handleSetParameterValuesResponse(envelope *SOAPEnvelope, _ *http.Request) {
	log.Println("SetParameterValuesResponse received")
	if envelope.Header != nil && strings.HasPrefix(envelope.Header.ID, "task-") {
		taskIDStr := strings.TrimPrefix(envelope.Header.ID, "task-")
		if taskID, err := strconv.ParseInt(taskIDStr, 10, 64); err == nil {
			now := time.Now()
			task := &models.DeviceTask{
				ID:          taskID,
				Status:      models.TaskCompleted,
				CompletedAt: &now,
			}
			s.DB.UpdateTask(task)
		}
	}
}

func (s *Server) handleRebootResponse(envelope *SOAPEnvelope, _ *http.Request) {
	log.Println("RebootResponse received")
	if envelope.Header != nil && strings.HasPrefix(envelope.Header.ID, "task-") {
		taskIDStr := strings.TrimPrefix(envelope.Header.ID, "task-")
		if taskID, err := strconv.ParseInt(taskIDStr, 10, 64); err == nil {
			now := time.Now()
			task := &models.DeviceTask{
				ID:          taskID,
				Status:      models.TaskCompleted,
				CompletedAt: &now,
			}
			s.DB.UpdateTask(task)
		}
	}
}

func (s *Server) handleFactoryResetResponse(envelope *SOAPEnvelope, _ *http.Request) {
	log.Println("FactoryResetResponse received")
	if envelope.Header != nil && strings.HasPrefix(envelope.Header.ID, "task-") {
		taskIDStr := strings.TrimPrefix(envelope.Header.ID, "task-")
		if taskID, err := strconv.ParseInt(taskIDStr, 10, 64); err == nil {
			now := time.Now()
			task := &models.DeviceTask{
				ID:          taskID,
				Status:      models.TaskCompleted,
				CompletedAt: &now,
			}
			s.DB.UpdateTask(task)
		}
	}
}

// SendConnectionRequest sends a connection request to a CPE
func (s *Server) SendConnectionRequest(device *models.Device) error {
	if device.ConnectionRequest == "" {
		return fmt.Errorf("no connection request URL for device %s", device.SerialNumber)
	}

	// Make the connection request
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", device.ConnectionRequest, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("connection request failed with status: %d", resp.StatusCode)
	}

	log.Printf("Connection request sent to device %s", device.SerialNumber)
	return nil
}

// ============== SOAP Message Types ==============

// SOAPEnvelope represents a SOAP envelope
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"Envelope"`
	Header  *SOAPHeader `xml:"Header,omitempty"`
	Body    SOAPBody    `xml:"Body"`
}

// SOAPHeader represents a SOAP header
type SOAPHeader struct {
	ID     string `xml:"ID,omitempty"`
	NoMore int    `xml:"NoMoreRequests,omitempty"`
}

// SOAPBody represents a SOAP body
type SOAPBody struct {
	InnerXML []byte `xml:",innerxml"`
}

// Inform message structure
type Inform struct {
	DeviceId      DeviceIdStruct      `xml:"DeviceId"`
	Event         EventList           `xml:"Event"`
	MaxEnvelopes  int                 `xml:"MaxEnvelopes"`
	CurrentTime   string              `xml:"CurrentTime"`
	RetryCount    int                 `xml:"RetryCount"`
	ParameterList ParameterValuesList `xml:"ParameterList"`
}

type DeviceIdStruct struct {
	Manufacturer string `xml:"Manufacturer"`
	OUI          string `xml:"OUI"`
	ProductClass string `xml:"ProductClass"`
	SerialNumber string `xml:"SerialNumber"`
}

type EventList struct {
	EventStruct []EventStruct `xml:"EventStruct"`
}

type EventStruct struct {
	EventCode  string `xml:"EventCode"`
	CommandKey string `xml:"CommandKey"`
}

type ParameterValuesList struct {
	ParameterValueStruct []ParameterValueStruct `xml:"ParameterValueStruct"`
}

type ParameterValueStruct struct {
	Name  string `xml:"Name"`
	Value string `xml:"Value"`
}

// ============== Helper Functions ==============

func parseSOAPEnvelope(data []byte) (*SOAPEnvelope, error) {
	// Remove common namespace prefixes for easier parsing
	dataStr := string(data)
	dataStr = strings.ReplaceAll(dataStr, "soap:", "")
	dataStr = strings.ReplaceAll(dataStr, "soap-env:", "")
	dataStr = strings.ReplaceAll(dataStr, "SOAP-ENV:", "")
	dataStr = strings.ReplaceAll(dataStr, "cwmp:", "")

	var envelope SOAPEnvelope
	decoder := xml.NewDecoder(bytes.NewReader([]byte(dataStr)))
	decoder.Strict = false

	if err := decoder.Decode(&envelope); err != nil {
		return nil, err
	}

	return &envelope, nil
}

func parseInform(body []byte) (*Inform, error) {
	bodyStr := string(body)

	// More robust way to find Inform element regardless of namespaces
	// We look for anything that ends in :Inform or is just <Inform
	informStart := -1
	tags := []string{"<cwmp:Inform", "<Inform", "<v1:Inform", "<v2:Inform"}
	for _, tag := range tags {
		if idx := strings.Index(bodyStr, tag); idx != -1 {
			informStart = idx
			break
		}
	}

	if informStart == -1 {
		return nil, fmt.Errorf("Inform element not found in SOAP body")
	}

	informEndTag := ""
	endTags := []string{"</cwmp:Inform>", "</Inform>", "</v1:Inform>", "</v2:Inform>"}
	for _, tag := range endTags {
		if idx := strings.Index(bodyStr[informStart:], tag); idx != -1 {
			informEndTag = tag
			break
		}
	}

	if informEndTag == "" {
		return nil, fmt.Errorf("Inform end tag not found")
	}

	informEnd := strings.Index(bodyStr[informStart:], informEndTag) + len(informEndTag)
	informXML := bodyStr[informStart : informStart+informEnd]

	// Clean up internal namespaces for unmarshaling
	informXML = strings.ReplaceAll(informXML, "cwmp:", "")
	informXML = strings.ReplaceAll(informXML, "v1:", "")
	informXML = strings.ReplaceAll(informXML, "v2:", "")

	var inform Inform
	if err := xml.Unmarshal([]byte(informXML), &inform); err != nil {
		return nil, err
	}

	return &inform, nil
}

func createInformResponse(header *SOAPHeader) *SOAPEnvelope {
	response := `<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:InformResponse>
      <MaxEnvelopes>1</MaxEnvelopes>
    </cwmp:InformResponse>
  </soap:Body>
</soap:Envelope>`

	id := "1"
	if header != nil && header.ID != "" {
		id = header.ID
	}

	// Parse and return as envelope
	var envelope SOAPEnvelope
	xml.Unmarshal([]byte(fmt.Sprintf(response, id)), &envelope)

	return &SOAPEnvelope{
		Header: &SOAPHeader{ID: id},
		Body: SOAPBody{
			InnerXML: []byte(`<cwmp:InformResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><MaxEnvelopes>1</MaxEnvelopes></cwmp:InformResponse>`),
		},
	}
}

func createTransferCompleteResponse(header *SOAPHeader) *SOAPEnvelope {
	id := "1"
	if header != nil && header.ID != "" {
		id = header.ID
	}

	return &SOAPEnvelope{
		Header: &SOAPHeader{ID: id},
		Body: SOAPBody{
			InnerXML: []byte(`<cwmp:TransferCompleteResponse xmlns:cwmp="urn:dslforum-org:cwmp-1-0"></cwmp:TransferCompleteResponse>`),
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// decodeSerialNumber converts hex-encoded GPON serials to human readable text
func decodeSerialNumber(sn string) string {
	if len(sn) < 12 {
		return sn
	}

	// Check if it's hexadecimal
	isHex, _ := regexp.MatchString("^[0-9A-Fa-f]+$", sn)
	if !isHex {
		return sn
	}

	// Logic from User's Script:
	// Avoid decoding if it contains Nokia OUI "40ee15" or if it is suspected EPON
	if strings.Contains(strings.ToLower(sn), "40ee15") {
		return sn
	}

	// Attempt to decode first 8 hex chars (4 bytes)
	hexPart := sn[:8]
	decoded, err := hex.DecodeString(hexPart)
	if err != nil {
		return sn
	}

	// Check if decoded part is printable ASCII (ZTEG, HWTC, etc)
	for _, b := range decoded {
		if b < 32 || b > 126 {
			return sn // Keep raw if not printable
		}
	}

	return string(decoded) + sn[8:]
}

// bootstrapDevice implements the logic from the user's Provisioning script
func (s *Server) bootstrapDevice(device *models.Device) {
	// 1. Determine if we need to set Remote Access (ACL)
	// We do this based on Manufacturer and Uptime

	params := make(map[string]string)
	isFiberHome := strings.Contains(strings.ToUpper(device.Manufacturer), "FIBERHOME")
	isHuawei := strings.Contains(strings.ToUpper(device.Manufacturer), "HUAWEI")
	isZTE := strings.Contains(strings.ToUpper(device.Manufacturer), "ZTE")

	// FiberHome Remote Access Logic
	if isFiberHome {
		// If uptime is small, it might be a fresh boot
		if device.Uptime > 0 {
			params["InternetGatewayDevice.X_FH_ACL.Enable"] = "1"
			if device.Uptime < 220 {
				params["InternetGatewayDevice.X_FH_FireWall.REMOTEACCEnable"] = "0"
				params["InternetGatewayDevice.X_FH_Remoteweblogin.webloginenable"] = "0"
			} else {
				params["InternetGatewayDevice.X_FH_FireWall.REMOTEACCEnable"] = "1"
				params["InternetGatewayDevice.X_FH_Remoteweblogin.webloginenable"] = "1"
			}
			// General ACL Rules
			params["InternetGatewayDevice.X_FH_ACL.Rule.1.Enable"] = "1"
			params["InternetGatewayDevice.X_FH_ACL.Rule.1.Direction"] = "1"
			params["InternetGatewayDevice.X_FH_ACL.Rule.1.Protocol"] = "ALL"
		}
	}

	// Huawei Remote Access Logic
	if isHuawei {
		params["InternetGatewayDevice.X_HW_Security.AclServices.SSHWanEnable"] = "1"
		params["InternetGatewayDevice.X_HW_Security.AclServices.HTTPWanEnable"] = "1"
		params["InternetGatewayDevice.X_HW_Security.AclServices.TELNETWanEnable"] = "1"
		params["InternetGatewayDevice.X_HW_Security.X_HW_FirewallLevel"] = "Custom"
		params["InternetGatewayDevice.X_HW_Security.Dosfilter.IcmpEchoReplyEn"] = "0"
	}

	// ZTE Remote Access Logic
	if isZTE {
		params["InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.ServiceType"] = "HTTP"
		params["InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.Ingress"] = "WAN_ALL"
		params["InternetGatewayDevice.Firewall.X_ZTE-COM_ServiceControl.IPV4ServiceControl.1.Enable"] = "1"
	}

	// If we have ACL parameters to set, queue a task
	if len(params) > 0 {
		payload, _ := json.Marshal(params)
		task := &models.DeviceTask{
			DeviceID:   device.ID,
			Type:       models.TaskSetParameterValues,
			Status:     models.TaskPending,
			Parameters: payload,
		}
		s.DB.CreateTask(task)
		log.Printf("Auto-provisioning: Queued Remote Access task for %s", device.SerialNumber)
	}

	// 2. Schedule Parameter Refresh (GetParameterValues)
	// Use comprehensive parameter system for better ONU data collection

	// Get standard ONU parameters
	standardParams := GetStandardONUParameters()
	commonParams := GetONUCommonParameters()

	// Get vendor-specific parameters
	vendorResolver := NewVendorSpecificParameterResolver(device.Manufacturer, device.ModelName)
	vendorParams := vendorResolver.GetVendorSpecificParameters()

	// Combine all parameter paths
	allPaths := make([]string, 0)

	// Add standard parameters
	for _, category := range standardParams {
		for _, param := range category.Parameters {
			allPaths = append(allPaths, param.Path)
		}
	}

	// Add common parameters
	allPaths = append(allPaths, commonParams...)

	// Add vendor-specific parameters
	allPaths = append(allPaths, vendorParams...)

	// Add legacy paths for backward compatibility
	legacyPaths := []string{
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.",
		"InternetGatewayDevice.LANDevice.1.Hosts.",
		"InternetGatewayDevice.DeviceInfo.",
	}
	allPaths = append(allPaths, legacyPaths...)

	// Add vendor-specific paths from legacy logic
	if isHuawei {
		allPaths = append(allPaths, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.X_HW_VenderClassID")
	}

	payloadRefresh, _ := json.Marshal(allPaths)
	refreshTask := &models.DeviceTask{
		DeviceID:   device.ID,
		Type:       models.TaskGetParameterValues,
		Status:     models.TaskPending,
		Parameters: payloadRefresh,
	}
	s.DB.CreateTask(refreshTask)

	log.Printf("Auto-provisioning: Queued comprehensive parameter refresh for %s (%s %s) with %d parameters",
		device.SerialNumber, device.Manufacturer, device.ModelName, len(allPaths))
}
