package mailer

import (
	"fmt"
	"net/smtp"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Mailer handles email sending
type Mailer struct {
	config Config
}

// New creates a new Mailer
func New(config Config) *Mailer {
	return &Mailer{config: config}
}

// Send sends an email
func (m *Mailer) Send(to string, subject string, body string) error {
	// If no config, just log (mock mode)
	if m.config.Host == "" {
		fmt.Printf("[MOCK MAIL] To: %s | Subject: %s | Body length: %d\n", to, subject, len(body))
		return nil
	}

	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n", to, subject, body))

	return smtp.SendMail(addr, auth, m.config.From, []string{to}, msg)
}

// GenerateInvoiceHTML generates HTML for invoice email
func GenerateInvoiceHTML(customerName, invoiceNo, dueDate, totals string) string {
	return fmt.Sprintf(`
		<html>
		<body>
			<h2>Invoice Notification</h2>
			<p>Dear %s,</p>
			<p>A new invoice <strong>%s</strong> has been generated for your account.</p>
			<p><strong>Total Amount:</strong> %s</p>
			<p><strong>Due Date:</strong> %s</p>
			<p>Please make payment before the due date to avoid service interruption.</p>
			<br>
			<p>Thank you,<br>GO-ACS Team</p>
		</body>
		</html>
	`, customerName, invoiceNo, totals, dueDate)
}

// GeneratePaymentReceiptHTML generates HTML for payment receipt
func GeneratePaymentReceiptHTML(customerName, invoiceNo, amount, paidDate string) string {
	return fmt.Sprintf(`
		<html>
		<body>
			<h2>Payment Receipt</h2>
			<p>Dear %s,</p>
			<p>We have received your payment for invoice <strong>%s</strong>.</p>
			<p><strong>Amount Paid:</strong> %s</p>
			<p><strong>Date:</strong> %s</p>
			<p>Your transaction has been completed successfully.</p>
			<br>
			<p>Thank you,<br>GO-ACS Team</p>
		</body>
		</html>
	`, customerName, invoiceNo, amount, paidDate)
}
