package scheduler

import (
	"fmt"
	"go-acs/internal/handlers"
	"go-acs/internal/models"
	"time"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	handler *handlers.Handler
}

// New creates a new Scheduler
func New(h *handlers.Handler) *Scheduler {
	return &Scheduler{handler: h}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	// Daily Tasks (e.g. Invoices)
	ticker := time.NewTicker(12 * time.Hour)
	go func() {
		for range ticker.C {
			s.runTasks()
		}
	}()

	// Bandwidth Monitoring (Every 5 minutes)
	monitorTicker := time.NewTicker(5 * time.Minute)
	go func() {
		for range monitorTicker.C {
			s.runBandwidthMonitor()
		}
	}()

	// Task Worker (Process pending tasks every 10 seconds)
	taskTicker := time.NewTicker(10 * time.Second)
	go func() {
		for range taskTicker.C {
			s.processPendingTasks()
		}
	}()
}

func (s *Scheduler) runTasks() {
	now := time.Now()

	// 1. Auto Invoice Generation (Run on 1st day of month)
	if now.Day() == 1 {
		fmt.Println("[SCHEDULER] Running monthly invoice generation...")
		count, err := s.handler.GenerateInvoicesInternal()
		if err != nil {
			fmt.Printf("[SCHEDULER] Error generating invoices: %v\n", err)
		} else {
			fmt.Printf("[SCHEDULER] Generated %d invoices\n", count)
		}
	}
}

func (s *Scheduler) runBandwidthMonitor() {
	if s.handler.Mikrotik == nil {
		return
	}

	customers, _, err := s.handler.DB.GetCustomers("active", "", 1000, 0)
	if err != nil {
		fmt.Printf("[MONITOR] Error fetching customers: %v\n", err)
		return
	}

	for _, cust := range customers {
		// Try multiple naming conventions for queue.
		// Adjust this based on your MikroTik setup.
		// Usually <pppoe-username> is dynamic queue name.
		queueName := cust.Username
		stats, err := s.handler.Mikrotik.GetQueueStats("<pppoe-" + queueName + ">")
		if err != nil {
			// Try plain username
			stats, err = s.handler.Mikrotik.GetQueueStats(queueName)
		}

		if err == nil && stats != nil {
			// Get devices associated with customer
			devices, err := s.handler.DB.GetDevicesByCustomer(cust.ID)
			if err == nil && len(devices) > 0 {
				// Record stats to the primary device
				// BytesSent (Upload) and BytesReceived (Download) from User perspective
				// which matches MikroTik simple queue target-upload/target-download usually
				s.handler.DB.RecordBandwidthUsage(devices[0].ID, stats.BytesSent, stats.BytesReceived)
			}
		}
	}
}

func (s *Scheduler) processPendingTasks() {
	// Get pending tasks
	tasks, err := s.handler.DB.GetPendingTasks(0) // 0 = all devices
	if err != nil {
		fmt.Printf("[TASK WORKER] Error fetching pending tasks: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	fmt.Printf("[TASK WORKER] Processing %d pending tasks...\n", len(tasks))

	for _, task := range tasks {
		fmt.Printf("[TASK WORKER] Processing task %d (type: %s, device: %d)\n", task.ID, task.Type, task.DeviceID)

		// Update task status to processing
		s.handler.DB.UpdateTaskStatus(task.ID, models.TaskRunning, nil, "")

		// Process task based on type
		var err error
		switch task.Type {
		case "getParameterValues":
			err = s.processGetParameterValues(task)
		case "setParameterValues":
			err = s.processSetParameterValues(task)
		case "refresh":
			err = s.processRefresh(task)
		case "reboot":
			err = s.processReboot(task)
		case "factoryReset":
			err = s.processFactoryReset(task)
		default:
			fmt.Printf("[TASK WORKER] Unknown task type: %s\n", task.Type)
			err = fmt.Errorf("unknown task type")
		}

		// Update task status
		if err != nil {
			errMsg := err.Error()
			s.handler.DB.UpdateTaskStatus(task.ID, models.TaskFailed, nil, errMsg)
			fmt.Printf("[TASK WORKER] Task %d failed: %v\n", task.ID, err)
		} else {
			s.handler.DB.UpdateTaskStatus(task.ID, models.TaskCompleted, nil, "")
			fmt.Printf("[TASK WORKER] Task %d completed\n", task.ID)
		}
	}
}

func (s *Scheduler) processGetParameterValues(task *models.DeviceTask) error {
	// Get device
	device, err := s.handler.DB.GetDevice(task.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}

	// Get WiFi parameters - this will be sent when device next connects
	// For now, mark as completed since we can't force device to send parameters
	fmt.Printf("[TASK WORKER] GetParameterValues for device %s (%s) - will be sent on next Inform\n", device.SerialNumber, device.Manufacturer)
	return nil
}

func (s *Scheduler) processSetParameterValues(task *models.DeviceTask) error {
	// Get device
	device, err := s.handler.DB.GetDevice(task.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}

	fmt.Printf("[TASK WORKER] SetParameterValues for device %s (%s)\n", device.SerialNumber, device.Manufacturer)
	// TODO: Implement parameter value setting
	return nil
}

func (s *Scheduler) processRefresh(task *models.DeviceTask) error {
	// Get device
	device, err := s.handler.DB.GetDevice(task.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}

	fmt.Printf("[TASK WORKER] Refresh for device %s (%s)\n", device.SerialNumber, device.Manufacturer)
	// TODO: Implement refresh - trigger GetParameterValues for WiFi
	return nil
}

func (s *Scheduler) processReboot(task *models.DeviceTask) error {
	// Get device
	device, err := s.handler.DB.GetDevice(task.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}

	fmt.Printf("[TASK WORKER] Reboot for device %s (%s)\n", device.SerialNumber, device.Manufacturer)
	// TODO: Implement reboot
	return nil
}

func (s *Scheduler) processFactoryReset(task *models.DeviceTask) error {
	// Get device
	device, err := s.handler.DB.GetDevice(task.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %v", err)
	}

	fmt.Printf("[TASK WORKER] Factory reset for device %s (%s)\n", device.SerialNumber, device.Manufacturer)
	// TODO: Implement factory reset
	return nil
}
