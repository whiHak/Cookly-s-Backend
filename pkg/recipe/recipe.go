package recipe

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hasura/go-graphql-client"
	"github.com/whiHak/food-recipe-backend/pkg/models"
)

type RecipeService struct {
	client *graphql.Client
	url    string
}

type headerTransport struct {
	token string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.token))
	}
	return http.DefaultTransport.RoundTrip(req)
}

func NewRecipeService(hasuraEndpoint string) *RecipeService {
	return &RecipeService{
		client: graphql.NewClient(hasuraEndpoint, nil),
		url:    hasuraEndpoint,
	}
}

func (s *RecipeService) withToken(token string) *graphql.Client {
	return graphql.NewClient(s.url, &http.Client{
		Transport: &headerTransport{token: token},
	})
}

// Define separate structs for each type
type RecipeStep struct {
	ID          string  `graphql:"id"`
	StepNumber  int     `graphql:"step_number"`
	Description string  `graphql:"description"`
	ImageURL    *string `graphql:"image_url"`
}

type RecipeIngredient struct {
	ID           string  `graphql:"id"`
	IngredientID string  `graphql:"ingredient_id"`
	Quantity     string  `graphql:"quantity"`
	Unit         *string `graphql:"unit"`
	Ingredient   struct {
		ID   string `graphql:"id"`
		Name string `graphql:"name"`
	} `graphql:"ingredient"`
}

type RecipeImage struct {
	ID         string `graphql:"id"`
	ImageURL   string `graphql:"image_url"`
	IsFeatured bool   `graphql:"is_featured"`
}

type Recipe struct {
	ID                string             `graphql:"id"`
	Title             string             `graphql:"title"`
	Description       *string            `graphql:"description"`
	PreparationTime   int                `graphql:"preparation_time"`
	CategoryID        string             `graphql:"category_id"`
	UserID            string             `graphql:"user_id"`
	FeaturedImage     string             `graphql:"featured_image"`
	Price             float64            `graphql:"price"`
	RecipeSteps       []RecipeStep       `graphql:"recipe_steps(order_by: {step_number: asc})"`
	RecipeIngredients []RecipeIngredient `graphql:"recipe_ingredients"`
	RecipeImages      []RecipeImage      `graphql:"recipe_images"`
}

