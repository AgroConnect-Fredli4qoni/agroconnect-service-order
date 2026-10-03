package models

type MidtransTransactionDetails struct {
	OrderID     string  `json:"order_id"`
	GrossAmount int64   `json:"gross_amount"`
}

type MidtransCustomerDetails struct {
	FirstName string `json:"first_name,omitempty"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

type MidtransItemDetail struct {
	ID       string `json:"id"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
	Name     string `json:"name"`
}

type MidtransSnapRequest struct {
	TransactionDetails MidtransTransactionDetails `json:"transaction_details"`
	CustomerDetails    MidtransCustomerDetails    `json:"customer_details"`
	ItemDetails        []MidtransItemDetail       `json:"item_details,omitempty"`
}

type MidtransSnapResponse struct {
	Token         string `json:"token"`
	RedirectURL   string `json:"redirect_url"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

type MidtransNotificationPayload struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status,omitempty"`
	SettlementTime    string `json:"settlement_time,omitempty"`
	TransactionID     string `json:"transaction_id,omitempty"`
}
