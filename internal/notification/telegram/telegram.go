package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client represents a Telegram bot client
type Client struct {
	Token  string
	ChatID string
}

// New creates a new Telegram client
func New(token, chatID string) *Client {
	return &Client{
		Token:  token,
		ChatID: chatID,
	}
}

// Message represents a Telegram message payload
type Message struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendMessage sends a message to Telegram
func (c *Client) SendMessage(message string) error {
	if c.Token == "" || c.ChatID == "" {
		return fmt.Errorf("telegram token or chat_id not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.Token)

	payload := Message{
		ChatID:    c.ChatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendUpdateNotification sends a formatted update notification
func (c *Client) SendUpdateNotification(status, message, details string) error {
	hostname, _ := os.Hostname()
	now := time.Now().Format("2006-01-02 15:04:05")

	var emoji string
	switch status {
	case "start":
		emoji = "🚀"
	case "success":
		emoji = "✅"
	case "error":
		emoji = "❌"
	case "warning":
		emoji = "⚠️"
	case "info":
		emoji = "ℹ️"
	default:
		emoji = "📢"
	}

	text := fmt.Sprintf(
		"<b>%s GO-ACS Update Notification</b>\n\n"+
			"<b>Server:</b> %s\n"+
			"<b>Time:</b> %s\n"+
			"<b>Status:</b> %s\n\n"+
			"<b>Message:</b>\n%s",
		emoji,
		hostname,
		now,
		status,
		message,
	)

	if details != "" {
		text += fmt.Sprintf("\n\n<b>Details:</b>\n<code>%s</code>", details)
	}

	return c.SendMessage(text)
}

// SendUpdateStart sends notification when update starts
func (c *Client) SendUpdateStart(branch, currentCommit string) error {
	hostname, _ := os.Hostname()
	message := fmt.Sprintf(
		"Update process started on <b>%s</b>\n\n"+
			"Current Branch: <code>%s</code>\n"+
			"Current Commit: <code>%s</code>",
		hostname,
		branch,
		currentCommit,
	)
	return c.SendUpdateNotification("start", message, "")
}

// SendUpdateProgress sends progress notification
func (c *Client) SendUpdateProgress(step, output string) error {
	message := fmt.Sprintf("Step: <b>%s</b>", step)
	return c.SendUpdateNotification("info", message, output)
}

// SendUpdateSuccess sends success notification
func (c *Client) SendUpdateSuccess(newCommit, duration string) error {
	hostname, _ := os.Hostname()
	message := fmt.Sprintf(
		"Update completed successfully on <b>%s</b>\n\n"+
			"New Commit: <code>%s</code>\n"+
			"Duration: %s\n\n"+
			"Service restarted successfully!",
		hostname,
		newCommit,
		duration,
	)
	return c.SendUpdateNotification("success", message, "")
}

// SendUpdateError sends error notification
func (c *Client) SendUpdateError(step, errorMsg string) error {
	hostname, _ := os.Hostname()
	message := fmt.Sprintf(
		"Update failed on <b>%s</b>\n\n"+
			"Failed at: <b>%s</b>",
		hostname,
		step,
	)
	return c.SendUpdateNotification("error", message, errorMsg)
}

// SendRebuildNotification sends rebuild notification
func (c *Client) SendRebuildNotification(success bool, output string) error {
	var status, message string
	if success {
		status = "success"
		message = "Application rebuild completed successfully"
	} else {
		status = "error"
		message = "Application rebuild failed"
	}
	return c.SendUpdateNotification(status, message, output)
}

// SendServiceRestartNotification sends service restart notification
func (c *Client) SendServiceRestartNotification(success bool) error {
	hostname, _ := os.Hostname()
	var status, message string
	if success {
		status = "success"
		message = fmt.Sprintf("GO-ACS service restarted successfully on <b>%s</b>", hostname)
	} else {
		status = "error"
		message = fmt.Sprintf("Failed to restart GO-ACS service on <b>%s</b>", hostname)
	}
	return c.SendUpdateNotification(status, message, "")
}
