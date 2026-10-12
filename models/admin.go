package models

import "time"

type PlatformAdminStatsDTO struct {
	TotalGMV                 float64          `json:"total_gmv"`
	TotalOrders              int              `json:"total_orders"`
	TotalFarmers             int              `json:"total_farmers"`
	TotalBuyers              int              `json:"total_buyers"`
	TotalItemsSold           int              `json:"total_items_sold"`
	AverageOrderValue        float64          `json:"average_order_value"`
	PendingOrders            int              `json:"pending_orders"`
	PaidOrders               int              `json:"paid_orders"`
	ShippedOrders            int              `json:"shipped_orders"`
	CompletedOrders          int              `json:"completed_orders"`
	CancelledOrders          int              `json:"cancelled_orders"`
	PendingWithdrawalsCount  int              `json:"pending_withdrawals_count"`
	PendingWithdrawalsAmount float64          `json:"pending_withdrawals_amount"`
	StatusBreakdown          []StatusCountDTO `json:"status_breakdown"`
	TopProducts              []TopProductDTO  `json:"top_products"`
	RecentOrders             []Order          `json:"recent_orders"`
}

type AdminUserDTO struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	OrderCount  int       `json:"order_count"`
	TotalOrders int       `json:"total_orders"`
	TotalVolume float64   `json:"total_volume"`
	TotalSpent  float64   `json:"total_spent"`
}

type AdminWithdrawalItemDTO struct {
	ID             int       `json:"id"`
	FarmerID       int       `json:"farmer_id"`
	FarmerName     string    `json:"farmer_name"`
	FarmerEmail    string    `json:"farmer_email"`
	Amount         float64   `json:"amount"`
	TargetType     string    `json:"target_type"`
	TargetProvider string    `json:"target_provider"`
	BankName       string    `json:"bank_name"`
	TargetAccount  string    `json:"target_account"`
	AccountNumber  string    `json:"account_number"`
	AccountHolder  string    `json:"account_holder"`
	Status         string    `json:"status"`
	AdminNotes     string    `json:"admin_notes"`
	CreatedAt      time.Time `json:"created_at"`
}

type UpdateWithdrawalStatusDTO struct {
	Status     string `json:"status"`
	AdminNotes string `json:"admin_notes"`
}
