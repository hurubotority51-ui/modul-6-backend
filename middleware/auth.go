package middleware

import (
	"context"
	"strings"
	"time"

	"modul6/helper"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, role, permission string) (bool, error)
}

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRole     = "role"
)

func AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		parts := strings.SplitN(c.Get("Authorization"), " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return helper.Unauthorized("token tidak valid")
		}
		claims, err := service.ValidateToken(parts[1])
		if err != nil {
			return helper.Unauthorized("token tidak valid")
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
		return helper.Forbidden("akses ditolak untuk role " + role)
	}
}

func RequirePermission(checker PermissionChecker, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals(ContextRole).(string)
		ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
		defer cancel()
		allowed, err := checker.HasPermission(ctx, role, permission)
		if err != nil {
			return helper.Internal(err)
		}
		if !allowed {
			return helper.Forbidden("tidak memiliki permission " + permission)
		}
		return c.Next()
	}
}
