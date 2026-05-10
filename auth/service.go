package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"project_smt6/domain"
	"project_smt6/internal/config"
	"project_smt6/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid email or password")

type Service struct {
	cfg   config.AuthConfig
	users repository.UserRepository
}

type Claims struct {
	UserID      uint            `json:"user_id"`
	Email       string          `json:"email"`
	Role        domain.RoleName `json:"role"`
	WorkspaceID *uint           `json:"workspace_id,omitempty"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

func NewService(cfg config.AuthConfig, users repository.UserRepository) *Service {
	return &Service{
		cfg:   cfg,
		users: users,
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (*domain.User, string, error) {
	email = normalizeEmail(email)
	if email == "" {
		email = normalizeEmail(s.cfg.BootstrapAdminEmail)
	}
	if email == "" || password == "" {
		return nil, "", ErrInvalidCredentials
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err == nil {
		if !user.IsActive {
			return nil, "", ErrInvalidCredentials
		}
		if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
			return nil, "", ErrInvalidCredentials
		}
		if err := s.users.TouchLastLogin(ctx, user.ID); err != nil {
			return nil, "", err
		}
		token, err := s.GenerateToken(user)
		return user, token, err
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, "", err
	}

	user, err = s.bootstrapFirstAdmin(ctx, email, password)
	if err != nil {
		return nil, "", err
	}
	token, err := s.GenerateToken(user)
	return user, token, err
}

func (s *Service) bootstrapFirstAdmin(ctx context.Context, email, password string) (*domain.User, error) {
	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrInvalidCredentials
	}
	if email != normalizeEmail(s.cfg.BootstrapAdminEmail) || password != s.cfg.BootstrapAdminPassword {
		return nil, ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         s.cfg.BootstrapAdminName,
		IsActive:     true,
	}
	if user.Name == "" {
		user.Name = email
	}
	if err := s.users.CreateLocalUser(ctx, user, domain.RoleSuperAdmin); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) GenerateToken(user *domain.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        user.Role.Name,
		WorkspaceID: user.WorkspaceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWTIssuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.JWTTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Service) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithIssuer(s.cfg.JWTIssuer))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
