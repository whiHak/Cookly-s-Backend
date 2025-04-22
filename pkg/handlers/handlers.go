package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"encoding/base64"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/whiHak/food-recipe-backend/pkg/auth"
	"github.com/whiHak/food-recipe-backend/pkg/models"
	"github.com/whiHak/food-recipe-backend/pkg/recipe"
)

type Handler struct {
	authService   *auth.AuthService
	recipeService *recipe.RecipeService
}

func NewHandler(hasuraEndpoint string) *Handler {
	return &Handler{
		authService:   auth.NewAuthService(hasuraEndpoint),
		recipeService: recipe.NewRecipeService(hasuraEndpoint),
	}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

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

	return c.Status(http.StatusCreated).JSON(resp)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
		})
	}

	resp, err := h.authService.Login(c.Context(), req)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "invalid password" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid credentials",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(resp)
}

func (h *Handler) CreateRecipe(c *fiber.Ctx) error {
	// Get user ID from context
	userIDStr := c.Locals("user_id").(string)
	if userIDStr == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized: no user ID found",
		})
	}

	fmt.Printf("User ID: %s\n", userIDStr)

	var req models.CreateRecipeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Title == "" || req.Description == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Title and description are required",
		})
	}

	// Get token from Authorization header
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	recipe, err := h.recipeService.CreateRecipeWithRelations(ctx, req, userIDStr)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusCreated).JSON(recipe)
}

func (h *Handler) GetRecipe(c *fiber.Ctx) error {
	// Get recipe ID from the URL parameter
	recipeID := c.Params("id")

	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	// Fetch the recipe by ID
	recipe, err := h.recipeService.GetRecipeByID(c.Context(), recipeID)
	if err != nil {
		if err.Error() == "recipe not found" {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Recipe not found",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(recipe)
}

func (h *Handler) UpdateRecipe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Get token from Authorization header
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	fmt.Println("Received payload:", string(c.Body()))
	var req models.CreateRecipeRequest
	if err := c.BodyParser(&req); err != nil {
		fmt.Println("Error parsing body:", err)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	updatedRecipe, err := h.recipeService.UpdateRecipe(ctx, recipeUUID.String(), req, userUUID.String())
	if err != nil {
		if err.Error() == "recipe not found" {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Recipe not found",
			})
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to update this recipe",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(updatedRecipe)
}

func (h *Handler) DeleteRecipe(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	// Get token from Authorization header
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	deletedRecipe, err := h.recipeService.DeleteRecipe(ctx, recipeUUID.String(), userUUID.String())
	if err != nil {
		if err.Error() == "recipe not found" {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "Recipe not found",
			})
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return c.Status(http.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to delete this recipe",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"message": "Recipe deleted successfully",
		"recipe":  deletedRecipe,
	})
}

func (h *Handler) LikeRecipe(c *fiber.Ctx) error {
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	response, err := h.recipeService.LikeRecipe(ctx, recipeUUID, userUUID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.SendStatus(http.StatusOK)
	return c.JSON(response)
}

func (h *Handler) UnlikeRecipe(c *fiber.Ctx) error {
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	response, err := h.recipeService.UnlikeRecipe(ctx, recipeUUID, userUUID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.Status(http.StatusOK)
	return c.JSON(response)
}

func (h *Handler) BookmarkRecipe(c *fiber.Ctx) error {

	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	res, err := h.recipeService.BookmarkRecipe(ctx, recipeUUID, userUUID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.SendStatus(http.StatusOK)
	return c.JSON(res)
}

func (h *Handler) UnbookmarkRecipe(c *fiber.Ctx) error {
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	res, err := h.recipeService.UnbookmarkRecipe(ctx, recipeUUID, userUUID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.SendStatus(http.StatusOK)
	return c.JSON(res)
}

func (h *Handler) RateRecipe(c *fiber.Ctx) error {
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	var req struct {
		Rating int `json:"rating"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	res, err := h.recipeService.RateRecipe(ctx, recipeUUID, userUUID, req.Rating)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.SendStatus(http.StatusOK)
	return c.JSON(res)
}

func (h *Handler) CommentOnRecipe(c *fiber.Ctx) error {
	token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "No authorization token provided",
		})
	}

	// Create context with token
	ctx := context.WithValue(c.Context(), "token", token)

	userID := c.Locals("user_id").(string)
	if userID == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	var req struct {
		Content string `json:"comment"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Comment content is required",
		})
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	res, err := h.recipeService.CommentOnRecipe(ctx, recipeUUID, userUUID, req.Content)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	c.SendStatus(http.StatusOK)
	return c.JSON(res)
}

func (h *Handler) GetRecipeComments(c *fiber.Ctx) error {
	// Get recipe ID from the URL parameter
	recipeID := c.Params("id")
	if recipeID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Recipe ID is required",
		})
	}

	// Parse recipe ID
	recipeUUID, err := uuid.Parse(recipeID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid recipe ID",
		})
	}

	// Fetch comments for the recipe
	comments, err := h.recipeService.GetCommentsOnRecipe(c.Context(), recipeUUID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(comments)
}

func (h *Handler) UploadImage(c *fiber.Ctx) error {
	var req struct {
		Image string `json:"image"` // base64 encoded image
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate base64 image
	if req.Image == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Image is required",
		})
	}

	// Remove data:image/... prefix if present
	base64Data := req.Image
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid base64 image",
		})
	}

	// Generate unique filename
	filename := uuid.New().String() + ".webp"
	uploadDir := "./uploads"
	filepath := path.Join(uploadDir, filename)

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Save the file
	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save image",
		})
	}

	// Return the image URL
	imageURL := fmt.Sprintf("/uploads/%s", filename)
	return c.JSON(fiber.Map{
		"url": imageURL,
	})
}

func (h *Handler) GetAllRecipes(c *fiber.Ctx) error {
	recipes, err := h.recipeService.GetAllRecipes(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(recipes)
}
