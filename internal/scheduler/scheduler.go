package scheduler

import (
	"fmt"
	"go-acs/internal/handlers"
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