// CreateRecipeWithRelations creates a recipe along with its related data (ingredients, steps, images)
func (s *RecipeService) CreateRecipeWithRelations(ctx context.Context, req models.CreateRecipeRequest, userID string) (*models.Recipe, error) {
	// Get token from context
	token, ok := ctx.Value("token").(string)
	if !ok {
		return nil, errors.New("no token found in context")
	}

	// Use client with token
	client := s.withToken(token)

	// First, upload the featured image and get its URL
	featuredImageURL, err := s.uploadImage(ctx, req.FeaturedImage)
	if err != nil {
		return nil, fmt.Errorf("failed to upload featured image: %v", err)
	}

	// Create recipe mutation
	var mutation struct {
		InsertRecipes struct {
			AffectedRows int `graphql:"affected_rows"`
		} `graphql:"insert_recipes(objects: [{title: $title, description: $description, preparation_time: $preparation_time, category_id: $category_id, user_id: $user_id, featured_image: $featured_image, price: $price}])"`
	}

	mutationVars := map[string]interface{}{
		"title":            req.Title,
		"description":      req.Description,
		"preparation_time": req.PreparationTime,
		"category_id":      req.CategoryID,
		"user_id":          userID,
		"featured_image":   featuredImageURL,
		"price":            req.Price,
	}

	err = client.Mutate(ctx, &mutation, mutationVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe: %v", err)
	}

	if mutation.InsertRecipes.AffectedRows == 0 {
		return nil, fmt.Errorf("no recipe was created")
	}

	// Query the created recipe
	var query struct {
		Recipes []struct {
			ID              string  `graphql:"id"`
			Title           string  `graphql:"title"`
			Description     string  `graphql:"description"`
			PreparationTime int     `graphql:"preparation_time"`
			CategoryID      string  `graphql:"category_id"`
			UserID          string  `graphql:"user_id"`
			FeaturedImage   string  `graphql:"featured_image"`
			Price           float64 `graphql:"price"`
		} `graphql:"recipes(where: {title: {_eq: $title}, user_id: {_eq: $user_id}}, limit: 1, order_by: {created_at: desc})"`
	}

	queryVars := map[string]interface{}{
		"title":   req.Title,
		"user_id": userID,
	}

	err = client.Query(ctx, &query, queryVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created recipe: %v", err)
	}

	if len(query.Recipes) == 0 {
		return nil, fmt.Errorf("created recipe not found")
	}

	recipe := query.Recipes[0]

	// Insert recipe steps
	if len(req.Steps) > 0 {
		var stepsMutation struct {
			InsertRecipeSteps struct {
				AffectedRows int `graphql:"affected_rows"`
			} `graphql:"insert_recipe_steps(objects: [{recipe_id: $recipe_id, step_number: $step_number, description: $description, image_url: $image_url}])"`
		}

		for _, step := range req.Steps {
			var stepImageURL *string
			if step.ImageBase64 != nil {
				// Upload step image if provided
				uploadedURL, err := s.uploadImage(ctx, *step.ImageBase64)
				if err != nil {
					return nil, fmt.Errorf("failed to upload step image: %v", err)
				}
				stepImageURL = &uploadedURL
			}

			stepVars := map[string]interface{}{
				"recipe_id":   recipe.ID,
				"step_number": step.StepNumber,
				"description": step.Description,
				"image_url":   stepImageURL,
			}

			err = client.Mutate(ctx, &stepsMutation, stepVars)
			if err != nil {
				return nil, fmt.Errorf("failed to create recipe step: %v", err)
			}

			if stepsMutation.InsertRecipeSteps.AffectedRows == 0 {
				return nil, fmt.Errorf("no recipe step was created")
			}
		}
	}

	// Insert recipe ingredients
	if len(req.Ingredients) > 0 {
		// First ensure all ingredients exist
		for _, ingredient := range req.Ingredients {
			var ingredientMutation struct {
				InsertIngredients struct {
					AffectedRows int `graphql:"affected_rows"`
				} `graphql:"insert_ingredients(objects: [{id: $id, name: $name}], on_conflict: {constraint: ingredients_pkey, update_columns: []})"`
			}

			ingredientVars := map[string]interface{}{
				"id":   graphql.String(ingredient.IngredientID.String()),
				"name": graphql.String(ingredient.IngredientID.String()), // Use the ID as the name if not provided
			}

			err = client.Mutate(ctx, &ingredientMutation, ingredientVars)
			if err != nil {
				return nil, fmt.Errorf("failed to create/update ingredient: %v", err)
			}
		}

		// Now create recipe ingredients
		var ingredientsMutation struct {
			InsertRecipeIngredients struct {
				AffectedRows int `graphql:"affected_rows"`
			} `graphql:"insert_recipe_ingredients(objects: [{recipe_id: $recipe_id, ingredient_id: $ingredient_id, quantity: $quantity, unit: $unit}])"`
		}

		for _, ingredient := range req.Ingredients {
			ingredientVars := map[string]interface{}{
				"recipe_id":     recipe.ID,
				"ingredient_id": graphql.String(ingredient.IngredientID.String()),
				"quantity":      ingredient.Quantity,
				"unit":          ingredient.Unit,
			}

			err = client.Mutate(ctx, &ingredientsMutation, ingredientVars)
			if err != nil {
				return nil, fmt.Errorf("failed to create recipe ingredient: %v", err)
			}

			if ingredientsMutation.InsertRecipeIngredients.AffectedRows == 0 {
				return nil, fmt.Errorf("no recipe ingredient was created")
			}
		}
	}

	// Insert recipe images
	if len(req.Images) > 0 {
		var imagesMutation struct {
			InsertRecipeImages struct {
				AffectedRows int `graphql:"affected_rows"`
			} `graphql:"insert_recipe_images(objects: [{recipe_id: $recipe_id, image_url: $image_url, is_featured: $is_featured}])"`
		}

		for _, image := range req.Images {
			// Upload image and get URL
			imageURL, err := s.uploadImage(ctx, image.ImageBase64)
			if err != nil {
				return nil, fmt.Errorf("failed to upload recipe image: %v", err)
			}

			imageVars := map[string]interface{}{
				"recipe_id":   recipe.ID,
				"image_url":   imageURL,
				"is_featured": image.IsFeatured,
			}

			err = client.Mutate(ctx, &imagesMutation, imageVars)
			if err != nil {
				return nil, fmt.Errorf("failed to create recipe image: %v", err)
			}

			if imagesMutation.InsertRecipeImages.AffectedRows == 0 {
				return nil, fmt.Errorf("no recipe image was created")
			}
		}
	}

	return &models.Recipe{
		ID:              recipe.ID,
		Title:           recipe.Title,
		Description:     &recipe.Description,
		PreparationTime: recipe.PreparationTime,
		CategoryID:      &recipe.CategoryID,
		UserID:          recipe.UserID,
		FeaturedImage:   recipe.FeaturedImage,
		Price:           recipe.Price,
	}, nil
}

