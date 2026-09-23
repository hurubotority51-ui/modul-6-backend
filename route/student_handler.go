package route

import (
	"errors"
	"strconv"
	"strings"

	"modul6/middleware"
	"modul6/model"
	"modul6/repository"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

type StudentHandler struct {
	service *service.StudentService
}

func NewStudentHandler(service *service.StudentService) *StudentHandler {
	return &StudentHandler{service: service}
}

func (h *StudentHandler) Register(app *fiber.App) {
	students := app.Group("/api/v1/students", middleware.AuthRequired())
	students.Get("/", h.list)
	students.Get("/:id", h.get)
	students.Post("/", middleware.RequireRoles("admin"), h.create)
}

func (h *StudentHandler) list(c *fiber.Ctx) error {
	query := model.ListQuery{
		Page: c.QueryInt("page", 1), Limit: c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")), Sort: strings.ToLower(c.Query("sort", "id")),
		Order: strings.ToLower(c.Query("order", "asc")),
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.Limit < 1 || query.Limit > 100 {
		query.Limit = 10
	}
	if query.Sort != "id" && query.Sort != "nim" && query.Sort != "name" && query.Sort != "grade" && query.Sort != "created_at" {
		query.Sort = "id"
	}
	if query.Order != "asc" && query.Order != "desc" {
		query.Order = "asc"
	}
	if raw := strings.ToLower(c.Query("is_active")); raw == "true" || raw == "false" {
		value := raw == "true"
		query.IsActive = &value
	}
	data, meta := h.service.List(query)
	return c.JSON(fiber.Map{"success": true, "message": "daftar student berhasil diambil", "data": data, "meta": meta})
}

func (h *StudentHandler) get(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return failure(c, fiber.StatusBadRequest, "id harus berupa angka positif", nil)
	}
	student, err := h.service.GetByID(id)
	if errors.Is(err, repository.ErrStudentNotFound) {
		return failure(c, fiber.StatusNotFound, "student tidak ditemukan", nil)
	}
	return success(c, "student ditemukan", student)
}

func (h *StudentHandler) create(c *fiber.Ctx) error {
	var request model.CreateStudentRequest
	if err := c.BodyParser(&request); err != nil {
		return failure(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid", nil)
	}
	student, err := h.service.Create(request)
	if errors.Is(err, service.ErrDuplicateNIM) {
		return failure(c, fiber.StatusBadRequest, "Validation failed", map[string]string{"nim": "sudah digunakan"})
	}
	if errors.Is(err, service.ErrInvalidStudent) {
		return failure(c, fiber.StatusBadRequest, "Validation failed", map[string]string{"student": "nim, nama, dan grade harus valid"})
	}
	return created(c, "student berhasil dibuat", student)
}
