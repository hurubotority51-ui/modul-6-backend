package service

import (
	"context"
	"errors"

	"modul6/model"
	"modul6/repository"
)

var (
	ErrAlreadySupervised  = errors.New("mahasiswa sudah memiliki pembimbing")
	ErrSupervisorFull     = errors.New("kuota pembimbing penuh")
	ErrNotSupervised      = errors.New("mahasiswa tidak memiliki pembimbing tersebut")
	ErrSupervisorVersion  = errors.New("data pembimbing sudah diubah pihak lain")
	ErrSupervisorQuota    = errors.New("kuota di bawah jumlah mahasiswa terbimbing")
	ErrSameSupervisor     = errors.New("pembimbing asal dan tujuan tidak boleh sama")
	ErrSupervisorNotFound = errors.New("pembimbing tidak ditemukan")
)

type SupervisorService struct {
	supervisors repository.SupervisorRepository
	tx          *repository.TxManager
}

func NewSupervisorService(supervisors repository.SupervisorRepository, tx *repository.TxManager) *SupervisorService {
	return &SupervisorService{supervisors: supervisors, tx: tx}
}

func (s *SupervisorService) List(ctx context.Context) ([]model.Supervisor, error) {
	return s.supervisors.List(ctx)
}

func (s *SupervisorService) Assign(ctx context.Context, studentID, supervisorID int) (model.Supervisor, error) {
	var assigned model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)
		if err := repo.LockStudent(ctx, studentID); err != nil {
			return err
		}
		if _, err := repo.FindSupervision(ctx, studentID); err == nil {
			return repository.ErrAlreadySupervised
		} else if !errors.Is(err, repository.ErrNotSupervised) {
			return err
		}
		if err := repo.LockSupervisors(ctx, supervisorID); err != nil {
			return err
		}
		var err error
		assigned, err = repo.IncrementSupervised(ctx, supervisorID)
		if err != nil {
			return err
		}
		return repo.InsertSupervision(ctx, studentID, supervisorID)
	})
	return assigned, mapSupervisorError(err)
}

func (s *SupervisorService) Switch(
	ctx context.Context, studentID, fromID, toID int,
) (model.Supervisor, error) {
	if fromID == toID {
		return model.Supervisor{}, ErrSameSupervisor
	}
	var assigned model.Supervisor
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.supervisors.WithTx(tx)
		if err := repo.LockStudent(ctx, studentID); err != nil {
			return err
		}
		if err := repo.LockSupervisors(ctx, fromID, toID); err != nil {
			return err
		}
		current, err := repo.FindSupervision(ctx, studentID)
		if err != nil {
			return err
		}
		if current.SupervisorID != fromID {
			return repository.ErrNotSupervised
		}
		if _, err := repo.DecrementSupervised(ctx, fromID); err != nil {
			return err
		}
		assigned, err = repo.IncrementSupervised(ctx, toID)
		if err != nil {
			return err
		}
		return repo.UpdateSupervision(ctx, studentID, toID)
	})
	return assigned, mapSupervisorError(err)
}

func (s *SupervisorService) UpdateQuota(
	ctx context.Context, id int, request model.UpdateSupervisorQuotaRequest,
) (model.Supervisor, error) {
	supervisor, err := s.supervisors.UpdateQuota(ctx, id, request.Quota, request.Version)
	return supervisor, mapSupervisorError(err)
}

func mapSupervisorError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return ErrSupervisorNotFound
	case errors.Is(err, repository.ErrStudentNotFound):
		return repository.ErrStudentNotFound
	case errors.Is(err, repository.ErrAlreadySupervised):
		return ErrAlreadySupervised
	case errors.Is(err, repository.ErrSupervisorFull):
		return ErrSupervisorFull
	case errors.Is(err, repository.ErrNotSupervised):
		return ErrNotSupervised
	case errors.Is(err, repository.ErrSupervisorVersion):
		return ErrSupervisorVersion
	case errors.Is(err, repository.ErrSupervisorQuotaBelow):
		return ErrSupervisorQuota
	case errors.Is(err, ErrSameSupervisor):
		return ErrSameSupervisor
	default:
		return err
	}
}
