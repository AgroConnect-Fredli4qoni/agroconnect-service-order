package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/agroconnect/service-order/models"
	"github.com/agroconnect/service-order/repository"
	"github.com/agroconnect/service-order/services"
	"github.com/gorilla/mux"
)

type PaymentHandler struct {
	orderRepo       repository.OrderRepository
	midtransService services.MidtransService
}

func NewPaymentHandler(orderRepo repository.OrderRepository, midtransService services.MidtransService) *PaymentHandler {
	return &PaymentHandler{
		orderRepo:       orderRepo,
		midtransService: midtransService,
	}
}

func (h *PaymentHandler) HandleMidtransWebhook(w http.ResponseWriter, r *http.Request) {
	var payload models.MidtransNotificationPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid notification payload"})
		return
	}

	if payload.OrderID == "" || payload.StatusCode == "" || payload.GrossAmount == "" || payload.SignatureKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing required signature fields"})
		return
	}

	if h.midtransService != nil {
		isValid := h.midtransService.VerifySignature(payload.OrderID, payload.StatusCode, payload.GrossAmount, payload.SignatureKey)
		if !isValid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid Midtrans SHA512 signature"})
			return
		}
	}

	err := h.orderRepo.UpdatePaymentFromWebhook(r.Context(), payload.OrderID, payload.PaymentType, payload.TransactionStatus, payload.TransactionStatus)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "OK",
		"message": "Payment notification processed successfully",
	})
}

func (h *PaymentHandler) GetOrderSnapToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderCode := vars["code"]
	if orderCode == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Order code parameter is required"})
		return
	}

	order, err := h.orderRepo.FindByCode(r.Context(), orderCode)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Order not found"})
		return
	}

	if order.SnapToken != "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"snap_token":        order.SnapToken,
			"snap_redirect_url": order.SnapRedirectURL,
		})
		return
	}

	if h.midtransService == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Midtrans service is not initialized"})
		return
	}

	customer := models.MidtransCustomerDetails{
		FirstName: order.CustomerName,
		Email:     order.CustomerEmail,
	}
	if customer.FirstName == "" {
		customer.FirstName = fmt.Sprintf("Pembeli #%d", order.UserID)
	}

	snapResp, err := h.midtransService.CreateSnapTransaction(r.Context(), order, order.Items, customer)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = h.orderRepo.SaveSnapToken(r.Context(), order.OrderCode, snapResp.Token, snapResp.RedirectURL)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"snap_token":        snapResp.Token,
		"snap_redirect_url": snapResp.RedirectURL,
	})
}
