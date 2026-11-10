package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"modul6/helper"

	"github.com/gofiber/fiber/v2"
)

var fallbackRequestSequence atomic.Uint64

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := newRequestID()
		if err != nil {
			sequence := fallbackRequestSequence.Add(1)
			id = fmt.Sprintf("%d-%d", time.Now().UnixNano(), sequence)
			c.Locals("request_id", id)
			c.Set("X-Request-ID", id)
			return helper.Internal(err)
		}
		c.Locals("request_id", id)
		c.Set("X-Request-ID", id)
		return c.Next()
	}
}

func newRequestID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] +
		"-" + encoded[16:20] + "-" + encoded[20:], nil
}

func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			var appErr *helper.AppError
			var fiberErr *fiber.Error
			switch {
			case errors.As(err, &appErr):
				status = appErr.Status
			case errors.As(err, &fiberErr):
				status = fiberErr.Code
			default:
				status = fiber.StatusInternalServerError
			}
		}
		logger.Info("http_request",
			slog.String("request_id", helper.RequestID(c)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		)
		return err
	}
}
