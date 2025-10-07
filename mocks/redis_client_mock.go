package mocks

import (
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

// NewRedisMock returns a real *redis.Client + redismock controller.
// We can ExpectGet/Set/Del and assert expectations.
func NewRedisMock() (*redis.Client, redismock.ClientMock) {
	db, mock := redismock.NewClientMock()
	return db, mock
}
