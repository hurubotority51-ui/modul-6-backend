package helper

import (
	"mime"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func RequireJSON(c *fiber.Ctx) error {
	contentType := strings.TrimSpace(c.Get(fiber.HeaderContentType))
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || (mediaType != fiber.MIMEApplicationJSON && !strings.HasSuffix(mediaType, "+json")) {
		return UnsupportedMediaType("Content-Type harus berupa application/json")
	}
	return nil
}
