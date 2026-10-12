package models

import "time"

type PlatformAdminStatsDTO struct {
	TotalGMV          float64          `json:"total_gmv"`
	TotalOrders       int              `json:"total_orders"`
	TotalFarmers      int              `json:"total_farmers"`
	TotalBuyers       int              `json:"total_buyers"`
	TotalItemsSold    int              `json:"total_items_sold"`
	AverageOrderValue float64          `json:"average_order_value"`
	StatusBreakdown   []StatusCountDTO `json:"status_breakdown"`
	TopProducts       []TopProductDTO  `json:"top_products"`
	RecentOrders      []Order          `json:"recent_orders"`
}

type AdminUserDTO struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	OrderCount int      `json:"order_count"`
	TotalVolume float64 `json:"total_volume"`
}

type AdminWithdrawalItemDTO struct {
	ID             int       `json:"id"`
	FarmerID       int       `json:"farmer_id"`
	FarmerName     string    `json:"farmer_name"`
	FarmerEmail    string    `json:"farmer_email"`
	Amount         float64   `json:"amount"`
	TargetType     string    `json:"target_type"`
	TargetProvider string    `json:"target_provider"`
	TargetAccount  string    `json:"target_account"`
	AccountHolder  string    `json:"account_holder"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type UpdateWithdrawalStatusDTO struct {
	Status string `json:"status"`
}
