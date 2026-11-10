package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"modul6/helper"

	"github.com/gofiber/fiber/v2"
)

func TestRequestLoggerUsesReturnedAppErrorStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	app := fiber.New(fiber.Config{ErrorHandler: helper.NewErrorHandler(logger)})
	app.Use(RequestID(), RequestLogger(logger))
	app.Get("/", func(c *fiber.Ctx) error {
		return helper.NotFound("not found")
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("response status = %d, want 404", response.StatusCode)
	}
	logOutput := logs.String()
	if !strings.Contains(logOutput, `"msg":"http_request"`) ||
		!strings.Contains(logOutput, `"status":404`) {
		t.Fatalf("access log did not record the returned 404: %s", logOutput)
	}
}
