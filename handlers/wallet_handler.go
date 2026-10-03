package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/agroconnect/service-order/models"
	"github.com/agroconnect/service-order/repository"
)

type WalletHandler struct {
	walletRepo repository.WalletRepository
}

func NewWalletHandler(walletRepo repository.WalletRepository) *WalletHandler {
	return &WalletHandler{walletRepo: walletRepo}
}

func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	farmerID := 0
	if headerID := r.Header.Get("X-User-ID"); headerID != "" {
		if id, err := strconv.Atoi(headerID); err == nil {
			farmerID = id
		}
	}
	if farmerID == 0 {
		if queryID := r.URL.Query().Get("farmer_id"); queryID != "" {
			if id, err := strconv.Atoi(queryID); err == nil {
				farmerID = id
			}
		}
	}

	if farmerID == 0 {
		farmerID = 2
	}

	overview, err := h.walletRepo.GetWalletOverview(r.Context(), farmerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(overview)
}

func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	farmerID := 0
	if headerID := r.Header.Get("X-User-ID"); headerID != "" {
		if id, err := strconv.Atoi(headerID); err == nil {
			farmerID = id
		}
	}
	if farmerID == 0 {
		if queryID := r.URL.Query().Get("farmer_id"); queryID != "" {
			if id, err := strconv.Atoi(queryID); err == nil {
				farmerID = id
			}
		}
	}

	if farmerID == 0 {
		farmerID = 2
	}

	var dto models.CreateWithdrawalDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Format data penarikan tidak valid"})
		return
	}

	if dto.Amount < 10000 || dto.TargetAccount == "" || dto.AccountHolder == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Nominal minimal Rp 10.000, serta nomor rekening/e-wallet dan nama pemilik wajib diisi"})
		return
	}

	req, err := h.walletRepo.CreateWithdrawalRequest(r.Context(), farmerID, dto)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

func (h *WalletHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	farmerID := 0
	if headerID := r.Header.Get("X-User-ID"); headerID != "" {
		if id, err := strconv.Atoi(headerID); err == nil {
			farmerID = id
		}
	}
	if farmerID == 0 {
		farmerID = 2
	}

	var dto models.UpdateWalletAccountDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Format data rekening tidak valid"})
		return
	}

	if err := h.walletRepo.UpdateWalletAccount(r.Context(), farmerID, dto); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Rekening default penarikan berhasil diperbarui"})
}
