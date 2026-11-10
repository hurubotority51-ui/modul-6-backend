package repository

import (
	"context"
	"errors"
	"fmt"

	"modul6/model"

	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound = errors.New("user tidak ditemukan")
	ErrDuplicate    = errors.New("data sudah ada")
)

type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	GetByUsername(ctx context.Context, username string) (model.User, error)
	GetByID(ctx context.Context, id int) (model.User, error)
	HasPermission(ctx context.Context, role, permission string) (bool, error)
}

type userPostgresRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &userPostgresRepository{db: db}
}

func (r *userPostgresRepository) Create(ctx context.Context, user model.User) (model.User, error) {
	if user.Email == "" {
		user.Email = user.Username + "@local.invalid"
	}
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		user.Username, user.Email, user.Password, user.Role,
	).Scan(&user.ID, &user.CreatedAt)
	if pgErrorCode(err) == codeUniqueViolation {
		return model.User{}, ErrDuplicate
	}
	if err != nil {
		return model.User{}, fmt.Errorf("creating user: %w", err)
	}
	return user, nil
}

func (r *userPostgresRepository) GetByUsername(ctx context.Context, username string) (model.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx,
		`SELECT id, username, email, password, role, created_at
		 FROM users WHERE username = $1`, username))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("getting user by username: %w", err)
	}
	return user, nil
}

func (r *userPostgresRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	user, err := scanUser(r.db.QueryRow(ctx,
		`SELECT id, username, email, password, role, created_at
		 FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("getting user by id: %w", err)
	}
	return user, nil
}

func (r *userPostgresRepository) HasPermission(ctx context.Context, role, permission string) (bool, error) {
	var allowed bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM role_permissions
			WHERE role_name = $1 AND permission_name = $2
		)`, role, permission,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("checking permission: %w", err)
	}
	return allowed, nil
}

func scanUser(row pgx.Row) (model.User, error) {
	var user model.User
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role, &user.CreatedAt)
	return user, err
}
