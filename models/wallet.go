package models

import "time"

type FarmerWallet struct {
	ID             int       `json:"id"`
	FarmerID       int       `json:"farmer_id"`
	Balance        float64   `json:"balance"`
	PendingBalance float64   `json:"pending_balance"`
	BankName       string    `json:"bank_name,omitempty"`
	AccountNumber  string    `json:"account_number,omitempty"`
	AccountHolder  string    `json:"account_holder,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WalletTransaction struct {
	ID          int       `json:"id"`
	WalletID    int       `json:"wallet_id"`
	OrderID     *int      `json:"order_id,omitempty"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type WithdrawalRequest struct {
	ID             int       `json:"id"`
	FarmerID       int       `json:"farmer_id"`
	Amount         float64   `json:"amount"`
	TargetType     string    `json:"target_type"`
	TargetProvider string    `json:"target_provider"`
	TargetAccount  string    `json:"target_account"`
	AccountHolder  string    `json:"account_holder"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateWithdrawalDTO struct {
	Amount         float64 `json:"amount"`
	TargetType     string  `json:"target_type"`
	TargetProvider string  `json:"target_provider"`
	TargetAccount  string  `json:"target_account"`
	AccountHolder  string  `json:"account_holder"`
}

type UpdateWalletAccountDTO struct {
	BankName      string `json:"bank_name"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
}

type PayoutAccount struct {
	ID            int       `json:"id"`
	FarmerID      int       `json:"farmer_id"`
	AccountType   string    `json:"account_type"`
	ProviderName  string    `json:"provider_name"`
	AccountNumber string    `json:"account_number"`
	AccountHolder string    `json:"account_holder"`
	IsPrimary     bool      `json:"is_primary"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreatePayoutAccountDTO struct {
	AccountType   string `json:"account_type"`
	ProviderName  string `json:"provider_name"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	IsPrimary     bool   `json:"is_primary"`
}

type WalletOverviewDTO struct {
	Wallet         FarmerWallet        `json:"wallet"`
	Transactions   []WalletTransaction `json:"transactions"`
	Withdrawals    []WithdrawalRequest `json:"withdrawals"`
	PayoutAccounts []PayoutAccount     `json:"payout_accounts"`
}

