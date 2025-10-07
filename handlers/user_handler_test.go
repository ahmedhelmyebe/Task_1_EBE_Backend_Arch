package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	
	"HelmyTask/mocks"
	"HelmyTask/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setup(r *gin.Engine, svc *mocks.UserServiceMock) {
	h := NewUserHandler(svc, "test-secret", time.Minute)
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.GET("/users/:id", h.GetUser)
	r.POST("/users", h.CreateUser)
	r.PUT("/users/:id", h.UpdateUser)
	r.DELETE("/users/:id", h.DeleteUser)
	r.GET("/users", h.ListUsers)
}

func TestRegister_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := new(mocks.UserServiceMock)
	setup(r, svc)

	req := models.RegisterRequest{Name: "ahmed", Email: "a@b.c", Password: "123456"}
	resp := &models.User{ID: 1, Name: "Ahmed", Email: "a@b.c"}
	svc.On("Register", req).Return(resp, nil)

	b, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"id":1`)
}

func TestLogin_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := new(mocks.UserServiceMock)
	setup(r, svc)

	body := models.LoginRequest{Email: "x@y.z", Password: "oops"}
	svc.On("Login", body, "test-secret", time.Minute).Return("", assert.AnError)

	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetUser_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := new(mocks.UserServiceMock)
	setup(r, svc)

	svc.On("GetUser", uint(99)).Return(nil, assert.AnError)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/99", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

