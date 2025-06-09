package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/whiHak/food-recipe-backend/pkg/auth"
	"github.com/whiHak/food-recipe-backend/pkg/models"
)

type Handler struct {
	authService *auth.AuthService
}

func NewHandler(hasuraEndpoint string) *Handler {
	return &Handler{
		authService: auth.NewAuthService(hasuraEndpoint),
	}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}
	fmt.Println("raw.......", raw)

	input, ok := raw["input"].(map[string]interface{})
	if !ok {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input payload",
		})
	}
	fmt.Println("input.......", input)
	var req models.RegisterRequest
	inputBytes, _ := json.Marshal(input)
	if err := json.Unmarshal(inputBytes, &req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input fields",
		})
	}
	fmt.Println("req....", req)

	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Username, email and password are required",
		})
	}

	resp, err := h.authService.Register(c.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			return c.Status(http.StatusConflict).JSON(fiber.Map{
				"error": "Username or email already exists",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Wrap response to match frontend expectation
	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"token": resp.Token,
		"user": fiber.Map{
			"id":        resp.UserID,
			"username":  resp.Username,
			"email":     resp.Email,
			"full_name": resp.FullName,
		},
	})
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var raw map[string]interface{}
	if err := c.BodyParser(&raw); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	input, ok := raw["input"].(map[string]interface{})
	if !ok {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input payload",
		})
	}

	// Now parse input into your LoginRequest struct
	var req models.LoginRequest
	inputBytes, _ := json.Marshal(input)
	if err := json.Unmarshal(inputBytes, &req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid input fields",
		})
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "Email and password are required",
		})
	}

	resp, err := h.authService.Login(c.Context(), req)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "invalid password" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid credentials",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	// Wrap response to match frontend expectation
	return c.JSON(fiber.Map{
		"token": resp.Token,
		"user": fiber.Map{
			"id":        resp.UserID,
			"username":  resp.Username,
			"email":     resp.Email,
			"full_name": resp.FullName,
		},
	})
}

// Add payment verification handler
func (h *Handler) VerifyPayment(c *fiber.Ctx) error {
	txRef := c.Query("tx_ref")
	if txRef == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "tx_ref is required",
		})
	}

	chapaSecret := os.Getenv("CHAPA_SECRET_KEY")
	if chapaSecret == "" {
		chapaSecret = "CHASECK_TEST-EfqZuWLxNvk9sa0IXsim3AYvEc6THs1J" // fallback for dev
	}

	url := "https://api.chapa.co/v1/transaction/verify/" + txRef
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	req.Header.Set("Authorization", "Bearer "+chapaSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(resp.StatusCode).JSON(result)
}
