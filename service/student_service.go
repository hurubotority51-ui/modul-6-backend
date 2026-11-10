package service

import (
	"errors"
	"sort"
	"strings"
	"time"

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

func (s *StudentService) List(query model.CursorQuery) ([]model.Student, model.CursorMeta) {
	students := s.repository.List()
	filtered := make([]model.Student, 0, len(students))
	for _, student := range students {
		if query.IsActive != nil && student.IsActive != *query.IsActive {
			continue
		}
		if query.Search != "" && !strings.Contains(strings.ToLower(student.Name), strings.ToLower(query.Search)) {
			continue
		}
		filtered = append(filtered, student)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID > filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	if query.After != nil {
		after := query.After
		remaining := filtered[:0]
		for _, student := range filtered {
			if student.CreatedAt.Before(after.CreatedAt) ||
				(student.CreatedAt.Equal(after.CreatedAt) && student.ID < after.ID) {
				remaining = append(remaining, student)
			}
		}
		filtered = remaining
	}
	hasMore := len(filtered) > query.Limit
	if hasMore {
		filtered = filtered[:query.Limit]
	}
	meta := model.CursorMeta{Limit: query.Limit, HasMore: hasMore}
	if hasMore && len(filtered) > 0 {
		last := filtered[len(filtered)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}
	return filtered, meta
}

func (s *StudentService) GetByID(id int) (model.Student, error) {
	return s.repository.GetByID(id)
}

func (s *StudentService) Create(request model.CreateStudentRequest) (model.Student, error) {
	request.NIM = strings.TrimSpace(request.NIM)
	request.Name = strings.TrimSpace(request.Name)
	for _, student := range s.repository.List() {
		if strings.EqualFold(student.NIM, request.NIM) {
			return model.Student{}, ErrDuplicateNIM
		}
	}
	return s.repository.Create(model.Student{
		NIM: request.NIM, Name: request.Name, Grade: request.Grade,
		IsActive: request.IsActive, CreatedAt: time.Now(),
	}), nil
}

func (s *StudentService) Patch(id int, request model.PatchStudentRequest) (model.Student, error) {
	student, err := s.repository.GetByID(id)
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
	for _, current := range s.repository.List() {
		if current.ID != student.ID && strings.EqualFold(current.NIM, student.NIM) {
			return model.Student{}, ErrDuplicateNIM
		}
	}
	if err := s.repository.Update(student); err != nil {
		return model.Student{}, err
	}
	return student, nil
}
