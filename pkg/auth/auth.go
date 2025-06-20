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
	httpClient := &http.Client{}
	client := graphql.NewClient(hasuraEndpoint, httpClient)
	return &AuthService{client: client}
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
	token, err := GenerateToken(userID.String())
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
	token, err := GenerateToken(userID.String())
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

type CustomClaims struct {
	UserID       string                 `json:"user_id"`
	HasuraClaims map[string]interface{} `json:"https://hasura.io/jwt/claims"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string) (string, error) {
	// Create Hasura-specific claims
	hasuraClaims := map[string]interface{}{
		"x-hasura-allowed-roles": []string{"user"},
		"x-hasura-default-role":  "user",
		"x-hasura-user-id":       userID,
	}

	claims := CustomClaims{
		UserID:       userID,
		HasuraClaims: hasuraClaims,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := []byte(os.Getenv("JWT_SECRET"))

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
