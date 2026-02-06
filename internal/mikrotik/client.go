package mikrotik

import (
	"fmt"
	"go-acs/internal/config"

	"strconv"
	"strings"

	"github.com/go-routeros/routeros"
)

// Client handles MikroTik API connections
type Client struct {
	cfg *config.Config
}

// New creates a new MikroTik client
func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

// connect establishes a connection to the router
func (c *Client) connect() (*routeros.Client, error) {
	if c.cfg.MikrotikHost == "" {
		return nil, fmt.Errorf("MikroTik host not configured")
	}

	address := fmt.Sprintf("%s:%d", c.cfg.MikrotikHost, c.cfg.MikrotikPort)
	client, err := routeros.Dial(address, c.cfg.MikrotikUser, c.cfg.MikrotikPass)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// CreatePPPProfile creates or updates a PPP profile
func (c *Client) SyncPPPProfile(name, rateLimit string) error {
	client, err := c.connect()
	if err != nil {
		return err // In real app, maybe just log error if MT is offline
	}
	defer client.Close()

	// Check if profile exists
	res, err := client.Run("/ppp/profile/print", "?name="+name)
	if err != nil {
		return err
	}

	if len(res.Re) > 0 {
		// Update existing
		id := res.Re[0].Map["_id"]
		_, err = client.Run("/ppp/profile/set", "=.id="+id, "=rate-limit="+rateLimit)
	} else {
		// Create new
		_, err = client.Run("/ppp/profile/add", "=name="+name, "=rate-limit="+rateLimit)
	}

	return err
}

// GetPPPProfiles retrieves all PPP profiles
func (c *Client) GetPPPProfiles() ([]map[string]string, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	res, err := client.Run("/ppp/profile/print")
	if err != nil {
		return nil, err
	}

	profiles := make([]map[string]string, 0)
	for _, re := range res.Re {
		profiles = append(profiles, re.Map)
	}
	return profiles, nil
}

// QueueStats holds simple queue statistics
type QueueStats struct {
	BytesSent     int64 // Upload (Target Upload)
	BytesReceived int64 // Download (Target Download)
}

// GetQueueStats retrieves stats for a simple queue by name
func (c *Client) GetQueueStats(name string) (*QueueStats, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Check pure name first (static queue)
	res, err := client.Run("/queue/simple/print", "?name="+name)
	if err != nil {
		return nil, err
	}

	// If empty, try dynamic name convention if needed (usually just name matches ppp secret)
	if len(res.Re) == 0 {
		return nil, fmt.Errorf("queue not found")
	}

	// bytes string format: "upload/download" (e.g. "48520/1905322")
	bytesStr := res.Re[0].Map["bytes"]

	parts := strings.Split(bytesStr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bytes format")
	}

	// MikroTik "bytes" is "upload-bytes/download-bytes" from router perspective
	// Router Upload = User Download? No.
	// Queue Target Upload = packet from target to dst (User Upload)
	// Queue Target Download = packet from dst to target (User Download)
	// Usually: first is upload, second is download relative to target

	upload, _ := strconv.ParseInt(parts[0], 10, 64)
	download, _ := strconv.ParseInt(parts[1], 10, 64)

	return &QueueStats{
		BytesSent:     upload,
		BytesReceived: download,
	}, nil
}

// GetSystemResource retrieves router information (uptime, version, etc)
func (c *Client) GetSystemResource() (map[string]string, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	res, err := client.Run("/system/resource/print")
	if err != nil {
		return nil, err
	}

	if len(res.Re) > 0 {
		return res.Re[0].Map, nil
	}
	return nil, fmt.Errorf("could not get system resources")
}

// SetPPPProfile changes the PPP profile for a specific user
func (c *Client) SetPPPProfile(username, profile string) error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	// Find the PPP secret for the user
	res, err := client.Run("/ppp/secret/print", "?name="+username)
	if err != nil {
		return err
	}

	if len(res.Re) == 0 {
		return fmt.Errorf("PPP secret not found for user: %s", username)
	}

	// Get the ID of the PPP secret
	id := res.Re[0].Map["_id"]

	// Change the profile for the user
	_, err = client.Run("/ppp/secret/set", "=.id="+id, "=profile="+profile)
	return err
}

// GetPPPUsers retrieves all PPP users
func (c *Client) GetPPPUsers() ([]map[string]string, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	res, err := client.Run("/ppp/secret/print")
	if err != nil {
		return nil, err
	}

	users := make([]map[string]string, 0)
	for _, re := range res.Re {
		users = append(users, re.Map)
	}
	return users, nil
}

// CreateIsolirProfile creates an isolir PPP profile with limited bandwidth
func (c *Client) CreateIsolirProfile(name, rateLimit string) error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	// Check if profile exists
	res, err := client.Run("/ppp/profile/print", "?name="+name)
	if err != nil {
		return err
	}

	if len(res.Re) > 0 {
		// Update existing
		id := res.Re[0].Map["_id"]
		_, err = client.Run("/ppp/profile/set", "=.id="+id, "=rate-limit="+rateLimit)
	} else {
		// Create new isolir profile
		_, err = client.Run("/ppp/profile/add", "=name="+name, "=rate-limit="+rateLimit, "=local-address=192.168.100.1", "=remote-address=192.168.100.0/24")
	}

	return err
}

// DisconnectPPPUser disconnects an active PPP session for a specific user
func (c *Client) DisconnectPPPUser(username string) error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	// Find active PPP connections for the user
	res, err := client.Run("/ppp/active/print", "?name="+username)
	if err != nil {
		return err
	}

	// Disconnect each active session for the user
	for _, re := range res.Re {
		id := re.Map["_id"]
		_, err = client.Run("/ppp/active/remove", "=.id="+id)
		if err != nil {
			// Log error but continue with other sessions
			fmt.Printf("Failed to disconnect PPP session %s for user %s: %v\n", id, username, err)
		}
	}

	return nil
}

// DisconnectAllPPPUsers disconnects all active PPP sessions
func (c *Client) DisconnectAllPPPUsers() error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	// Find all active PPP connections
	res, err := client.Run("/ppp/active/print")
	if err != nil {
		return err
	}

	// Disconnect each active session
	for _, re := range res.Re {
		id := re.Map["_id"]
		_, err = client.Run("/ppp/active/remove", "=.id="+id)
		if err != nil {
			// Log error but continue with other sessions
			fmt.Printf("Failed to disconnect PPP session %s: %v\n", id, err)
		}
	}

	return nil
}
