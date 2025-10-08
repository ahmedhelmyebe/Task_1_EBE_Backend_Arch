package mocks

import (
	"HelmyTask/models"
	"github.com/stretchr/testify/mock"
)

// UserRepositoryMock is a testify/mock for repositories.UserRepository.
// We use this to unit-test the service layer without touching a DB.
type UserRepositoryMock struct{ mock.Mock }

func (m *UserRepositoryMock) Create(u *models.User) error {
	return m.Called(u).Error(0)
}

func (m *UserRepositoryMock) FindByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserRepositoryMock) FindByID(id uint) (*models.User, error) { 
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserRepositoryMock) Update(u *models.User) error {
	return m.Called(u).Error(0)
}

func (m *UserRepositoryMock) Delete(id uint) error {
	return m.Called(id).Error(0)
}

func (m *UserRepositoryMock) List(offset, limit int) ([]models.User, int64, error) {
	args := m.Called(offset, limit)
	var items []models.User
	if v := args.Get(0); v != nil {
		items = v.([]models.User)
	}
	var total int64
	if v := args.Get(1); v != nil {
		total = v.(int64)
	}
	return items, total, args.Error(2)
}
