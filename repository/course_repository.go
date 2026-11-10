package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"modul6/model"

	"github.com/jackc/pgx/v5"
)

var (
	ErrNotFound         = errors.New("data tidak ditemukan")
	ErrCourseFull       = errors.New("kuota kelas sudah penuh")
	ErrNotEnrolled      = errors.New("tidak terdaftar di kelas ini")
	ErrVersionConflict  = errors.New("data sudah diubah pihak lain")
	ErrQuotaBelowFilled = errors.New("kuota lebih kecil dari jumlah peserta")
	ErrCreditLimit      = errors.New("batas SKS terlampaui")
)

type CourseRepository interface {
	WithTx(tx DBTX) CourseRepository
	FindAll(ctx context.Context) ([]model.Course, error)
	FindByID(ctx context.Context, id int) (model.Course, error)
	FindByStudent(ctx context.Context, userID int) ([]model.Course, error)
	LockStudent(ctx context.Context, userID int) error
	LockCourses(ctx context.Context, ids ...int) error
	SumCredits(ctx context.Context, userID int) (int, error)
	HasEnrollment(ctx context.Context, courseID, userID int) (bool, error)
	InsertEnrollment(ctx context.Context, courseID, userID int) error
	DeleteEnrollment(ctx context.Context, courseID, userID int) error
	IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Course, error)
}

const courseColumns = `id, code, name, credits, quota, enrolled_count, version, created_at`

type coursePostgresRepository struct {
	db DBTX
}

func NewCourseRepository(db DBTX) CourseRepository {
	return &coursePostgresRepository{db: db}
}

func (r *coursePostgresRepository) WithTx(tx DBTX) CourseRepository {
	return &coursePostgresRepository{db: tx}
}

func scanCourse(row pgx.Row) (model.Course, error) {
	var course model.Course
	err := row.Scan(
		&course.ID, &course.Code, &course.Name, &course.Credits, &course.Quota,
		&course.EnrolledCount, &course.Version, &course.CreatedAt,
	)
	return course, err
}

func (r *coursePostgresRepository) FindAll(ctx context.Context) ([]model.Course, error) {
	rows, err := r.db.Query(ctx, "SELECT "+courseColumns+" FROM courses ORDER BY code")
	if err != nil {
		return nil, fmt.Errorf("listing courses: %w", err)
	}
	defer rows.Close()
	courses := make([]model.Course, 0)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning course: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading courses: %w", err)
	}
	return courses, nil
}

func (r *coursePostgresRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx,
		"SELECT "+courseColumns+" FROM courses WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, ErrNotFound
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("getting course: %w", err)
	}
	return course, nil
}

func (r *coursePostgresRepository) FindByStudent(ctx context.Context, userID int) ([]model.Course, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.id, c.code, c.name, c.credits, c.quota, c.enrolled_count, c.version, c.created_at
		 FROM courses c
		 JOIN enrollments e ON e.course_id = c.id
		 WHERE e.user_id = $1
		 ORDER BY c.code`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing enrolled courses: %w", err)
	}
	defer rows.Close()
	courses := make([]model.Course, 0)
	for rows.Next() {
		course, err := scanCourse(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning enrolled course: %w", err)
		}
		courses = append(courses, course)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading enrolled courses: %w", err)
	}
	return courses, nil
}

func (r *coursePostgresRepository) LockStudent(ctx context.Context, userID int) error {
	var id int
	err := r.db.QueryRow(ctx, "SELECT id FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return fmt.Errorf("locking student account: %w", err)
	}
	return nil
}

func (r *coursePostgresRepository) LockCourses(ctx context.Context, ids ...int) error {
	ordered := append([]int(nil), ids...)
	sort.Ints(ordered)
	lastID := 0
	for _, id := range ordered {
		if id == lastID {
			continue
		}
		lastID = id
		var found int
		err := r.db.QueryRow(ctx, "SELECT id FROM courses WHERE id = $1 FOR UPDATE", id).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("locking course: %w", err)
		}
	}
	return nil
}

func (r *coursePostgresRepository) SumCredits(ctx context.Context, userID int) (int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.credits), 0)
		 FROM enrollments e JOIN courses c ON c.id = e.course_id
		 WHERE e.user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("summing enrolled credits: %w", err)
	}
	return total, nil
}

func (r *coursePostgresRepository) HasEnrollment(ctx context.Context, courseID, userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM enrollments WHERE course_id = $1 AND user_id = $2)",
		courseID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking enrollment: %w", err)
	}
	return exists, nil
}

func (r *coursePostgresRepository) InsertEnrollment(ctx context.Context, courseID, userID int) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)", courseID, userID)
	if pgErrorCode(err) == codeUniqueViolation {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("inserting enrollment: %w", err)
	}
	return nil
}

func (r *coursePostgresRepository) DeleteEnrollment(ctx context.Context, courseID, userID int) error {
	tag, err := r.db.Exec(ctx,
		"DELETE FROM enrollments WHERE course_id = $1 AND user_id = $2", courseID, userID)
	if err != nil {
		return fmt.Errorf("deleting enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotEnrolled
	}
	return nil
}

func (r *coursePostgresRepository) IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count + 1
		 WHERE id = $1 AND enrolled_count < quota
		 RETURNING `+courseColumns, courseID))
	if err == nil {
		return course, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, fmt.Errorf("incrementing enrollment count: %w", err)
	}
	if _, findErr := r.FindByID(ctx, courseID); findErr != nil {
		return model.Course{}, findErr
	}
	return model.Course{}, ErrCourseFull
}

func (r *coursePostgresRepository) DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses SET enrolled_count = enrolled_count - 1
		 WHERE id = $1 AND enrolled_count > 0
		 RETURNING `+courseColumns, courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, findErr := r.FindByID(ctx, courseID); findErr != nil {
			return model.Course{}, findErr
		}
		return model.Course{}, errors.New("enrollment counter is inconsistent")
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("decrementing enrollment count: %w", err)
	}
	return course, nil
}

func (r *coursePostgresRepository) UpdateQuota(ctx context.Context, id, quota, version int) (model.Course, error) {
	course, err := scanCourse(r.db.QueryRow(ctx,
		`UPDATE courses
		 SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3 AND $1 >= enrolled_count
		 RETURNING `+courseColumns, quota, id, version))
	if err == nil {
		return course, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, fmt.Errorf("updating course quota: %w", err)
	}
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return model.Course{}, err
	}
	if current.Version != version {
		return model.Course{}, ErrVersionConflict
	}
	return model.Course{}, ErrQuotaBelowFilled
}