// Helper function to upload image and return URL
func (s *RecipeService) uploadImage(ctx context.Context, base64Image string) (string, error) {
	// TODO: Implement actual image upload logic here
	// This should:
	// 1. Decode the base64 string
	// 2. Upload to your storage service (e.g., S3, GCS)
	// 3. Return the public URL
	return base64Image, nil
}

func (s *RecipeService) GetAllRecipes(ctx context.Context) ([]*models.Recipe, error) {
	// Get token from context
	token, ok := ctx.Value("token").(string)
	if !ok || token == "" {
		return nil, errors.New("no authorization token provided")
	}

	// Create a new client with the token
	client := s.withToken(token)

	// First, get all recipes
	var recipesQuery struct {
		Recipes []struct {
			ID              string  `graphql:"id"`
			Title           string  `graphql:"title"`
			Description     *string `graphql:"description"`
			PreparationTime int     `graphql:"preparation_time"`
			CategoryID      string  `graphql:"category_id"`
			UserID          string  `graphql:"user_id"`
			FeaturedImage   string  `graphql:"featured_image"`
			Price           float64 `graphql:"price"`
		} `graphql:"recipes"`
	}

	err := client.Query(ctx, &recipesQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipes: %v", err)
	}

	// Get all recipe steps
	var stepsQuery struct {
		Steps []struct {
			ID          string  `graphql:"id"`
			RecipeID    string  `graphql:"recipe_id"`
			StepNumber  int     `graphql:"step_number"`
			Description string  `graphql:"description"`
			ImageURL    *string `graphql:"image_url"`
		} `graphql:"recipe_steps(order_by: {step_number: asc})"`
	}

	err = client.Query(ctx, &stepsQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe steps: %v", err)
	}

	// Get all recipe_ingredients
	var recipeIngredientsQuery struct {
		RecipeIngredients []struct {
			ID           string  `graphql:"id"`
			RecipeID     string  `graphql:"recipe_id"`
			IngredientID string  `graphql:"ingredient_id"`
			Quantity     string  `graphql:"quantity"`
			Unit         *string `graphql:"unit"`
		} `graphql:"recipe_ingredients"`
	}

	err = client.Query(ctx, &recipeIngredientsQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe ingredients: %v", err)
	}

	// Get all ingredients
	var ingredientsQuery struct {
		Ingredients []struct {
			ID   string `graphql:"id"`
			Name string `graphql:"name"`
		} `graphql:"ingredients"`
	}

	err = client.Query(ctx, &ingredientsQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ingredients: %v", err)
	}

	// Create ingredients map for quick lookup
	ingredientsMap := make(map[string]string) // Map ingredient ID to name
	for _, ingredient := range ingredientsQuery.Ingredients {
		ingredientsMap[ingredient.ID] = ingredient.Name
	}

	// Get all recipe images
	var imagesQuery struct {
		Images []struct {
			ID         string `graphql:"id"`
			RecipeID   string `graphql:"recipe_id"`
			ImageURL   string `graphql:"image_url"`
			IsFeatured bool   `graphql:"is_featured"`
		} `graphql:"recipe_images"`
	}

	err = client.Query(ctx, &imagesQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe images: %v", err)
	}

	// Create a map to store steps, ingredients, and images by recipe ID
	stepsByRecipe := make(map[string][]models.RecipeStep)
	for _, step := range stepsQuery.Steps {
		stepsByRecipe[step.RecipeID] = append(stepsByRecipe[step.RecipeID], models.RecipeStep{
			ID:          uuid.MustParse(step.ID),
			RecipeID:    uuid.MustParse(step.RecipeID),
			StepNumber:  step.StepNumber,
			Description: step.Description,
			ImageURL:    step.ImageURL,
		})
	}

	ingredientsByRecipe := make(map[string][]models.RecipeIngredient)
	for _, recipeIngredient := range recipeIngredientsQuery.RecipeIngredients {
		ingredientsByRecipe[recipeIngredient.RecipeID] = append(ingredientsByRecipe[recipeIngredient.RecipeID], models.RecipeIngredient{
			ID:           uuid.MustParse(recipeIngredient.ID),
			RecipeID:     uuid.MustParse(recipeIngredient.RecipeID),
			IngredientID: uuid.MustParse(recipeIngredient.IngredientID),
			Quantity:     recipeIngredient.Quantity,
			Unit:         recipeIngredient.Unit,
		})
	}

	imagesByRecipe := make(map[string][]models.RecipeImage)
	for _, image := range imagesQuery.Images {
		imagesByRecipe[image.RecipeID] = append(imagesByRecipe[image.RecipeID], models.RecipeImage{
			ID:         uuid.MustParse(image.ID),
			RecipeID:   uuid.MustParse(image.RecipeID),
			ImageURL:   image.ImageURL,
			IsFeatured: image.IsFeatured,
		})
	}

	// Build the final recipes slice
	recipes := make([]*models.Recipe, len(recipesQuery.Recipes))
	for i, recipe := range recipesQuery.Recipes {
		recipes[i] = &models.Recipe{
			ID:              recipe.ID,
			Title:           recipe.Title,
			Description:     recipe.Description,
			PreparationTime: recipe.PreparationTime,
			CategoryID:      &recipe.CategoryID,
			UserID:          recipe.UserID,
			FeaturedImage:   recipe.FeaturedImage,
			Price:           recipe.Price,
			Steps:           stepsByRecipe[recipe.ID],
			Ingredients:     ingredientsByRecipe[recipe.ID],
			Images:          imagesByRecipe[recipe.ID],
		}
	}

	return recipes, nil
}

