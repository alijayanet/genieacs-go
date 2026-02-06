package whatsapp

import (
	"fmt"
	"go-acs/internal/config"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles WhatsApp notifications
type Client struct {
	cfg *config.Config
}

// New creates a new WhatsApp client
func New(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

// Send sends a WhatsApp message
func (c *Client) Send(phone, message string) error {
	if c.cfg.WAApiKey == "" {
		fmt.Printf("[MOCK WA] To: %s | Message: %s\n", phone, message)
		return nil
	}

	// Normalize phone number (API usually needs 628xxx or 08xxx depending on provider)
	// For Fonnte/Generic usually 08 or 62 is handled, but let's keep as is or ensure numeric
	// Assuming generic webhook style

	data := url.Values{}
	data.Set("target", phone)
	data.Set("message", message)

	// Create request
	req, err := http.NewRequest("POST", c.cfg.WAProviderURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.cfg.WAApiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("whatsapp API error: %d", resp.StatusCode)
	}

	// Optional: Check body response for provider specific errors
	// var result map[string]interface{}
	// json.NewDecoder(resp.Body).Decode(&result)

	return nil
}

// Templates for common messages

func GenerateInvoiceMessage(customerName, invoiceNo, dueDate, amount string) string {
	return fmt.Sprintf("*Tagihan Baru - GO-ACS*\n\nHalo %s,\nTagihan baru (#%s) telah terbit.\n\nTotal: %s\nJatuh Tempo: %s\n\nMohon segera lakukan pembayaran untuk menghindari isolir layanan.\nTerima kasih.", customerName, invoiceNo, amount, dueDate)
}

func GeneratePaymentReceiptMessage(customerName, invoiceNo, paymentDate, amount string) string {
	return fmt.Sprintf("*Pembayaran Diterima - GO-ACS*\n\nHalo %s,\nPembayaran tagihan #%s sebesar %s telah kami terima pada %s.\n\nLayanan Anda aktif kembali/diperpanjang.\nTerima kasih.", customerName, invoiceNo, amount, paymentDate)
}

func GenerateSuspensionMessage(customerName string) string {
	return fmt.Sprintf("*Layanan Diisolir - GO-ACS*\n\nHalo %s,\nMohon maaf, layanan internet Anda diisolir sementara karena keterlambatan pembayaran.\n\nSilahkan lakukan pembayaran untuk mengaktifkan kembali layanan otomatis.\nTerima kasih.", customerName)
}
