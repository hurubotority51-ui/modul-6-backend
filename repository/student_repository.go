package repository

import (
	"context"
	"errors"
	"fmt"

	"modul6/model"

	"github.com/jackc/pgx/v5"
)

var ErrStudentNotFound = errors.New("student tidak ditemukan")

type StudentRepository interface {
	List(ctx context.Context, query model.CursorQuery) ([]model.Student, error)
	GetByID(ctx context.Context, id int) (model.Student, error)
	Create(ctx context.Context, student model.Student) (model.Student, error)
	Update(ctx context.Context, student model.Student) error
}

type studentPostgresRepository struct {
	db DBTX
}

func NewStudentRepository(db DBTX) StudentRepository {
	return &studentPostgresRepository{db: db}
}

const studentColumns = `id, nim, name, grade, is_active, created_at`

func (r *studentPostgresRepository) List(ctx context.Context, query model.CursorQuery) ([]model.Student, error) {
	args := make([]any, 0, 5)
	where := " WHERE TRUE"
	if query.Search != "" {
		args = append(args, "%"+query.Search+"%")
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args))
	}
	if query.IsActive != nil {
		args = append(args, *query.IsActive)
		where += fmt.Sprintf(" AND is_active = $%d", len(args))
	}
	if query.After != nil {
		args = append(args, query.After.CreatedAt, query.After.ID)
		where += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}
	args = append(args, query.Limit+1)
	sql := fmt.Sprintf(
		"SELECT %s FROM students%s ORDER BY created_at DESC, id DESC LIMIT $%d",
		studentColumns, where, len(args))
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("listing students: %w", err)
	}
	defer rows.Close()
	students := make([]model.Student, 0, query.Limit+1)
	for rows.Next() {
		var student model.Student
		if err := rows.Scan(
			&student.ID, &student.NIM, &student.Name, &student.Grade,
			&student.IsActive, &student.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning student: %w", err)
		}
		students = append(students, student)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading students: %w", err)
	}
	return students, nil
}

func (r *studentPostgresRepository) GetByID(ctx context.Context, id int) (model.Student, error) {
	var student model.Student
	err := r.db.QueryRow(ctx,
		"SELECT "+studentColumns+" FROM students WHERE id = $1", id,
	).Scan(&student.ID, &student.NIM, &student.Name, &student.Grade, &student.IsActive, &student.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Student{}, ErrStudentNotFound
	}
	if err != nil {
		return model.Student{}, fmt.Errorf("getting student: %w", err)
	}
	return student, nil
}

func (r *studentPostgresRepository) Create(ctx context.Context, student model.Student) (model.Student, error) {
	err := r.db.QueryRow(ctx,
		`INSERT INTO students (nim, name, grade, is_active)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		student.NIM, student.Name, student.Grade, student.IsActive,
	).Scan(&student.ID, &student.CreatedAt)
	if pgErrorCode(err) == codeUniqueViolation {
		return model.Student{}, ErrDuplicate
	}
	if err != nil {
		return model.Student{}, fmt.Errorf("creating student: %w", err)
	}
	return student, nil
}

func (r *studentPostgresRepository) Update(ctx context.Context, student model.Student) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE students SET nim = $1, name = $2, grade = $3, is_active = $4
		 WHERE id = $5`,
		student.NIM, student.Name, student.Grade, student.IsActive, student.ID,
	)
	if pgErrorCode(err) == codeUniqueViolation {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("updating student: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStudentNotFound
	}
	return nil
}
