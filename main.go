package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"modul6/helper"
	"modul6/middleware"
	"modul6/repository"
	"modul6/route"
	"modul6/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL wajib diatur; jalankan go run ./cmd/migrate terlebih dahulu")
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	maxConnections := int32(10)
	if raw := os.Getenv("DB_MAX_CONNS"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return errors.New("DB_MAX_CONNS harus berupa angka positif")
		}
		maxConnections = int32(value)
	}
	poolConfig.MaxConns = maxConnections
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	defer pool.Close()
	startupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(startupCtx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	users := repository.NewUserRepository(pool)
	students := repository.NewStudentRepository(pool)
	courses := repository.NewCourseRepository(pool)
	supervisors := repository.NewSupervisorRepository(pool)
	txManager := repository.NewTxManager(pool)

	authHandler := route.NewAuthHandler(service.NewAuthService(users))
	studentHandler := route.NewStudentHandler(service.NewStudentService(students))
	courseHandler := route.NewCourseHandler(service.NewCourseService(courses, txManager))
	supervisorHandler := route.NewSupervisorHandler(service.NewSupervisorService(supervisors, txManager))

	app := fiber.New(fiber.Config{ErrorHandler: helper.NewErrorHandler(logger)})
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(logger))
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		ctx, cancel := helper.RequestContext(c)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Internal(err)
		}
		return helper.Success(c, fiber.StatusOK, "layanan sehat", fiber.Map{"status": "ok"})
	})

	authHandler.Register(app)
	studentHandler.Register(app)
	courseHandler.Register(app, users)
	supervisorHandler.Register(app, users)

	protected := app.Group("/api/v1", middleware.AuthRequired())
	protected.Get("/profile", route.Profile)
	protected.Get("/user-area", middleware.RequireRoles("user", "admin"), route.UserArea)
	protected.Get("/admin-area", middleware.RequireRoles("admin"), route.AdminArea)
	app.Get("/profile", middleware.AuthRequired(), route.Profile)
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})

	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":3000"
	}
	logger.Info("server_started", slog.String("address", address))
	if err := app.Listen(address); err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}
