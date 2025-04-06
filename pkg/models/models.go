package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	PasswordHash   string    `json:"password_hash"`
	FullName       string    `json:"full_name"`
	Bio            *string   `json:"bio"`
	ProfilePicture *string   `json:"profile_picture"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Category struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	ImageURL    *string   `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type Recipe struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	PreparationTime int       `json:"preparation_time"`
	CategoryID      *string   `json:"category_id"`
	UserID          string    `json:"user_id"`
	FeaturedImage   string    `json:"featured_image"`
	Price           float64   `json:"price"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type RecipeImage struct {
	ID         uuid.UUID `json:"id"`
	RecipeID   uuid.UUID `json:"recipe_id"`
	ImageURL   string    `json:"image_url"`
	IsFeatured bool      `json:"is_featured"`
	CreatedAt  time.Time `json:"created_at"`
}

type RecipeStep struct {
	ID          uuid.UUID `json:"id"`
	RecipeID    uuid.UUID `json:"recipe_id"`
	StepNumber  int       `json:"step_number"`
	Description string    `json:"description"`
	ImageURL    *string   `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type Ingredient struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type RecipeIngredient struct {
	ID           uuid.UUID `json:"id"`
	RecipeID     uuid.UUID `json:"recipe_id"`
	IngredientID uuid.UUID `json:"ingredient_id"`
	Quantity     string    `json:"quantity"`
	Unit         *string   `json:"unit"`
	CreatedAt    time.Time `json:"created_at"`
}

type RecipeLike struct {
	ID        uuid.UUID `json:"id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RecipeBookmark struct {
	ID        uuid.UUID `json:"id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RecipeComment struct {
	ID        uuid.UUID `json:"id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RecipeRating struct {
	ID        uuid.UUID `json:"id"`
	RecipeID  uuid.UUID `json:"recipe_id"`
	UserID    uuid.UUID `json:"user_id"`
	Rating    int       `json:"rating"`
	CreatedAt time.Time `json:"created_at"`
}

type RecipePurchase struct {
	ID            uuid.UUID `json:"id"`
	RecipeID      uuid.UUID `json:"recipe_id"`
	UserID        uuid.UUID `json:"user_id"`
	Amount        float64   `json:"amount"`
	TransactionID string    `json:"transaction_id"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// Request/Response types
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

type CreateRecipeRequest struct {
	Title           string                  `json:"title"`
	Description     string                  `json:"description"`
	PreparationTime int                     `json:"preparation_time"`
	CategoryID      string                  `json:"category_id"`
	FeaturedImage   string                  `json:"featured_image"`
	Price           float64                 `json:"price"`
	Steps           []Step                  `json:"steps"`
	Ingredients     []RecipeIngredientInput `json:"ingredients"`
}

type Step struct {
	StepNumber  int    `json:"step_number"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

type RecipeIngredientInput struct {
	IngredientID string `json:"ingredient_id"`
	Quantity     string `json:"quantity"`
	Unit         string `json:"unit"`
}
