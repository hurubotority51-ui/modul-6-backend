package repository

import (
	"errors"
	"sync"

	"modul6/model"
)

var ErrStudentNotFound = errors.New("student tidak ditemukan")

type StudentRepository interface {
	List() []model.Student
	GetByID(id int) (model.Student, error)
	Create(student model.Student) model.Student
}

type InMemoryStudentRepository struct {
	mu       sync.RWMutex
	students []model.Student
	nextID   int
}

func NewInMemoryStudentRepository() *InMemoryStudentRepository {
	return &InMemoryStudentRepository{nextID: 1}
}

func (r *InMemoryStudentRepository) List() []model.Student {
	r.mu.RLock()
	defer r.mu.RUnlock()
	students := make([]model.Student, len(r.students))
	copy(students, r.students)
	return students
}

func (r *InMemoryStudentRepository) GetByID(id int) (model.Student, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, student := range r.students {
		if student.ID == id {
			return student, nil
		}
	}
	return model.Student{}, ErrStudentNotFound
}

func (r *InMemoryStudentRepository) Create(student model.Student) model.Student {
	r.mu.Lock()
	defer r.mu.Unlock()
	student.ID = r.nextID
	r.nextID++
	r.students = append(r.students, student)
	return student
}
