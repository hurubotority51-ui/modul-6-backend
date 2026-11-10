package route

import (
	"errors"
	"strconv"
	"strings"

	"modul6/helper"
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
	students.Patch("/:id", middleware.RequireRoles("admin"), h.patch)
}

func (h *StudentHandler) list(c *fiber.Ctx) error {
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 {
			return helper.BadRequest("limit harus berupa angka positif")
		}
		if limit > 100 {
			limit = 100
		}
	}
	query := model.CursorQuery{
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
	}
	if raw := strings.TrimSpace(c.Query("is_active")); raw != "" {
		value, parseErr := strconv.ParseBool(raw)
		if parseErr != nil {
			return helper.BadRequest("is_active harus bernilai true atau false")
		}
		query.IsActive = &value
	}
	if raw := c.Query("cursor"); raw != "" {
		cursor, decodeErr := helper.DecodeCursor(raw)
		if decodeErr != nil {
			return helper.BadRequest("cursor tidak valid")
		}
		query.After = &cursor
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	data, meta, err := h.service.List(ctx, query)
	if err != nil {
		return helper.Internal(err)
	}
	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, data)
	}
	return c.JSON(fiber.Map{
		"success": true, "message": "daftar student berhasil diambil",
		"data": data, "meta": meta,
	})
}

func (h *StudentHandler) get(c *fiber.Ctx) error {
	id, err := parseStudentID(c)
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	student, err := h.service.GetByID(ctx, id)
	if errors.Is(err, repository.ErrStudentNotFound) {
		return helper.NotFound("student tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return success(c, "student ditemukan", student)
}

func (h *StudentHandler) create(c *fiber.Ctx) error {
	if err := helper.RequireJSON(c); err != nil {
		return err
	}
	var request model.CreateStudentRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	request.NIM = strings.TrimSpace(request.NIM)
	request.Name = strings.TrimSpace(request.Name)
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	student, err := h.service.Create(ctx, request)
	if errors.Is(err, service.ErrDuplicateNIM) {
		return helper.Conflict("nim sudah digunakan")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return created(c, "student berhasil dibuat", student)
}

func (h *StudentHandler) patch(c *fiber.Ctx) error {
	if err := helper.RequireJSON(c); err != nil {
		return err
	}
	id, err := parseStudentID(c)
	if err != nil {
		return err
	}
	var request model.PatchStudentRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if request.IsEmpty() {
		return helper.BadRequest("setidaknya satu field harus dikirim")
	}
	if request.NIM != nil {
		value := strings.TrimSpace(*request.NIM)
		request.NIM = &value
	}
	if request.Name != nil {
		value := strings.TrimSpace(*request.Name)
		request.Name = &value
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	student, err := h.service.Patch(ctx, id, request)
	switch {
	case errors.Is(err, repository.ErrStudentNotFound):
		return helper.NotFound("student tidak ditemukan")
	case errors.Is(err, service.ErrDuplicateNIM):
		return helper.Conflict("nim sudah digunakan")
	case err != nil:
		return helper.Internal(err)
	default:
		return success(c, "student berhasil diperbarui", student)
	}
}

func parseStudentID(c *fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, helper.BadRequest("id harus berupa angka positif")
	}
	return id, nil
}
