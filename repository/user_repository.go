package repository

import (
	"errors"
	"sync"

	"modul6/model"
)

var ErrUserNotFound = errors.New("user tidak ditemukan")

type UserRepository interface {
	Create(user model.User) model.User
	GetByUsername(username string) (model.User, error)
	GetByID(id int) (model.User, error)
}

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	users  []model.User
	nextID int
}

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{nextID: 1}
}

func (r *InMemoryUserRepository) Create(user model.User) model.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	user.ID = r.nextID
	r.nextID++
	r.users = append(r.users, user)
	return user
}

func (r *InMemoryUserRepository) GetByUsername(username string) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.Username == username {
			return user, nil
		}
	}
	return model.User{}, ErrUserNotFound
}

func (r *InMemoryUserRepository) GetByID(id int) (model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return model.User{}, ErrUserNotFound
}