func (s *RecipeService) CreateRecipe(ctx context.Context, req models.CreateRecipeRequest, userID string) (*models.Recipe, error) {
	// Get token from context
	token, ok := ctx.Value("token").(string)
	if !ok {
		return nil, errors.New("no token found in context")
	}

	// Use client with token
	client := s.withToken(token)

	// Create recipe mutation
	var mutation struct {
		InsertRecipes struct {
			AffectedRows int `graphql:"affected_rows"`
		} `graphql:"insert_recipes(objects: [{title: $title, description: $description, preparation_time: $preparation_time, category_id: $category_id, user_id: $user_id, featured_image: $featured_image, price: $price}])"`
	}

	mutationVars := map[string]interface{}{
		"title":            req.Title,
		"description":      req.Description,
		"preparation_time": req.PreparationTime,
		"category_id":      req.CategoryID,
		"user_id":          userID,
		"featured_image":   req.FeaturedImage,
		"price":            req.Price,
	}

	err := client.Mutate(ctx, &mutation, mutationVars)
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe: %v", err)
	}

	if mutation.InsertRecipes.AffectedRows == 0 {
		return nil, fmt.Errorf("no recipe was created")
	}

	// Query the created recipe
	var query struct {
		Recipes []struct {
			ID              string  `graphql:"id"`
			Title           string  `graphql:"title"`
			Description     string  `graphql:"description"`
			PreparationTime int     `graphql:"preparation_time"`
			CategoryID      string  `graphql:"category_id"`
			UserID          string  `graphql:"user_id"`
			FeaturedImage   string  `graphql:"featured_image"`
			Price           float64 `graphql:"price"`
		} `graphql:"recipes(where: {title: {_eq: $title}, user_id: {_eq: $user_id}}, limit: 1, order_by: {created_at: desc})"`
	}

	queryVars := map[string]interface{}{
		"title":   req.Title,
		"user_id": userID,
	}

	err = client.Query(ctx, &query, queryVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created recipe: %v", err)
	}

	if len(query.Recipes) == 0 {
		return nil, fmt.Errorf("created recipe not found")
	}

	recipe := query.Recipes[0]
	return &models.Recipe{
		ID:              recipe.ID,
		Title:           recipe.Title,
		Description:     &recipe.Description,
		PreparationTime: recipe.PreparationTime,
		CategoryID:      &req.CategoryID,
		UserID:          recipe.UserID,
		FeaturedImage:   recipe.FeaturedImage,
		Price:           recipe.Price,
	}, nil
}

