package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agroconnect/service-order/models"
)

type MidtransService interface {
	CreateSnapTransaction(ctx context.Context, order *models.Order, items []models.OrderItem, customer models.MidtransCustomerDetails) (*models.MidtransSnapResponse, error)
	VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool
}

type midtransService struct {
	serverKey  string
	clientKey  string
	apiURL     string
	httpClient *http.Client
}

func NewMidtransService(serverKey, clientKey string, isProduction bool) MidtransService {
	apiURL := "https://app.sandbox.midtrans.com/snap/v1/transactions"
	if isProduction {
		apiURL = "https://app.midtrans.com/snap/v1/transactions"
	}

	return &midtransService{
		serverKey:  serverKey,
		clientKey:  clientKey,
		apiURL:     apiURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *midtransService) CreateSnapTransaction(ctx context.Context, order *models.Order, items []models.OrderItem, customer models.MidtransCustomerDetails) (*models.MidtransSnapResponse, error) {
	grossAmount := int64(order.TotalAmount)
	if grossAmount <= 0 {
		var calcGross int64
		for _, it := range items {
			calcGross += int64(it.Price * float64(it.Quantity))
		}
		grossAmount = calcGross
	}

	var midtransItems []models.MidtransItemDetail
	for _, it := range items {
		midtransItems = append(midtransItems, models.MidtransItemDetail{
			ID:       it.ProductID,
			Price:    int64(it.Price),
			Quantity: it.Quantity,
			Name:     it.ProductName,
		})
	}

	reqPayload := models.MidtransSnapRequest{
		TransactionDetails: models.MidtransTransactionDetails{
			OrderID:     order.OrderCode,
			GrossAmount: grossAmount,
		},
		CustomerDetails: customer,
		ItemDetails:     midtransItems,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snap payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize snap http request: %w", err)
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(s.serverKey+":"))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("midtrans snap request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read midtrans snap response: %w", err)
	}

	var snapResp models.MidtransSnapResponse
	if err := json.Unmarshal(respBytes, &snapResp); err != nil {
		return nil, fmt.Errorf("failed to parse midtrans response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("midtrans error HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	return &snapResp, nil
}

func (s *midtransService) VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	rawSignatureInput := orderID + statusCode + grossAmount + s.serverKey
	hasher := sha512.New()
	hasher.Write([]byte(rawSignatureInput))
	computedHash := hex.EncodeToString(hasher.Sum(nil))

	return strings.EqualFold(computedHash, signatureKey)
}
