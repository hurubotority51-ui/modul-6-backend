package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"modul6/model"
	"modul6/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("username atau password tidak valid")
var ErrUserAlreadyExists = errors.New("username sudah digunakan")

var jwtSecret = []byte(func() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "modul6-secret-key"
}())

type AuthService struct {
	repository repository.UserRepository
}

func NewAuthService(repository repository.UserRepository) *AuthService {
	return &AuthService{repository: repository}
}

func (s *AuthService) Register(ctx context.Context, request model.RegisterRequest) (model.User, error) {
	request.Username = strings.TrimSpace(request.Username)
	request.Role = "user"
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	user, err := s.repository.Create(ctx, model.User{
		Username:  request.Username,
		Password:  string(hash),
		Role:      request.Role,
		CreatedAt: time.Now(),
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return model.User{}, ErrUserAlreadyExists
	}
	return user, err
}

func (s *AuthService) Login(ctx context.Context, request model.LoginRequest) (model.AuthResponse, error) {
	user, err := s.repository.GetByUsername(ctx, strings.TrimSpace(request.Username))
	if errors.Is(err, repository.ErrUserNotFound) ||
		(err == nil && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil) {
		return model.AuthResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return model.AuthResponse{}, err
	}
	token, err := GenerateAccessToken(user)
	if err != nil {
		return model.AuthResponse{}, err
	}
	user.Password = ""
	return model.AuthResponse{Token: token, User: user}, nil
}

func GenerateAccessToken(user model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID, "username": user.Username, "role": user.Role,
		"exp": time.Now().Add(24 * time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) GetUserByID(ctx context.Context, id int) (model.User, error) {
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}
	user.Password = ""
	return user, nil
}

func ValidateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode token tidak valid")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}
