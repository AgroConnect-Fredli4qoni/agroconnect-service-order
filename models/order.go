package models

import "time"

type OrderItem struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	Subtotal    float64 `json:"subtotal"`
}

type Order struct {
	ID              int         `json:"id"`
	OrderCode       string      `json:"order_code"`
	UserID          int         `json:"user_id"`
	TotalAmount     float64     `json:"total_amount"`
	Status          string      `json:"status"`
	ShippingAddress string      `json:"shipping_address"`
	CreatedAt       time.Time   `json:"created_at"`
	Items           []OrderItem `json:"items,omitempty"`
}

type CreateOrderItemDTO struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

type CreateOrderDTO struct {
	UserID          int                  `json:"user_id"`
	ShippingAddress string               `json:"shipping_address"`
	Items           []CreateOrderItemDTO `json:"items"`
}
