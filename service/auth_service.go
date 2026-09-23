package service

import (
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

func (s *AuthService) Register(request model.RegisterRequest) (model.User, error) {
	request.Username = strings.TrimSpace(request.Username)
	request.Password = strings.TrimSpace(request.Password)
	request.Role = strings.ToLower(strings.TrimSpace(request.Role))
	if request.Username == "" || len(request.Password) < 6 || (request.Role != "admin" && request.Role != "user") {
		return model.User{}, ErrInvalidCredentials
	}
	if _, err := s.repository.GetByUsername(request.Username); err == nil {
		return model.User{}, ErrUserAlreadyExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	return s.repository.Create(model.User{
		Username:  request.Username,
		Password:  string(hash),
		Role:      request.Role,
		CreatedAt: time.Now(),
	}), nil
}

func (s *AuthService) Login(request model.LoginRequest) (model.AuthResponse, error) {
	user, err := s.repository.GetByUsername(strings.TrimSpace(request.Username))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)) != nil {
		return model.AuthResponse{}, ErrInvalidCredentials
	}
	claims := jwt.MapClaims{
		"user_id": user.ID, "username": user.Username, "role": user.Role,
		"exp": time.Now().Add(24 * time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
	if err != nil {
		return model.AuthResponse{}, err
	}
	user.Password = ""
	return model.AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) GetUserByID(id int) (model.User, error) {
	user, err := s.repository.GetByID(id)
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
