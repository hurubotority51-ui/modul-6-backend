package route

import (
	"errors"

	"modul6/middleware"
	"modul6/model"
	"modul6/repository"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct{ service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/api/v1/auth")
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
	auth.Get("/me", middleware.AuthRequired(), h.me)
}

func (h *AuthHandler) register(c *fiber.Ctx) error {
	var request model.RegisterRequest
	if err := c.BodyParser(&request); err != nil {
		return failure(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid", nil)
	}
	user, err := h.service.Register(request)
	if errors.Is(err, service.ErrUserAlreadyExists) {
		return failure(c, fiber.StatusBadRequest, "Validation failed", map[string]string{"username": "sudah digunakan"})
	}
	if errors.Is(err, service.ErrInvalidCredentials) {
		return failure(c, fiber.StatusBadRequest, "Validation failed", map[string]string{"username": "minimal 1 karakter, password minimal 6 karakter, role user atau admin"})
	}
	return created(c, "user berhasil dibuat", user)
}

func (h *AuthHandler) login(c *fiber.Ctx) error {
	var request model.LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return failure(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid", nil)
	}
	response, err := h.service.Login(request)
	if err != nil {
		return failure(c, fiber.StatusUnauthorized, "username atau password salah", nil)
	}
	return success(c, "login berhasil", response)
}

func (h *AuthHandler) me(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return failure(c, fiber.StatusUnauthorized, "token tidak valid", nil)
	}
	user, err := h.service.GetUserByID(userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return failure(c, fiber.StatusNotFound, "user tidak ditemukan", nil)
	}
	if err != nil {
		return failure(c, fiber.StatusInternalServerError, "gagal mengambil user", nil)
	}
	return success(c, "user berhasil diambil", user)
}

func Profile(c *fiber.Ctx) error {
	return success(c, "profile berhasil diakses", fiber.Map{
		"username": c.Locals(middleware.ContextUsername), "role": c.Locals(middleware.ContextRole),
	})
}

func UserArea(c *fiber.Ctx) error {
	return success(c, "user area berhasil diakses", nil)
}

func AdminArea(c *fiber.Ctx) error {
	return success(c, "admin area berhasil diakses", nil)
}
