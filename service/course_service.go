package service

import (
	"context"
	"errors"

	"modul6/model"
	"modul6/repository"
)

var ErrSameCourse = errors.New("kelas asal dan tujuan tidak boleh sama")

type CourseService struct {
	courses repository.CourseRepository
	tx      *repository.TxManager
}

func NewCourseService(courses repository.CourseRepository, tx *repository.TxManager) *CourseService {
	return &CourseService{courses: courses, tx: tx}
}

func (s *CourseService) List(ctx context.Context) ([]model.Course, error) {
	return s.courses.FindAll(ctx)
}

func (s *CourseService) Get(ctx context.Context, id int) (model.Course, error) {
	return s.courses.FindByID(ctx, id)
}

func (s *CourseService) Mine(ctx context.Context, userID int) ([]model.Course, int, error) {
	courses, err := s.courses.FindByStudent(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	for _, course := range courses {
		total += course.Credits
	}
	return courses, total, nil
}

func (s *CourseService) Enroll(ctx context.Context, courseID, userID int) (model.Course, error) {
	var enrolled model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		if err := repo.LockStudent(ctx, userID); err != nil {
			return err
		}
		course, err := repo.FindByID(ctx, courseID)
		if err != nil {
			return err
		}
		exists, err := repo.HasEnrollment(ctx, courseID, userID)
		if err != nil {
			return err
		}
		if exists {
			return repository.ErrDuplicate
		}
		taken, err := repo.SumCredits(ctx, userID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(taken, course.Credits) {
			return repository.ErrCreditLimit
		}
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		return repo.InsertEnrollment(ctx, courseID, userID)
	})
	return enrolled, err
}

func (s *CourseService) Drop(ctx context.Context, courseID, userID int) error {
	return s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		if err := repo.LockStudent(ctx, userID); err != nil {
			return err
		}
		if err := repo.LockCourses(ctx, courseID); err != nil {
			return err
		}
		if err := repo.DeleteEnrollment(ctx, courseID, userID); err != nil {
			return err
		}
		_, err := repo.DecrementEnrolled(ctx, courseID)
		return err
	})
}

func (s *CourseService) Swap(ctx context.Context, userID, fromID, toID int) (model.Course, error) {
	if fromID == toID {
		return model.Course{}, ErrSameCourse
	}
	var joined model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)
		if err := repo.LockStudent(ctx, userID); err != nil {
			return err
		}
		if err := repo.LockCourses(ctx, fromID, toID); err != nil {
			return err
		}
		from, err := repo.FindByID(ctx, fromID)
		if err != nil {
			return err
		}
		to, err := repo.FindByID(ctx, toID)
		if err != nil {
			return err
		}
		hasFrom, err := repo.HasEnrollment(ctx, fromID, userID)
		if err != nil {
			return err
		}
		if !hasFrom {
			return repository.ErrNotEnrolled
		}
		hasTo, err := repo.HasEnrollment(ctx, toID, userID)
		if err != nil {
			return err
		}
		if hasTo {
			return repository.ErrDuplicate
		}
		taken, err := repo.SumCredits(ctx, userID)
		if err != nil {
			return err
		}
		if CreditsAfterSwap(taken, from.Credits, to.Credits) > MaxCreditsPerSemester {
			return repository.ErrCreditLimit
		}
		if err := repo.DeleteEnrollment(ctx, fromID, userID); err != nil {
			return err
		}
		if _, err := repo.DecrementEnrolled(ctx, fromID); err != nil {
			return err
		}
		joined, err = repo.IncrementEnrolled(ctx, toID)
		if err != nil {
			return err
		}
		return repo.InsertEnrollment(ctx, toID, userID)
	})
	return joined, err
}

func (s *CourseService) UpdateQuota(ctx context.Context, id int, request model.UpdateQuotaRequest) (model.Course, error) {
	return s.courses.UpdateQuota(ctx, id, request.Quota, request.Version)
}
