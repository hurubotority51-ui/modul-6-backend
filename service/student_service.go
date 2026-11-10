package service

import (
	"context"
	"errors"
	"strings"

	"modul6/helper"
	"modul6/model"
	"modul6/repository"
)

var ErrDuplicateNIM = errors.New("nim sudah digunakan")

type StudentService struct {
	repository repository.StudentRepository
}

func NewStudentService(repository repository.StudentRepository) *StudentService {
	return &StudentService{repository: repository}
}

func (s *StudentService) List(ctx context.Context, query model.CursorQuery) ([]model.Student, model.CursorMeta, error) {
	students, err := s.repository.List(ctx, query)
	if err != nil {
		return nil, model.CursorMeta{}, err
	}
	hasMore := len(students) > query.Limit
	if hasMore {
		students = students[:query.Limit]
	}
	meta := model.CursorMeta{Limit: query.Limit, HasMore: hasMore}
	if hasMore && len(students) > 0 {
		last := students[len(students)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}
	return students, meta, nil
}

func (s *StudentService) GetByID(ctx context.Context, id int) (model.Student, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *StudentService) Create(ctx context.Context, request model.CreateStudentRequest) (model.Student, error) {
	request.NIM = strings.TrimSpace(request.NIM)
	request.Name = strings.TrimSpace(request.Name)
	student, err := s.repository.Create(ctx, model.Student{
		NIM: request.NIM, Name: request.Name, Grade: request.Grade, IsActive: request.IsActive,
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return model.Student{}, ErrDuplicateNIM
	}
	return student, err
}

func (s *StudentService) Patch(ctx context.Context, id int, request model.PatchStudentRequest) (model.Student, error) {
	student, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return model.Student{}, err
	}
	if request.NIM != nil {
		student.NIM = strings.TrimSpace(*request.NIM)
	}
	if request.Name != nil {
		student.Name = strings.TrimSpace(*request.Name)
	}
	if request.Grade != nil {
		student.Grade = *request.Grade
	}
	if request.IsActive != nil {
		student.IsActive = *request.IsActive
	}
	if err := s.repository.Update(ctx, student); errors.Is(err, repository.ErrDuplicate) {
		return model.Student{}, ErrDuplicateNIM
	} else if err != nil {
		return model.Student{}, err
	}
	return student, nil
}
