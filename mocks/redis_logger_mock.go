package mocks

import (
	"HelmyTask/utils/redislog"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"time"
)

// NewRedisLoggerWithMock constructs a real redislog.Logger over a mocked redis client.
// This lets us check LPUSH/LTRIM/EXPIRE calls when Info/Warn/Error are used.
func NewRedisLoggerWithMock() (*redislog.Logger, *redis.Client, redismock.ClientMock) {
	rc, mock := redismock.NewClientMock()
	logger := redislog.New(rc, "logs:app", 100, 24*time.Hour)
	return logger, rc, mock
}
