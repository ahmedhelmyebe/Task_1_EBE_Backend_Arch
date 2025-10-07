package services

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"HelmyTask/mocks"
	"HelmyTask/models"
	"HelmyTask/repositories"

	"HelmyTask/utils"
	"HelmyTask/utils/redislog"

	// "github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newSvc(repo repositories.UserRepository, rdb *redis.Client, l *redislog.Logger) UserService {
	return NewUserService(repo, rdb, l)
}

// small helper to build deterministic JSON for a user (matches service marshal)
func mustUserJSON(u models.User) string {
	b, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestUserService_Register_EmailExists(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	// repo claims email exists
	repo.On("FindByEmail", "a@b.c").Return(&models.User{ID: 1}, nil)

	// use a NO-OP logger (nil redis client) so we don't need to mock LPUSH/LTRIM/EXPIRE
	noLog := redislog.New(nil, "", 0, 0)

	svc := newSvc(repo, nil, noLog)

	u, err := svc.Register(models.RegisterRequest{Name: "  aHMED  ", Email: "a@b.c", Password: "123456"})
	assert.Nil(t, u)
	assert.EqualError(t, err, "email already exists")
}

func TestUserService_Register_Success_NormalizesAndCaches(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	rdb, rmock := mocks.NewRedisMock()

	// use a NO-OP logger (nil redis client) so we don't need to mock LPUSH/LTRIM/EXPIRE
	noLog := redislog.New(nil, "", 0, 0)

	// email not found
	repo.On("FindByEmail", "a@b.c").Return(nil, errors.New("not found"))
	// Create sets an ID; we capture and modify the arg
	repo.On("Create", mock.AnythingOfType("*models.User")).Return(nil).Run(func(args mock.Arguments) {
		u := args.Get(0).(*models.User)
		u.ID = 10
	})

	// exact JSON cached by service after register
	expectedCached := mustUserJSON(models.User{
		ID:    10,
		Name:  "AHMED", // NormalizeName applied
		Email: "a@b.c",
		// Password omitted by json:"-"
		// CreatedAt/UpdatedAt are zero values → "0001-01-01T00:00:00Z"
	})
	rmock.ExpectSet("user:10", []byte(expectedCached), 10*time.Minute).SetVal("OK")

	svc := newSvc(repo, rdb, noLog)

	u, err := svc.Register(models.RegisterRequest{Name: "  aHMED  ", Email: "a@b.c", Password: "123456"})
	assert.NoError(t, err)
	assert.Equal(t, uint(10), u.ID)
	assert.Equal(t, "AHMED", u.Name) // PROVES NormalizeName was applied

	assert.NoError(t, rmock.ExpectationsWereMet())
}

func TestUserService_Login_Invalid(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	repo.On("FindByEmail", "x@y.z").Return(nil, errors.New("not found"))

	svc := newSvc(repo, nil, nil)
	tok, err := svc.Login(models.LoginRequest{Email: "x@y.z", Password: "pw"}, "sec", time.Hour)
	assert.Empty(t, tok)
	assert.EqualError(t, err, "invalid credentials")
}

func TestUserService_Login_Success_JWT(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	hash, _ := utils.HashPassword("good")
	repo.On("FindByEmail", "x@y.z").Return(&models.User{ID: 7, Email: "x@y.z", Password: hash}, nil)

	svc := newSvc(repo, nil, nil)
	tok, err := svc.Login(models.LoginRequest{Email: "x@y.z", Password: "good"}, "sec", time.Minute)
	assert.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestUserService_GetByID_CacheHit(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	rdb, rmock := mocks.NewRedisMock()
	svc := newSvc(repo, rdb, nil)

	u := models.User{ID: 5, Name: "Ahmed", Email: "a@b.c"}
	b, _ := json.Marshal(u)
	rmock.ExpectGet("user:5").SetVal(string(b))

	got, err := svc.GetByID(5)
	assert.NoError(t, err)
	assert.Equal(t, u.Email, got.Email)
	assert.NoError(t, rmock.ExpectationsWereMet())
}

func TestUserService_GetByID_MissThenDBThenSet(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	rdb, rmock := mocks.NewRedisMock()
	svc := newSvc(repo, rdb, nil)

	rmock.ExpectGet("user:9").RedisNil()
	repo.On("FindByID", uint(9)).Return(&models.User{ID: 9, Email: "a@b.c"}, nil)

	// exact JSON for the cached value after DB hit
	expectedCached := mustUserJSON(models.User{
		ID:    9,
		Email: "a@b.c",
	})
	rmock.ExpectSet("user:9", []byte(expectedCached), 10*time.Minute).SetVal("OK")

	got, err := svc.GetByID(9)
	assert.NoError(t, err)
	assert.Equal(t, uint(9), got.ID)
	assert.NoError(t, rmock.ExpectationsWereMet())
}

func TestUserService_UpdateUser_NameNormalized_RefreshCache(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	rdb, rmock := mocks.NewRedisMock()
	svc := newSvc(repo, rdb, nil)

	repo.On("FindByID", uint(2)).Return(&models.User{ID: 2, Name: "Old"}, nil)
	repo.On("Update", mock.AnythingOfType("*models.User")).Return(nil)

	rmock.ExpectDel("user:2").SetVal(1)

	// exact JSON for the updated cached value (after normalization)
	expectedCached := mustUserJSON(models.User{
		ID:   2,
		Name: "AHMED",
	})
	rmock.ExpectSet("user:2", []byte(expectedCached), 10*time.Minute).SetVal("OK")

	newName := "  aHMED "
	got, err := svc.UpdateUser(2, models.UpdateUserRequest{Name: &newName})
	assert.NoError(t, err)
	assert.Equal(t, "AHMED", got.Name) // again proves NormalizeName

	assert.NoError(t, rmock.ExpectationsWereMet())
}

func TestUserService_DeleteUser_DeletesAndClearsCache(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	rdb, rmock := mocks.NewRedisMock()
	svc := newSvc(repo, rdb, nil)

	repo.On("Delete", uint(3)).Return(nil)
	rmock.ExpectDel("user:3").SetVal(1)

	err := svc.DeleteUser(3)
	assert.NoError(t, err)
	assert.NoError(t, rmock.ExpectationsWereMet())
}

func TestUserService_ListUsers_Clamp(t *testing.T) {
	repo := new(mocks.UserRepositoryMock)
	svc := newSvc(repo, nil, nil)

	repo.On("List", 0, 10).Return([]models.User{{ID: 1}}, int64(1), nil)

	out, err := svc.ListUsers(0, 1000)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(out.Items))
	assert.Equal(t, int64(1), out.Total)
}
