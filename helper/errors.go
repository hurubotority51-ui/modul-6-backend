package helper

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

const (
	CodeValidation       = "VALIDATION_ERROR"
	CodeBadRequest       = "BAD_REQUEST"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeForbidden        = "FORBIDDEN"
	CodeNotFound         = "NOT_FOUND"
	CodeConflict         = "CONFLICT"
	CodeUnsupportedMedia = "UNSUPPORTED_MEDIA_TYPE"
	CodeNotAcceptable    = "NOT_ACCEPTABLE"
	CodeTooManyRequests  = "TOO_MANY_REQUESTS"
	CodeInternal         = "INTERNAL_ERROR"
	CodeUnavailable      = "SERVICE_UNAVAILABLE"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.cause }

func BadRequest(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Code: CodeBadRequest, Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: fiber.StatusUnauthorized, Code: CodeUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: fiber.StatusForbidden, Code: CodeForbidden, Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: fiber.StatusNotFound, Code: CodeNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: fiber.StatusConflict, Code: CodeConflict, Message: message}
}

func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func Validation(fields map[string]string) *AppError {
	return &AppError{
		Status: fiber.StatusUnprocessableEntity, Code: CodeValidation,
		Message: "validasi gagal", Fields: fields,
	}
}

func UnsupportedMediaType(message string) *AppError {
	return &AppError{
		Status: fiber.StatusUnsupportedMediaType, Code: CodeUnsupportedMedia, Message: message,
	}
}

func NotAcceptable(message string) *AppError {
	return &AppError{
		Status: fiber.StatusNotAcceptable, Code: CodeNotAcceptable, Message: message,
	}
}

func TooManyRequests(message string) *AppError {
	return &AppError{
		Status: fiber.StatusTooManyRequests, Code: CodeTooManyRequests, Message: message,
	}
}

func ServiceUnavailable(cause error) *AppError {
	return &AppError{
		Status: fiber.StatusServiceUnavailable, Code: CodeUnavailable,
		Message: "layanan sementara tidak tersedia", cause: cause,
	}
}

func Internal(cause error) *AppError {
	return &AppError{
		Status: fiber.StatusInternalServerError, Code: CodeInternal,
		Message: "terjadi kesalahan pada server", cause: cause,
	}
}

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(fiber.Map{
		"success": true, "message": message, "data": data,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set(fiber.HeaderLocation, location)
	return Success(c, fiber.StatusCreated, message, data)
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id"`
}

func RequestID(c *fiber.Ctx) string {
	id, _ := c.Locals("request_id").(string)
	return id
}

func NewErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		requestID := RequestID(c)
		var appErr *AppError
		if !errors.As(err, &appErr) {
			var fiberErr *fiber.Error
			if errors.Is(err, fiber.ErrRequestEntityTooLarge) {
				appErr = &AppError{
					Status:  fiber.StatusRequestEntityTooLarge,
					Code:    "PAYLOAD_TOO_LARGE",
					Message: "ukuran body melebihi batas yang diizinkan",
				}
			} else if errors.As(err, &fiberErr) {
				appErr = appErrorFromFiber(fiberErr)
			} else {
				appErr = Internal(err)
			}
		}

		attrs := []any{
			slog.String("request_id", requestID),
			slog.String("path", c.Path()),
			slog.String("code", appErr.Code),
			slog.Int("status", appErr.Status),
		}
		if appErr.Status >= fiber.StatusInternalServerError {
			logger.Error("request_failed", append(attrs, slog.String("error", appErr.Error()))...)
		} else {
			logger.Warn("request_rejected", attrs...)
		}

		return c.Status(appErr.Status).JSON(ErrorResponse{
			Success: false, Code: appErr.Code, Message: appErr.Message,
			Fields: appErr.Fields, RequestID: requestID,
		})
	}
}

func appErrorFromFiber(err *fiber.Error) *AppError {
	switch err.Code {
	case fiber.StatusNotFound:
		return NotFound("endpoint tidak ditemukan")
	case fiber.StatusUnauthorized:
		return Unauthorized("belum terautentikasi")
	case fiber.StatusForbidden:
		return Forbidden("akses ditolak")
	case fiber.StatusUnsupportedMediaType:
		return UnsupportedMediaType("format body tidak didukung")
	case fiber.StatusNotAcceptable:
		return NotAcceptable("format response tidak dapat diterima")
	case fiber.StatusRequestEntityTooLarge:
		return &AppError{
			Status: fiber.StatusRequestEntityTooLarge, Code: "PAYLOAD_TOO_LARGE",
			Message: "ukuran body melebihi batas yang diizinkan",
		}
	case fiber.StatusTooManyRequests:
		return TooManyRequests("terlalu banyak permintaan")
	case fiber.StatusServiceUnavailable:
		return ServiceUnavailable(err)
	case fiber.StatusBadRequest:
		return BadRequest("permintaan tidak valid")
	default:
		if err.Code >= fiber.StatusInternalServerError {
			return Internal(err)
		}
		if err.Code >= fiber.StatusBadRequest {
			return &AppError{
				Status: err.Code, Code: "HTTP_ERROR",
				Message: "permintaan tidak dapat diproses",
			}
		}
		return Internal(err)
	}
}
