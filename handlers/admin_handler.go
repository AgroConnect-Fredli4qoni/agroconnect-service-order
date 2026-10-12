package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/agroconnect/service-order/models"
	"github.com/agroconnect/service-order/repository"

	"github.com/gorilla/mux"
)

type AdminHandler struct {
	adminRepo repository.AdminRepository
}

func NewAdminHandler(adminRepo repository.AdminRepository) *AdminHandler {
	return &AdminHandler{adminRepo: adminRepo}
}

func (h *AdminHandler) checkAdminAccess(r *http.Request) bool {
	role := r.Header.Get("X-User-Role")
	return role == "admin"
}

func (h *AdminHandler) GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAccess(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: Administrator role required"})
		return
	}

	stats, err := h.adminRepo.GetPlatformAdminStats(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) GetAdminUsers(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAccess(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: Administrator role required"})
		return
	}

	roleFilter := r.URL.Query().Get("role")
	search := r.URL.Query().Get("search")

	users, err := h.adminRepo.FindAllUsers(r.Context(), roleFilter, search)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *AdminHandler) GetAdminOrders(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAccess(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: Administrator role required"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	orders, err := h.adminRepo.FindAllOrdersAdmin(r.Context(), statusFilter, search)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (h *AdminHandler) GetAdminWithdrawals(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAccess(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: Administrator role required"})
		return
	}

	statusFilter := r.URL.Query().Get("status")
	withdrawals, err := h.adminRepo.FindAllWithdrawalsAdmin(r.Context(), statusFilter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
}

func (h *AdminHandler) UpdateWithdrawalStatus(w http.ResponseWriter, r *http.Request) {
	if !h.checkAdminAccess(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: Administrator role required"})
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid withdrawal ID format"})
		return
	}

	var dto models.UpdateWithdrawalStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil || dto.Status == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Valid status is required"})
		return
	}

	if err := h.adminRepo.UpdateWithdrawalStatus(r.Context(), id, dto.Status); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Withdrawal status updated successfully"})
}
