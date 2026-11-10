package route

import (
	"errors"
	"strings"

	"modul6/helper"
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
	if err := helper.RequireJSON(c); err != nil {
		return err
	}
	var request model.RegisterRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	request.Username = strings.TrimSpace(request.Username)
	request.Role = strings.ToLower(strings.TrimSpace(request.Role))
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	user, err := h.service.Register(request)
	if errors.Is(err, service.ErrUserAlreadyExists) {
		return helper.Conflict("username sudah digunakan")
	}
	if errors.Is(err, service.ErrInvalidCredentials) {
		return helper.BadRequest("data registrasi tidak valid")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return created(c, "user berhasil dibuat", user)
}

func (h *AuthHandler) login(c *fiber.Ctx) error {
	if err := helper.RequireJSON(c); err != nil {
		return err
	}
	var request model.LoginRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	response, err := h.service.Login(request)
	if errors.Is(err, service.ErrInvalidCredentials) {
		return helper.Unauthorized("username atau password salah")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return success(c, "login berhasil", response)
}

func (h *AuthHandler) me(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return helper.Unauthorized("token tidak valid")
	}
	user, err := h.service.GetUserByID(userID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return helper.NotFound("user tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
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
