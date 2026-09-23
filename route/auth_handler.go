package route

import (
	"errors"

	"modul6/middleware"
	"modul6/model"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct{ service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

func (h *AuthHandler) Register(app *fiber.App) {
	auth := app.Group("/api/v1/auth")
	auth.Post("/register", h.register)
	auth.Post("/login", h.login)
}

func (h *AuthHandler) register(c *fiber.Ctx) error {
	var request model.RegisterRequest
	if err := c.BodyParser(&request); err != nil {
		return failure(c, 400, "body harus berupa JSON yang valid")
	}
	user, err := h.service.Register(request)
	if errors.Is(err, service.ErrUserAlreadyExists) {
		return failure(c, 400, "username sudah digunakan")
	}
	if errors.Is(err, service.ErrInvalidCredentials) {
		return failure(c, 400, "username, password, atau role tidak valid")
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "message": "register berhasil", "data": user})
}

func (h *AuthHandler) login(c *fiber.Ctx) error {
	var request model.LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return failure(c, 400, "body harus berupa JSON yang valid")
	}
	response, err := h.service.Login(request)
	if err != nil {
		return failure(c, 401, "username atau password salah")
	}
	return c.JSON(fiber.Map{"success": true, "message": "login berhasil", "data": response})
}

func Profile(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "profile berhasil diakses", "data": fiber.Map{
		"username": c.Locals(middleware.ContextUsername), "role": c.Locals(middleware.ContextRole),
	}})
}

func UserArea(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "user area berhasil diakses"})
}

func AdminArea(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "admin area berhasil diakses"})
}

func failure(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"success": false, "message": message})
}
