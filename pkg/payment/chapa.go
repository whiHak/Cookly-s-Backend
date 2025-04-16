package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
)

const (
	chapaBaseURL = "https://api.chapa.co/v1"
)

type ChapaService struct {
	secretKey string
}

type InitializePaymentRequest struct {
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	TxRef       string `json:"tx_ref"`
	CallbackURL string `json:"callback_url"`
	ReturnURL   string `json:"return_url"`
}

type InitializePaymentResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
}

type VerifyPaymentResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
	Data    struct {
		Amount      float64 `json:"amount"`
		Currency    string  `json:"currency"`
		Status      string  `json:"status"`
		Reference   string  `json:"reference"`
		TxRef       string  `json:"tx_ref"`
		PaymentType string  `json:"payment_type"`
	} `json:"data"`
}

func NewChapaService() *ChapaService {
	secretKey := os.Getenv("CHAPA_SECRET_KEY")
	if secretKey == "" {
		panic("CHAPA_SECRET_KEY environment variable is not set")
	}

	return &ChapaService{
		secretKey: secretKey,
	}
}

func (s *ChapaService) InitializePayment(amount float64, email, firstName, lastName string) (*InitializePaymentResponse, error) {
	// Generate unique transaction reference
	txRef := fmt.Sprintf("tx-ref-%s", uuid.New().String())

	// Prepare request payload
	payload := InitializePaymentRequest{
		Amount:      fmt.Sprintf("%.2f", amount),
		Currency:    "ETB",
		Email:       email,
		FirstName:   firstName,
		LastName:    lastName,
		TxRef:       txRef,
		CallbackURL: os.Getenv("CHAPA_CALLBACK_URL"),
		ReturnURL:   os.Getenv("CHAPA_RETURN_URL"),
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", chapaBaseURL+"/transaction/initialize", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create payment request: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+s.secretKey)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send payment request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response InitializePaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode payment response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("payment initialization failed: %s", response.Message)
	}

	return &response, nil
}

func (s *ChapaService) VerifyPayment(txRef string) (*VerifyPaymentResponse, error) {
	// Create HTTP request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/transaction/verify/%s", chapaBaseURL, txRef), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verification request: %v", err)
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+s.secretKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send verification request: %v", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response VerifyPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode verification response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("payment verification failed: %s", response.Message)
	}

	return &response, nil
}

func (s *ChapaService) HandleCallback(txRef string) error {
	// Verify the payment
	response, err := s.VerifyPayment(txRef)
	if err != nil {
		return err
	}

	// Check payment status
	if response.Data.Status != "success" {
		return fmt.Errorf("payment was not successful: %s", response.Data.Status)
	}

	return nil
}
