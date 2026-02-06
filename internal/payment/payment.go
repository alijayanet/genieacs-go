package payment

import (
	"net/http"
)

// TransactionResponse holds common payment response data
type TransactionResponse struct {
	ReferenceID string // ID from gateway
	CheckoutURL string // URL to redirect user
	Amount      int64
	Status      string
	ExpiryDate  int64
}

// CallbackData holds standardized callback data
type CallbackData struct {
	InvoiceID     string
	Status        string // PAID, EXPIRED, FAILED
	Amount        int64
	PaidAt        int64
	PaymentMethod string
	ReferenceID   string
}

// PaymentChannel represents available payment method
type PaymentChannel struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Type string `json:"type"` // e.g. "VA", "EWALLET", "RETAIL"
	Logo string `json:"logo_url"`
}

// Gateway defines interface for payment providers
type Gateway interface {
	CreateTransaction(req TransactionRequest) (*TransactionResponse, error)
	GetChannels() ([]PaymentChannel, error)
	HandleCallback(r *http.Request) (*CallbackData, error)
}

// TransactionRequest holds data for creating transaction
type TransactionRequest struct {
	InvoiceID   string
	Amount      int64
	Customer    Customer
	Items       []Item
	Description string
	ReturnURL   string
}

type Customer struct {
	Name  string
	Email string
	Phone string
}

type Item struct {
	Name     string
	Price    int64
	Quantity int
}
