package fcm

import (
	"context"
	"fmt"
	"log"

	"go-acs/internal/config"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Client handles FCM notifications
type Client struct {
	app *firebase.App
	cfg *config.Config
}

// New creates a new FCM client
func New(cfg *config.Config) *Client {
	if cfg.FirebaseCredentialsFile == "" {
		log.Println("⚠ FCM: FirebaseCredentialsFile not set, notifications disabled")
		return &Client{cfg: cfg}
	}

	opt := option.WithCredentialsFile(cfg.FirebaseCredentialsFile)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Printf("⚠ FCM: Failed to initialize Firebase app: %v", err)
		return &Client{cfg: cfg}
	}

	log.Println("✓ FCM: Firebase initialized successfully")
	return &Client{
		app: app,
		cfg: cfg,
	}
}

// Send sends a push notification to a specific token
func (c *Client) Send(token, title, body string) error {
	if c.app == nil {
		return nil // FCM not initialized
	}

	if token == "" {
		return fmt.Errorf("FCM: empty token")
	}

	ctx := context.Background()
	client, err := c.app.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("FCM: error getting messaging client: %v", err)
	}

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: token,
	}

	response, err := client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("FCM: error sending message: %v", err)
	}

	log.Printf("✓ FCM: Successfully sent message: %s", response)
	return nil
}
