package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"modul6/model"
)

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  []model.User
	nextID int
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{nextID: 1}
}

func (r *InMemoryUserRepository) Create(_ context.Context, user model.User) (model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, current := range r.users {
		if current.Username == user.Username {
			return model.User{}, ErrDuplicate
		}
	}
	user.ID = r.nextID
	r.nextID++
	if user.Email == "" {
		user.Email = user.Username + "@local.invalid"
	}
	r.users = append(r.users, user)
	return user, nil
}

func (r *InMemoryUserRepository) GetByUsername(_ context.Context, username string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return model.User{}, ErrUserNotFound
}

func (r *InMemoryUserRepository) GetByID(_ context.Context, id int) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return model.User{}, ErrUserNotFound
}

func (r *InMemoryUserRepository) HasPermission(_ context.Context, role, permission string) (bool, error) {
	return role == "admin" &&
		(permission == "course:manage" || permission == "supervision:manage"), nil
}

type InMemoryStudentRepository struct {
	mu       sync.RWMutex
	students []model.Student
	nextID   int
}

func NewInMemoryStudentRepository() *InMemoryStudentRepository {
	return &InMemoryStudentRepository{nextID: 1}
}

func (r *InMemoryStudentRepository) List(_ context.Context, query model.CursorQuery) ([]model.Student, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	students := make([]model.Student, 0, len(r.students))
	for _, student := range r.students {
		if query.IsActive != nil && student.IsActive != *query.IsActive {
			continue
		}
		if query.Search != "" && !strings.Contains(strings.ToLower(student.Name), strings.ToLower(query.Search)) {
			continue
		}
		if query.After != nil &&
			!(student.CreatedAt.Before(query.After.CreatedAt) ||
				(student.CreatedAt.Equal(query.After.CreatedAt) && student.ID < query.After.ID)) {
			continue
		}
		students = append(students, student)
	}
	sort.Slice(students, func(i, j int) bool {
		if students[i].CreatedAt.Equal(students[j].CreatedAt) {
			return students[i].ID > students[j].ID
		}
		return students[i].CreatedAt.After(students[j].CreatedAt)
	})
	if len(students) > query.Limit+1 {
		students = students[:query.Limit+1]
	}
	return students, nil
}

func (r *InMemoryStudentRepository) GetByID(_ context.Context, id int) (model.Student, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, student := range r.students {
		if student.ID == id {
			return student, nil
		}
	}
	return model.Student{}, ErrStudentNotFound
}

func (r *InMemoryStudentRepository) Create(_ context.Context, student model.Student) (model.Student, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, current := range r.students {
		if strings.EqualFold(current.NIM, student.NIM) {
			return model.Student{}, ErrDuplicate
		}
	}
	student.ID = r.nextID
	r.nextID++
	if student.CreatedAt.IsZero() {
		student.CreatedAt = time.Now().UTC()
	}
	r.students = append(r.students, student)
	return student, nil
}

func (r *InMemoryStudentRepository) Update(_ context.Context, student model.Student) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.students {
		if r.students[i].ID == student.ID {
			for j, current := range r.students {
				if j != i && strings.EqualFold(current.NIM, student.NIM) {
					return ErrDuplicate
				}
			}
			r.students[i] = student
			return nil
		}
	}
	return ErrStudentNotFound
}
