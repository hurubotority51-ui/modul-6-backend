package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"modul6/helper"
	"modul6/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := create(); err != nil {
		log.Fatal(err)
	}
}

func create() error {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	username := strings.TrimSpace(os.Getenv("ADMIN_USERNAME"))
	password := os.Getenv("ADMIN_PASSWORD")
	if databaseURL == "" || username == "" || password == "" {
		return errors.New("DATABASE_URL, ADMIN_USERNAME, dan ADMIN_PASSWORD wajib diatur")
	}
	if fields := helper.ValidateStruct(model.RegisterRequest{
		Username: username, Password: password,
	}); fields != nil {
		return errors.New("username/password admin tidak memenuhi aturan")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO users (username, email, password, role)
		 VALUES ($1, $2, $3, 'admin')`,
		username, username+"@admin.local", string(hash))
	if err != nil {
		return err
	}
	log.Printf("created administrator %q", username)
	return nil
}
