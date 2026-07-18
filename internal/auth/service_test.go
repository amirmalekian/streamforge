package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"streamforge/internal/config"
	"streamforge/internal/database"
)

func TestNewService(t *testing.T) {
	repo := &database.Repository{}
	cfg := config.JWTConfig{
		Secret: "test-secret",
		Expiry: time.Hour,
	}

	svc := NewService(repo, cfg)
	assert.NotNil(t, svc)
	assert.Equal(t, cfg.Expiry, svc.expiry)
	assert.Equal(t, []byte(cfg.Secret), svc.jwtKey)
}

func TestService_GenerateToken(t *testing.T) {
	repo := &database.Repository{}
	cfg := config.JWTConfig{
		Secret: "test-secret",
		Expiry: time.Hour,
	}
	svc := NewService(repo, cfg)

	userID := uuid.New()
	email := "test@example.com"

	token, err := svc.generateToken(userID, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestService_ValidateToken_Invalid(t *testing.T) {
	repo := &database.Repository{}
	cfg := config.JWTConfig{
		Secret: "test-secret",
		Expiry: time.Hour,
	}
	svc := NewService(repo, cfg)

	claims, err := svc.ValidateToken("invalid.token.string")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Nil(t, claims)
}

func TestService_ValidateToken_WrongSecret(t *testing.T) {
	repo := &database.Repository{}
	cfg1 := config.JWTConfig{Secret: "secret1", Expiry: time.Hour}
	cfg2 := config.JWTConfig{Secret: "secret2", Expiry: time.Hour}

	svc1 := NewService(repo, cfg1)
	svc2 := NewService(repo, cfg2)

	userID := uuid.New()
	email := "test@example.com"

	token, err := svc1.generateToken(userID, email)
	assert.NoError(t, err)

	claims, err := svc2.ValidateToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
	assert.Nil(t, claims)
}

func TestRegisterRequest_Valid(t *testing.T) {
	req := RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	assert.NotEmpty(t, req.Email)
	assert.NotEmpty(t, req.Password)
	assert.GreaterOrEqual(t, len(req.Password), 8)
}

func TestLoginRequest_Valid(t *testing.T) {
	req := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	assert.NotEmpty(t, req.Email)
	assert.NotEmpty(t, req.Password)
}
