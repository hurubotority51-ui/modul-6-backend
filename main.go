package main

import (
	"log/slog"
	"os"

	"modul6/helper"
	"modul6/middleware"
	"modul6/repository"
	"modul6/route"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	app := fiber.New(fiber.Config{ErrorHandler: helper.NewErrorHandler(logger)})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(logger))

	userRepository := repository.NewInMemoryUserRepository()
	authService := service.NewAuthService(userRepository)
	authHandler := route.NewAuthHandler(authService)
	authHandler.Register(app)

	studentRepository := repository.NewInMemoryStudentRepository()
	studentService := service.NewStudentService(studentRepository)
	studentHandler := route.NewStudentHandler(studentService)
	studentHandler.Register(app)

	protected := app.Group("/api/v1", middleware.AuthRequired())
	protected.Get("/profile", route.Profile)
	protected.Get("/user-area", middleware.RequireRoles("user", "admin"), route.UserArea)
	protected.Get("/admin-area", middleware.RequireRoles("admin"), route.AdminArea)
	app.Get("/profile", middleware.AuthRequired(), route.Profile)
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	logger.Info("server_started", slog.String("address", "http://localhost:3000"))
	if err := app.Listen(":3000"); err != nil {
		logger.Error("server_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
