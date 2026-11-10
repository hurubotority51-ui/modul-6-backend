package route

import (
	"errors"
	"strconv"

	"modul6/helper"
	"modul6/middleware"
	"modul6/model"
	"modul6/repository"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

type CourseHandler struct {
	service *service.CourseService
}

func NewCourseHandler(courseService *service.CourseService) *CourseHandler {
	return &CourseHandler{service: courseService}
}

func (h *CourseHandler) Register(app *fiber.App, permissions middleware.PermissionChecker) {
	api := app.Group("/api/v1", middleware.AuthRequired())
	courses := api.Group("/courses")
	courses.Get("/", h.list)
	courses.Get("/mine", h.mine)
	courses.Get("/:id", h.get)
	courses.Post("/:id/enroll", h.enroll)
	courses.Delete("/:id/enroll", h.drop)
	courses.Patch("/:id", helper.RequireJSON, middleware.RequirePermission(permissions, "course:manage"), h.updateQuota)

	api.Post("/enrollments/swap", helper.RequireJSON, h.swap)
}

func (h *CourseHandler) list(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	courses, err := h.service.List(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar kelas berhasil diambil", courses)
}

func (h *CourseHandler) mine(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return helper.Unauthorized("token tidak valid")
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	courses, total, err := h.service.Mine(ctx, userID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "KRS berhasil diambil", fiber.Map{
		"courses": courses, "total_credits": total, "max_credits": service.MaxCreditsPerSemester,
	})
}

func (h *CourseHandler) get(c *fiber.Ctx) error {
	id, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	course, err := h.service.Get(ctx, id)
	if err != nil {
		return courseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kelas ditemukan", course)
}

func (h *CourseHandler) enroll(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return helper.Unauthorized("token tidak valid")
	}
	courseID, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	course, err := h.service.Enroll(ctx, courseID, userID)
	if err != nil {
		return courseError(err)
	}
	return helper.Created(c, "berhasil mendaftar kelas", course,
		"/api/v1/courses/"+strconv.Itoa(courseID))
}

func (h *CourseHandler) drop(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return helper.Unauthorized("token tidak valid")
	}
	courseID, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	if err := h.service.Drop(ctx, courseID, userID); err != nil {
		return courseError(err)
	}
	return helper.NoContent(c)
}

func (h *CourseHandler) swap(c *fiber.Ctx) error {
	userID, ok := c.Locals(middleware.ContextUserID).(int)
	if !ok {
		return helper.Unauthorized("token tidak valid")
	}
	var request model.SwapRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	if request.FromCourseID == request.ToCourseID {
		return helper.Validation(map[string]string{
			"to_course_id": "tidak boleh sama dengan from_course_id",
		})
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	course, err := h.service.Swap(ctx, userID, request.FromCourseID, request.ToCourseID)
	if err != nil {
		return courseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "berhasil pindah kelas", course)
}

func (h *CourseHandler) updateQuota(c *fiber.Ctx) error {
	id, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	var request model.UpdateQuotaRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	course, err := h.service.UpdateQuota(ctx, id, request)
	if err != nil {
		return courseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota berhasil diubah", course)
}

func courseError(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	case errors.Is(err, repository.ErrUserNotFound):
		return helper.Unauthorized("akun tidak ditemukan")
	case errors.Is(err, repository.ErrCourseFull):
		return helper.NewError(fiber.StatusConflict, "COURSE_FULL", "kuota kelas sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.NewError(fiber.StatusConflict, "ALREADY_ENROLLED", "Anda sudah terdaftar di kelas ini")
	case errors.Is(err, repository.ErrNotEnrolled):
		return helper.NewError(fiber.StatusConflict, "NOT_ENROLLED", "Anda tidak terdaftar di kelas tersebut")
	case errors.Is(err, repository.ErrCreditLimit):
		return helper.NewError(fiber.StatusConflict, "CREDIT_LIMIT_EXCEEDED",
			"batas "+strconv.Itoa(service.MaxCreditsPerSemester)+" SKS terlampaui")
	case errors.Is(err, repository.ErrVersionConflict):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT",
			"data kelas sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowFilled):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_ENROLLED",
			"kuota tidak boleh lebih kecil dari jumlah peserta saat ini")
	default:
		return helper.Internal(err)
	}
}

func parseID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		return 0, helper.BadRequest("id harus berupa angka positif")
	}
	return id, nil
}
