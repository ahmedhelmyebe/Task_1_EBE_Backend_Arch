package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"HelmyTask/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetup_Smoke(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	svc := new(mocks.UserServiceMock)

	Setup(r, svc, "secret", time.Hour)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code) // route exists; body missing
}
