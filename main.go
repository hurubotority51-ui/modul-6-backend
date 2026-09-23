package main

import (
	"log"

	"modul6/middleware"
	"modul6/repository"
	"modul6/route"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

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

	log.Println("server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
