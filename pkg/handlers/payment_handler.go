package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/whiHak/food-recipe-backend/pkg/payment"
)

type PaymentHandler struct {
	chapaService *payment.ChapaService
}

func NewPaymentHandler(chapaService *payment.ChapaService) *PaymentHandler {
	return &PaymentHandler{
		chapaService: chapaService,
	}
}

// VerifyPayment handles payment verification requests
func (h *PaymentHandler) VerifyPayment(c *fiber.Ctx) error {
	txRef := c.Params("txRef")
	if txRef == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Transaction reference is required",
		})
	}

	// Verify payment using Chapa service
	response, err := h.chapaService.VerifyPayment(txRef)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "Payment verified successfully",
		"data":    response.Data,
	})
}
