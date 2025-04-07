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
		UserID:          userID,
		FeaturedImage:   recipe.FeaturedImage,
		Price:           recipe.Price,
	}, nil
}

func (s *RecipeService) GetRecipeByID(ctx context.Context, recipeID string) (*models.Recipe, error) {
	var query struct {
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

	variables := map[string]interface{}{
		"id": recipeID,
	}

	err := s.client.Query(ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	if query.Recipe.ID == "" {
		return nil, errors.New("recipe not found")
	}

	return &models.Recipe{
		ID:              query.Recipe.ID,
		Title:           query.Recipe.Title,
		Description:     query.Recipe.Description,
		PreparationTime: query.Recipe.PreparationTime,
		CategoryID:      &query.Recipe.CategoryID,
		UserID:          query.Recipe.UserID,
		FeaturedImage:   query.Recipe.FeaturedImage,
		Price:           query.Recipe.Price,
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

// CreateRecipeWithRelations creates a recipe along with its related data (ingredients, steps, images)
func (s *RecipeService) CreateRecipeWithRelations(ctx context.Context, req models.CreateRecipeRequest, userID string) (*models.Recipe, error) {
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

	// Insert recipe steps
	if len(req.Steps) > 0 {
		var stepsMutation struct {
			InsertRecipeSteps struct {
				AffectedRows int `graphql:"affected_rows"`
			} `graphql:"insert_recipe_steps(objects: [{recipe_id: $recipe_id, step_number: $step_number, description: $description, image_url: $image_url}])"`
		}

		for _, step := range req.Steps {
			stepVars := map[string]interface{}{
				"recipe_id":   recipe.ID,
				"step_number": step.StepNumber,
				"description": step.Description,
				"image_url":   step.ImageURL,
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
				"id":   ingredient.IngredientID,
				"name": ingredient.IngredientID, // Use the ID as the name if not provided
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
				"ingredient_id": ingredient.IngredientID,
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
			imageVars := map[string]interface{}{
				"recipe_id":   recipe.ID,
				"image_url":   image.ImageURL,
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