func (s *RecipeService) GetRecipeByID(ctx context.Context, recipeID string) (*models.Recipe, error) {
	token, ok := ctx.Value("token").(string)
	if !ok || token == "" {
		return nil, errors.New("no authorization token provided")
	}

	// Create a new client with the token
	client := s.withToken(token)

	// Fetch the recipe details
	var recipeQuery struct {
		Recipe struct {
			ID              string  `graphql:"id"`
			Title           string  `graphql:"title"`
			Description     *string `graphql:"description"`
			PreparationTime int     `graphql:"preparation_time"`
			CategoryID      string  `graphql:"category_id"`
			UserID          string  `graphql:"user_id"`
			FeaturedImage   string  `graphql:"featured_image"`
			Price           float64 `graphql:"price"`
		} `graphql:"recipes_by_pk(id: $id)"`
	}

	recipeVars := map[string]interface{}{
		"id": recipeID,
	}

	err := client.Query(ctx, &recipeQuery, recipeVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe: %v", err)
	}

	if recipeQuery.Recipe.ID == "" {
		return nil, errors.New("recipe not found")
	}

	// Fetch the recipe steps
	var stepsQuery struct {
		Steps []struct {
			ID          string  `graphql:"id"`
			RecipeID    string  `graphql:"recipe_id"`
			StepNumber  int     `graphql:"step_number"`
			Description string  `graphql:"description"`
			ImageURL    *string `graphql:"image_url"`
		} `graphql:"recipe_steps(where: {recipe_id: {_eq: $recipe_id}}, order_by: {step_number: asc})"`
	}

	stepsVars := map[string]interface{}{
		"recipe_id": recipeID,
	}

	err = client.Query(ctx, &stepsQuery, stepsVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe steps: %v", err)
	}

	// Fetch the recipe ingredients
	var ingredientsQuery struct {
		Ingredients []struct {
			ID           string  `graphql:"id"`
			RecipeID     string  `graphql:"recipe_id"`
			IngredientID string  `graphql:"ingredient_id"`
			Quantity     string  `graphql:"quantity"`
			Unit         *string `graphql:"unit"`
			Ingredient   struct {
				ID   string `graphql:"id"`
				Name string `graphql:"name"`
			} `graphql:"ingredient"`
		} `graphql:"recipe_ingredients(where: {recipe_id: {_eq: $recipe_id}})"`
	}

	ingredientsVars := map[string]interface{}{
		"recipe_id": recipeID,
	}

	err = client.Query(ctx, &ingredientsQuery, ingredientsVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe ingredients: %v", err)
	}

	// Fetch the recipe images
	var imagesQuery struct {
		Images []struct {
			ID         string `graphql:"id"`
			RecipeID   string `graphql:"recipe_id"`
			ImageURL   string `graphql:"image_url"`
			IsFeatured bool   `graphql:"is_featured"`
		} `graphql:"recipe_images(where: {recipe_id: {_eq: $recipe_id}})"`
	}

	imagesVars := map[string]interface{}{
		"recipe_id": recipeID,
	}

	err = client.Query(ctx, &imagesQuery, imagesVars)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe images: %v", err)
	}

	// Convert steps
	steps := make([]models.RecipeStep, len(stepsQuery.Steps))
	for i, step := range stepsQuery.Steps {
		steps[i] = models.RecipeStep{
			ID:          uuid.MustParse(step.ID),
			RecipeID:    uuid.MustParse(step.RecipeID),
			StepNumber:  step.StepNumber,
			Description: step.Description,
			ImageURL:    step.ImageURL,
		}
	}

	// Convert ingredients
	ingredients := make([]models.RecipeIngredient, len(ingredientsQuery.Ingredients))
	for i, ingredient := range ingredientsQuery.Ingredients {
		ingredients[i] = models.RecipeIngredient{
			ID:           uuid.MustParse(ingredient.ID),
			RecipeID:     uuid.MustParse(ingredient.RecipeID),
			IngredientID: uuid.MustParse(ingredient.IngredientID),
			Quantity:     ingredient.Quantity,
			Unit:         ingredient.Unit,
		}
	}

	// Convert images
	images := make([]models.RecipeImage, len(imagesQuery.Images))
	for i, image := range imagesQuery.Images {
		images[i] = models.RecipeImage{
			ID:         uuid.MustParse(image.ID),
			RecipeID:   uuid.MustParse(image.RecipeID),
			ImageURL:   image.ImageURL,
			IsFeatured: image.IsFeatured,
		}
	}

	// Build the final recipe object
	return &models.Recipe{
		ID:              recipeQuery.Recipe.ID,
		Title:           recipeQuery.Recipe.Title,
		Description:     recipeQuery.Recipe.Description,
		PreparationTime: recipeQuery.Recipe.PreparationTime,
		CategoryID:      &recipeQuery.Recipe.CategoryID,
		UserID:          recipeQuery.Recipe.UserID,
		FeaturedImage:   recipeQuery.Recipe.FeaturedImage,
		Price:           recipeQuery.Recipe.Price,
		Steps:           steps,
		Ingredients:     ingredients,
		Images:          images,
	}, nil
}

