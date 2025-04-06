package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hasura/go-graphql-client"
	"github.com/whiHak/food-recipe-backend/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	client *graphql.Client
}

func NewAuthService(hasuraEndpoint string) *AuthService {
	// Create HTTP client with admin secret header
	httpClient := &http.Client{
		Transport: &headerTransport{
			headers: map[string]string{
				"X-Hasura-Admin-Secret": os.Getenv("HASURA_ADMIN_SECRET"),
			},
		},
	}

	client := graphql.NewClient(hasuraEndpoint, httpClient)
	return &AuthService{client: client}
}

type headerTransport struct {
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func (s *AuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create user mutation
	var mutation struct {
		InsertUsersOne struct {
			ID       string `graphql:"id"`
			Username string `graphql:"username"`
			Email    string `graphql:"email"`
			FullName string `graphql:"full_name"`
		} `graphql:"insert_users_one(object: {username: $username, email: $email, password_hash: $password_hash, full_name: $full_name})"`
	}

	variables := map[string]interface{}{
		"username":      req.Username,
		"email":         req.Email,
		"password_hash": string(hashedPassword),
		"full_name":     req.FullName,
	}

	err = s.client.Mutate(ctx, &mutation, variables)
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(mutation.InsertUsersOne.ID)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := generateJWT(userID.String())
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:    token,
		UserID:   userID.String(),
		Username: mutation.InsertUsersOne.Username,
		Email:    mutation.InsertUsersOne.Email,
		FullName: mutation.InsertUsersOne.FullName,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	var query struct {
		Users []struct {
			ID           string `graphql:"id"`
			Username     string `graphql:"username"`
			Email        string `graphql:"email"`
			FullName     string `graphql:"full_name"`
			PasswordHash string `graphql:"password_hash"`
		} `graphql:"users(where: {email: {_eq: $email}})"`
	}

	variables := map[string]interface{}{
		"email": req.Email,
	}

	err := s.client.Query(ctx, &query, variables)
	if err != nil {
		return nil, err
	}

	// Check if user exists
	if len(query.Users) == 0 {
		return nil, errors.New("user not found")
	}

	user := query.Users[0]

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := generateJWT(userID.String())
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		Token:    token,
		UserID:   userID.String(),
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	}, nil
}

func generateJWT(userID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET not set")
	}

	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
		"iss": "food-recipe-app",
		"https://hasura.io/jwt/claims": map[string]interface{}{
			"x-hasura-default-role":  "user",
			"x-hasura-allowed-roles": []string{"user"},
			"x-hasura-user-id":       userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
