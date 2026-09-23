package route

import "github.com/gofiber/fiber/v2"

type webResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func success(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(webResponse{Success: true, Message: message, Data: data})
}

func created(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusCreated).JSON(webResponse{Success: true, Message: message, Data: data})
}

func failure(c *fiber.Ctx, status int, message string, errors any) error {
	return c.Status(status).JSON(webResponse{Success: false, Message: message, Errors: errors})
}
