package service

import (
	"errors"
	"sort"
	"strings"
	"time"

	"modul6/model"
	"modul6/repository"
)

var ErrInvalidStudent = errors.New("data student tidak valid")
var ErrDuplicateNIM = errors.New("nim sudah digunakan")

type StudentService struct {
	repository repository.StudentRepository
}

func NewStudentService(repository repository.StudentRepository) *StudentService {
	return &StudentService{repository: repository}
}

func (s *StudentService) List(query model.ListQuery) ([]model.Student, model.Meta) {
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

	sort.SliceStable(filtered, func(i, j int) bool {
		var less bool
		switch query.Sort {
		case "nim":
			less = filtered[i].NIM < filtered[j].NIM
		case "name":
			less = filtered[i].Name < filtered[j].Name
		case "grade":
			less = filtered[i].Grade < filtered[j].Grade
		case "created_at":
			less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
		default:
			less = filtered[i].ID < filtered[j].ID
		}
		if query.Order == "desc" {
			return !less
		}
		return less
	})

	total := len(filtered)
	totalPages := 0
	if total > 0 {
		totalPages = (total + query.Limit - 1) / query.Limit
	}
	start := (query.Page - 1) * query.Limit
	if start > total {
		start = total
	}
	end := start + query.Limit
	if end > total {
		end = total
	}
	return filtered[start:end], model.Meta{Page: query.Page, Limit: query.Limit, Total: total, TotalPages: totalPages}
}

func (s *StudentService) GetByID(id int) (model.Student, error) {
	return s.repository.GetByID(id)
}

func (s *StudentService) Create(request model.CreateStudentRequest) (model.Student, error) {
	request.NIM = strings.TrimSpace(request.NIM)
	request.Name = strings.TrimSpace(request.Name)
	if request.NIM == "" || request.Name == "" || request.Grade < 0 || request.Grade > 4 {
		return model.Student{}, ErrInvalidStudent
	}
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
