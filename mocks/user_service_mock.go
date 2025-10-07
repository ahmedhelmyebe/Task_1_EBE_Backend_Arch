package mocks

import (
	"HelmyTask/models"
	"github.com/stretchr/testify/mock"
	"time"
)

// UserServiceMock is a testify/mock for services.UserService.
// We use this to test the HTTP handlers without real business logic.
type UserServiceMock struct{ mock.Mock }

func (m *UserServiceMock) Register(req models.RegisterRequest) (*models.User, error) {
	args := m.Called(req)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserServiceMock) Login(req models.LoginRequest, jwtSecret string, exp time.Duration) (string, error) {
	args := m.Called(req, jwtSecret, exp)
	return args.String(0), args.Error(1)
}

func (m *UserServiceMock) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserServiceMock) CreateUser(req models.RegisterRequest) (*models.User, error) {
	args := m.Called(req)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserServiceMock) GetUser(id uint) (*models.User, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserServiceMock) UpdateUser(id uint, req models.UpdateUserRequest) (*models.User, error) {
	args := m.Called(id, req)
	if v := args.Get(0); v != nil {
		return v.(*models.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *UserServiceMock) DeleteUser(id uint) error {
	return m.Called(id).Error(0)
}

func (m *UserServiceMock) ListUsers(page, limit int) (*models.PagedUsers, error) {
	args := m.Called(page, limit)
	if v := args.Get(0); v != nil {
		return v.(*models.PagedUsers), args.Error(1)
	}
	return nil, args.Error(1)
}
