package main

import (
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/whiHak/food-recipe-backend/pkg/handlers"
	"github.com/whiHak/food-recipe-backend/pkg/middleware"
	"github.com/whiHak/food-recipe-backend/pkg/payment"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file")
	}

	// Get Hasura endpoint from environment
	hasuraEndpoint := os.Getenv("HASURA_ENDPOINT")
	if hasuraEndpoint == "" {
		hasuraEndpoint = "http://localhost:8080/v1/graphql"
	}

	// Create a new Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default error handling
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Add middleware
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	// Initialize services
	chapaService := payment.NewChapaService()

	// Create handlers
	h := handlers.NewHandler(hasuraEndpoint)
	ph := handlers.NewPaymentHandler(chapaService)

	// Public routes
	app.Post("/api/auth/register", h.Register)
	app.Post("/api/auth/login", h.Login)
	app.Get("/api/recipes/all", h.GetAllRecipes)
	app.Get("/api/recipes/:id", h.GetRecipe)
	app.Get("/api/recipes/:id/comments", h.GetRecipeComments)

	// Protected routes
	api := app.Group("/api", middleware.AuthMiddleware())

	// Recipe routes
	recipes := api.Group("/recipes")
	recipes.Post("/", h.CreateRecipe)
	recipes.Put("/:id", h.UpdateRecipe)
	recipes.Delete("/:id", h.DeleteRecipe)
	recipes.Post("/upload", h.UploadImage)

	// Recipe interaction routes
	recipes.Post("/:id/like", h.LikeRecipe)
	recipes.Delete("/:id/like", h.UnlikeRecipe)
	recipes.Post("/:id/bookmark", h.BookmarkRecipe)
	recipes.Delete("/:id/bookmark", h.UnbookmarkRecipe)
	recipes.Post("/:id/rate", h.RateRecipe)
	recipes.Post("/:id/comment", h.CommentOnRecipe)

	// Payment routes
	payments := api.Group("/payments")
	payments.Get("/verify/:txRef", ph.VerifyPayment)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
