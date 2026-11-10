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
	ErrSupervisorFull       = errors.New("kuota pembimbing sudah penuh")
	ErrAlreadySupervised    = errors.New("mahasiswa sudah memiliki pembimbing")
	ErrNotSupervised        = errors.New("penetapan pembimbing tidak ditemukan")
	ErrSupervisorVersion    = errors.New("data pembimbing sudah diubah pihak lain")
	ErrSupervisorQuotaBelow = errors.New("kuota di bawah jumlah mahasiswa terbimbing")
)

type SupervisorRepository interface {
	WithTx(tx DBTX) SupervisorRepository
	List(ctx context.Context) ([]model.Supervisor, error)
	FindByID(ctx context.Context, id int) (model.Supervisor, error)
	LockStudent(ctx context.Context, studentID int) error
	LockSupervisors(ctx context.Context, ids ...int) error
	FindSupervision(ctx context.Context, studentID int) (model.Supervision, error)
	InsertSupervision(ctx context.Context, studentID, supervisorID int) error
	UpdateSupervision(ctx context.Context, studentID, supervisorID int) error
	IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error)
	UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error)
}

const supervisorColumns = `id, name, quota, supervised_count, version, created_at`

type supervisorPostgresRepository struct {
	db DBTX
}

func NewSupervisorRepository(db DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: db}
}

func (r *supervisorPostgresRepository) WithTx(tx DBTX) SupervisorRepository {
	return &supervisorPostgresRepository{db: tx}
}

func scanSupervisor(row pgx.Row) (model.Supervisor, error) {
	var supervisor model.Supervisor
	err := row.Scan(
		&supervisor.ID, &supervisor.Name, &supervisor.Quota,
		&supervisor.SupervisedCount, &supervisor.Version, &supervisor.CreatedAt,
	)
	return supervisor, err
}

func (r *supervisorPostgresRepository) List(ctx context.Context) ([]model.Supervisor, error) {
	rows, err := r.db.Query(ctx, "SELECT "+supervisorColumns+" FROM supervisors ORDER BY name, id")
	if err != nil {
		return nil, fmt.Errorf("listing supervisors: %w", err)
	}
	defer rows.Close()
	supervisors := make([]model.Supervisor, 0)
	for rows.Next() {
		supervisor, err := scanSupervisor(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning supervisor: %w", err)
		}
		supervisors = append(supervisors, supervisor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading supervisors: %w", err)
	}
	return supervisors, nil
}

func (r *supervisorPostgresRepository) FindByID(ctx context.Context, id int) (model.Supervisor, error) {
	supervisor, err := scanSupervisor(r.db.QueryRow(ctx,
		"SELECT "+supervisorColumns+" FROM supervisors WHERE id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, ErrNotFound
	}
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("getting supervisor: %w", err)
	}
	return supervisor, nil
}

func (r *supervisorPostgresRepository) LockStudent(ctx context.Context, studentID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		"SELECT id FROM students WHERE id = $1 FOR UPDATE", studentID,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrStudentNotFound
	}
	if err != nil {
		return fmt.Errorf("locking student: %w", err)
	}
	return nil
}

func (r *supervisorPostgresRepository) LockSupervisors(ctx context.Context, ids ...int) error {
	ordered := append([]int(nil), ids...)
	sort.Ints(ordered)
	lastID := 0
	for _, id := range ordered {
		if id == lastID {
			continue
		}
		lastID = id
		var found int
		err := r.db.QueryRow(ctx,
			"SELECT id FROM supervisors WHERE id = $1 FOR UPDATE", id,
		).Scan(&found)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("locking supervisor: %w", err)
		}
	}
	return nil
}

func (r *supervisorPostgresRepository) FindSupervision(ctx context.Context, studentID int) (model.Supervision, error) {
	var supervision model.Supervision
	err := r.db.QueryRow(ctx,
		`SELECT student_id, supervisor_id, assigned_at
		 FROM supervisions WHERE student_id = $1`, studentID,
	).Scan(&supervision.StudentID, &supervision.SupervisorID, &supervision.AssignedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Supervision{}, ErrNotSupervised
	}
	if err != nil {
		return model.Supervision{}, fmt.Errorf("getting supervision: %w", err)
	}
	return supervision, nil
}

func (r *supervisorPostgresRepository) InsertSupervision(ctx context.Context, studentID, supervisorID int) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO supervisions (student_id, supervisor_id) VALUES ($1, $2)",
		studentID, supervisorID,
	)
	if pgErrorCode(err) == codeUniqueViolation {
		return ErrAlreadySupervised
	}
	if err != nil {
		return fmt.Errorf("assigning supervision: %w", err)
	}
	return nil
}

func (r *supervisorPostgresRepository) UpdateSupervision(ctx context.Context, studentID, supervisorID int) error {
	tag, err := r.db.Exec(ctx,
		"UPDATE supervisions SET supervisor_id = $1, assigned_at = NOW() WHERE student_id = $2",
		supervisorID, studentID,
	)
	if err != nil {
		return fmt.Errorf("switching supervision: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotSupervised
	}
	return nil
}

func (r *supervisorPostgresRepository) IncrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	supervisor, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count + 1
		 WHERE id = $1 AND supervised_count < quota
		 RETURNING `+supervisorColumns, supervisorID))
	if err == nil {
		return supervisor, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("incrementing supervised count: %w", err)
	}
	if _, findErr := r.FindByID(ctx, supervisorID); findErr != nil {
		return model.Supervisor{}, findErr
	}
	return model.Supervisor{}, ErrSupervisorFull
}

func (r *supervisorPostgresRepository) DecrementSupervised(ctx context.Context, supervisorID int) (model.Supervisor, error) {
	supervisor, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors SET supervised_count = supervised_count - 1
		 WHERE id = $1 AND supervised_count > 0
		 RETURNING `+supervisorColumns, supervisorID))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, findErr := r.FindByID(ctx, supervisorID); findErr != nil {
			return model.Supervisor{}, findErr
		}
		return model.Supervisor{}, errors.New("supervisor count is inconsistent")
	}
	if err != nil {
		return model.Supervisor{}, fmt.Errorf("decrementing supervised count: %w", err)
	}
	return supervisor, nil
}

func (r *supervisorPostgresRepository) UpdateQuota(ctx context.Context, id, quota, version int) (model.Supervisor, error) {
	supervisor, err := scanSupervisor(r.db.QueryRow(ctx,
		`UPDATE supervisors
		 SET quota = $1, version = version + 1
		 WHERE id = $2 AND version = $3 AND $1 >= supervised_count
		 RETURNING `+supervisorColumns, quota, id, version))
	if err == nil {
		return supervisor, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Supervisor{}, fmt.Errorf("updating supervisor quota: %w", err)
	}
	current, err := r.FindByID(ctx, id)
	if err != nil {
		return model.Supervisor{}, err
	}
	if current.Version != version {
		return model.Supervisor{}, ErrSupervisorVersion
	}
	return model.Supervisor{}, ErrSupervisorQuotaBelow
}
