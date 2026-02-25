package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrUserExists         = errors.New("usuário já existe")
)

type Service struct {
	repo        *Repository
	jwtSecret   []byte
	tokenExpiry time.Duration
}

func NewService(repo *Repository, jwtSecret string, tokenExpiryHours int) *Service {
	return &Service{
		repo:        repo,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: time.Duration(tokenExpiryHours) * time.Hour,
	}
}

func (s *Service) CreateAdmin(ctx context.Context, username, password string) error {
	existing, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrUserExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := &Admin{
		Username: username,
		Password: string(hashed),
	}
	return s.repo.Insert(ctx, admin)
}

func (s *Service) Login(ctx context.Context, username, password string) (string, error) {
	admin, err := s.repo.FindByUsername(ctx, username)
	if err != nil || admin == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	claims := jwt.MapClaims{
		"sub":   admin.ID.Hex(),
		"exp":   time.Now().Add(s.tokenExpiry).Unix(),
		"admin": true,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("método de assinatura inválido")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !claims["admin"].(bool) {
		return "", ErrInvalidCredentials
	}

	return claims["sub"].(string), nil
}