func (s *RecipeService) UpdateRecipe(ctx context.Context, recipeID string, req models.CreateRecipeRequest, userID string) error {
	// First verify the recipe belongs to the user
	recipe, err := s.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return err
	}

	if recipe.UserID != userID {
		return errors.New("unauthorized: recipe does not belong to user")
	}

	var mutation struct {
		UpdateRecipe struct {
			AffectedRows int
		} `graphql:"update_recipes_by_pk(pk_columns: {id: $id}, _set: $updates)"`
	}

	variables := map[string]interface{}{
		"id": recipeID,
		"updates": map[string]interface{}{
			"title":            req.Title,
			"description":      req.Description,
			"preparation_time": req.PreparationTime,
			"category_id":      req.CategoryID,
			"featured_image":   req.FeaturedImage,
			"price":            req.Price,
		},
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) DeleteRecipe(ctx context.Context, recipeID string, userID string) error {
	// First verify the recipe belongs to the user
	recipe, err := s.GetRecipeByID(ctx, recipeID)
	if err != nil {
		return err
	}

	if recipe.UserID != userID {
		return errors.New("unauthorized: recipe does not belong to user")
	}

	var mutation struct {
		DeleteRecipe struct {
			AffectedRows int
		} `graphql:"delete_recipes_by_pk(id: $id)"`
	}

	variables := map[string]interface{}{
		"id": recipeID,
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) LikeRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID) error {
	var mutation struct {
		InsertLike struct {
			AffectedRows int
		} `graphql:"insert_recipe_likes_one(object: $like)"`
	}

	variables := map[string]interface{}{
		"like": map[string]interface{}{
			"recipe_id": recipeID,
			"user_id":   userID,
		},
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) UnlikeRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID) error {
	var mutation struct {
		DeleteLike struct {
			AffectedRows int
		} `graphql:"delete_recipe_likes(where: {recipe_id: {_eq: $recipe_id}, user_id: {_eq: $user_id}})"`
	}

	variables := map[string]interface{}{
		"recipe_id": recipeID.String(),
		"user_id":   userID.String(),
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) BookmarkRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID) error {
	var mutation struct {
		InsertBookmark struct {
			AffectedRows int
		} `graphql:"insert_recipe_bookmarks_one(object: $bookmark)"`
	}

	variables := map[string]interface{}{
		"bookmark": map[string]interface{}{
			"recipe_id": recipeID,
			"user_id":   userID,
		},
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) UnbookmarkRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID) error {
	var mutation struct {
		DeleteBookmark struct {
			AffectedRows int
		} `graphql:"delete_recipe_bookmarks(where: {recipe_id: {_eq: $recipe_id}, user_id: {_eq: $user_id}})"`
	}

	variables := map[string]interface{}{
		"recipe_id": recipeID.String(),
		"user_id":   userID.String(),
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) RateRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID, rating int) error {
	if rating < 1 || rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}

	var mutation struct {
		InsertRating struct {
			AffectedRows int
		} `graphql:"insert_recipe_ratings_one(object: $rating, on_conflict: {constraint: recipe_ratings_recipe_id_user_id_key, update_columns: [rating]})"`
	}

	variables := map[string]interface{}{
		"rating": map[string]interface{}{
			"recipe_id": recipeID,
			"user_id":   userID,
			"rating":    rating,
		},
	}

	return s.client.Mutate(ctx, &mutation, variables)
}

func (s *RecipeService) CommentOnRecipe(ctx context.Context, recipeID uuid.UUID, userID uuid.UUID, content string) error {
	var mutation struct {
		InsertComment struct {
			AffectedRows int
		} `graphql:"insert_recipe_comments_one(object: $comment)"`
	}

	variables := map[string]interface{}{
		"comment": map[string]interface{}{
			"recipe_id": recipeID,
			"user_id":   userID,
			"content":   content,
		},
	}

	return s.client.Mutate(ctx, &mutation, variables)
}
