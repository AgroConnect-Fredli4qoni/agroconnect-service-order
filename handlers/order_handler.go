package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/agroconnect/service-order/models"
	"github.com/agroconnect/service-order/repository"
	"github.com/gorilla/mux"
)

type OrderHandler struct {
	orderRepo         repository.OrderRepository
	catalogServiceURL string
	httpClient        *http.Client
}

func NewOrderHandler(orderRepo repository.OrderRepository, catalogServiceURL string) *OrderHandler {
	return &OrderHandler{
		orderRepo:         orderRepo,
		catalogServiceURL: catalogServiceURL,
		httpClient:        &http.Client{Timeout: 5 * time.Second},
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var dto models.CreateOrderDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload format"})
		return
	}

	if dto.UserID <= 0 || len(dto.Items) == 0 || dto.ShippingAddress == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Validation failed: UserID, ShippingAddress, and Items are required"})
		return
	}

	var totalAmount float64
	var items []models.OrderItem

	for _, itemDTO := range dto.Items {
		if itemDTO.ProductID == "" || itemDTO.Quantity <= 0 || itemDTO.Price <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid item specifications in order"})
			return
		}

		subtotal := itemDTO.Price * float64(itemDTO.Quantity)
		totalAmount += subtotal

		items = append(items, models.OrderItem{
			ProductID:   itemDTO.ProductID,
			ProductName: itemDTO.ProductName,
			Price:       itemDTO.Price,
			Quantity:    itemDTO.Quantity,
			Subtotal:    subtotal,
		})
	}

	rNumber := rand.Intn(9000) + 1000
	orderCode := fmt.Sprintf("ORD-%d-%04d", time.Now().Year(), rNumber)

	order := models.Order{
		OrderCode:       orderCode,
		UserID:          dto.UserID,
		TotalAmount:     totalAmount,
		Status:          "PENDING",
		ShippingAddress: dto.ShippingAddress,
		CreatedAt:       time.Now(),
	}

	stockDeductor := func(ctx context.Context, productID string, qty int) error {
		url := fmt.Sprintf("%s/api/products/%s/stock", h.catalogServiceURL, productID)
		payload, err := json.Marshal(map[string]int{"quantity": qty})
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := h.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errors.New("catalog service rejected stock update: status " + resp.Status)
		}

		return nil
	}

	if err := h.orderRepo.CreateOrderACID(r.Context(), &order, items, stockDeductor); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Transaction failed: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetOrdersByUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		userIDStr = "3"
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid user_id parameter"})
		return
	}

	orders, err := h.orderRepo.FindByUser(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
}

func (h *OrderHandler) GetOrderByCode(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	order, err := h.orderRepo.FindByCode(r.Context(), code)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(order)
}
