package middleware

import (
	"strconv"
	"strings"

	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRole     = "role"
)

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		parts := strings.SplitN(c.Get("Authorization"), " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "token tidak valid"})
		}
		claims, err := service.ValidateToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "token tidak valid"})
		}
		if userID, ok := claims[ContextUserID].(float64); ok {
			c.Locals(ContextUserID, int(userID))
		}
		if username, ok := claims[ContextUsername].(string); ok {
			c.Locals(ContextUsername, username)
		}
		if role, ok := claims[ContextRole].(string); ok {
			c.Locals(ContextRole, role)
		}
		return c.Next()
	}
}

func RequireRoles(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(ContextRole).(string)
		for _, allowed := range roles {
			if role == allowed {
				return c.Next()
			}
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false, "message": "akses ditolak untuk role " + strconv.Quote(role),
		})
	}
}
