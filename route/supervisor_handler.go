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

type SupervisorHandler struct {
	service *service.SupervisorService
}

func NewSupervisorHandler(supervisorService *service.SupervisorService) *SupervisorHandler {
	return &SupervisorHandler{service: supervisorService}
}

func (h *SupervisorHandler) Register(app *fiber.App, permissions middleware.PermissionChecker) {
	api := app.Group("/api/v1", middleware.AuthRequired())
	supervisors := api.Group("/supervisors")
	supervisors.Get("/", h.list)
	supervisors.Post("/:id/assign", helper.RequireJSON,
		middleware.RequirePermission(permissions, "supervision:manage"), h.assign)
	supervisors.Patch("/:id", helper.RequireJSON,
		middleware.RequirePermission(permissions, "supervision:manage"), h.updateQuota)
	api.Post("/supervisions/switch", helper.RequireJSON,
		middleware.RequirePermission(permissions, "supervision:manage"), h.switchSupervisor)
}

func (h *SupervisorHandler) list(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	supervisors, err := h.service.List(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar pembimbing berhasil diambil", supervisors)
}

func (h *SupervisorHandler) assign(c *fiber.Ctx) error {
	supervisorID, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	var request model.AssignSupervisionRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	supervisor, err := h.service.Assign(ctx, request.StudentID, supervisorID)
	if err != nil {
		return supervisionError(err)
	}
	return helper.Created(c, "pembimbing berhasil ditetapkan", supervisor,
		"/api/v1/supervisors/"+strconv.Itoa(supervisorID))
}

func (h *SupervisorHandler) switchSupervisor(c *fiber.Ctx) error {
	var request model.SwitchSupervisionRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	if request.FromSupervisorID == request.ToSupervisorID {
		return helper.Validation(map[string]string{
			"to_supervisor_id": "tidak boleh sama dengan from_supervisor_id",
		})
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	supervisor, err := h.service.Switch(
		ctx, request.StudentID, request.FromSupervisorID, request.ToSupervisorID)
	if err != nil {
		return supervisionError(err)
	}
	return helper.Success(c, fiber.StatusOK, "pembimbing berhasil diganti", supervisor)
}

func (h *SupervisorHandler) updateQuota(c *fiber.Ctx) error {
	supervisorID, err := parseID(c.Params("id"))
	if err != nil {
		return err
	}
	var request model.UpdateSupervisorQuotaRequest
	if err := c.BodyParser(&request); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if fields := helper.ValidateStruct(request); fields != nil {
		return helper.Validation(fields)
	}
	ctx, cancel := helper.RequestContext(c)
	defer cancel()
	supervisor, err := h.service.UpdateQuota(ctx, supervisorID, request)
	if err != nil {
		return supervisionError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota pembimbing berhasil diubah", supervisor)
}

func supervisionError(err error) error {
	switch {
	case errors.Is(err, repository.ErrStudentNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, service.ErrSupervisorNotFound):
		return helper.NotFound("pembimbing tidak ditemukan")
	case errors.Is(err, service.ErrAlreadySupervised):
		return helper.NewError(fiber.StatusConflict, "ALREADY_SUPERVISED", "mahasiswa sudah memiliki pembimbing")
	case errors.Is(err, service.ErrSupervisorFull):
		return helper.NewError(fiber.StatusConflict, "SUPERVISOR_FULL", "kuota pembimbing sudah penuh")
	case errors.Is(err, service.ErrNotSupervised):
		return helper.NewError(fiber.StatusConflict, "NOT_SUPERVISED", "mahasiswa tidak memiliki pembimbing tersebut")
	case errors.Is(err, service.ErrSupervisorVersion):
		return helper.NewError(fiber.StatusConflict, "VERSION_CONFLICT",
			"data pembimbing sudah diubah pihak lain, muat ulang lalu coba lagi")
	case errors.Is(err, service.ErrSupervisorQuota):
		return helper.NewError(fiber.StatusConflict, "QUOTA_BELOW_SUPERVISED",
			"kuota tidak boleh lebih kecil dari jumlah mahasiswa terbimbing")
	case errors.Is(err, service.ErrSameSupervisor):
		return helper.Validation(map[string]string{
			"to_supervisor_id": "tidak boleh sama dengan from_supervisor_id",
		})
	default:
		return helper.Internal(err)
	}
}
