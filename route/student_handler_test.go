package route

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"modul6/helper"
	"modul6/middleware"
	"modul6/model"
	"modul6/repository"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

func newStudentTestApp(t *testing.T) (*fiber.App, string) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := fiber.New(fiber.Config{ErrorHandler: helper.NewErrorHandler(logger)})
	app.Use(middleware.RequestID(), middleware.RequestLogger(logger))

	users := repository.NewInMemoryUserRepository()
	auth := service.NewAuthService(users)
	if _, err := auth.Register(model.RegisterRequest{
		Username: "admin", Password: "Securepass7", Role: "admin",
	}); err != nil {
		t.Fatalf("register test admin: %v", err)
	}
	login, err := auth.Login(model.LoginRequest{Username: "admin", Password: "Securepass7"})
	if err != nil {
		t.Fatalf("login test admin: %v", err)
	}

	students := repository.NewInMemoryStudentRepository()
	students.Create(model.Student{
		NIM: "12345678", Name: "Alice, Smith", Grade: 3.5, IsActive: true,
		CreatedAt: time.Date(2026, 9, 13, 14, 12, 40, 0, time.UTC),
	})
	NewStudentHandler(service.NewStudentService(students)).Register(app)
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})
	return app, login.Token
}

func TestStudentErrorResponsesAreCentralized(t *testing.T) {
	app, token := newStudentTestApp(t)
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/students/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.StatusCode)
	}
	var body helper.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Success || body.Code != helper.CodeUnauthorized || body.RequestID == "" {
		t.Fatalf("unexpected error body: %#v", body)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/students/", strings.NewReader("plain"))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "text/plain")
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusUnsupportedMediaType {
		t.Fatalf("unsupported media status = %d, want 415", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != helper.CodeUnsupportedMedia || body.RequestID == "" {
		t.Fatalf("unexpected media error body: %#v", body)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/students/?cursor=not-base64!!", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("invalid cursor status = %d, want 400", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if body.Code != helper.CodeBadRequest || body.RequestID == "" {
		t.Fatalf("unexpected cursor error body: %#v", body)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/students/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/xml")
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != fiber.StatusNotAcceptable {
		t.Fatalf("unacceptable media status = %d, want 406", response.StatusCode)
	}
	response.Body.Close()

	request = httptest.NewRequest(http.MethodGet, "/api/v1/students/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "*/*")
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK ||
		!strings.HasPrefix(response.Header.Get(fiber.HeaderContentType), helper.FormatJSON) {
		t.Fatalf("wildcard Accept response = %d %q, want JSON 200",
			response.StatusCode, response.Header.Get(fiber.HeaderContentType))
	}
	response.Body.Close()

	request = httptest.NewRequest(http.MethodGet, "/api/v1/not-found", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNotFound {
		t.Fatalf("unknown endpoint status = %d, want 404", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if body.Code != helper.CodeNotFound {
		t.Fatalf("unknown endpoint body = %#v", body)
	}
}

func TestStudentPatchValidationAndContentNegotiation(t *testing.T) {
	app, token := newStudentTestApp(t)
	request := authenticatedRequest(http.MethodPatch, "/api/v1/students/1", `{"name":""}`, token)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	var body helper.ErrorResponse
	if response.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("empty name status = %d, want 422", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if body.Code != helper.CodeValidation || body.Fields["name"] == "" {
		t.Fatalf("empty name validation response = %#v", body)
	}

	request = authenticatedRequest(http.MethodPatch, "/api/v1/students/1", `{}`, token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("empty patch status = %d, want 400", response.StatusCode)
	}

	request = authenticatedRequest(http.MethodPatch, "/api/v1/students/1", `{"is_active":false}`, token)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("patch with omitted name status = %d, want 200", response.StatusCode)
	}

	request = authenticatedRequest(http.MethodGet, "/api/v1/students/?limit=1", "", token)
	request.Header.Set("Accept", helper.FormatCSV)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	contentType := response.Header.Get(fiber.HeaderContentType)
	if !strings.HasPrefix(contentType, helper.FormatCSV) {
		t.Fatalf("Content-Type = %q, want text/csv", contentType)
	}
	if response.Header.Get(fiber.HeaderContentDisposition) != `attachment; filename="students.csv"` {
		t.Fatalf("unexpected Content-Disposition: %q", response.Header.Get(fiber.HeaderContentDisposition))
	}
	csv, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(csv), `"Alice, Smith"`) {
		t.Fatalf("CSV did not contain quoted student data: %s", csv)
	}
}

func authenticatedRequest(method, target, body, token string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", fiber.MIMEApplicationJSON)
	return request
}
