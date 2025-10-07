package config

import (
	"os"
	"testing"

	

	"github.com/stretchr/testify/assert"
)

func TestLoad_EnvOverrides(t *testing.T) {
	_ = os.Setenv("APP_HTTP_PORT", "9090")
	_ = os.Setenv("APP_DB_DRIVER", "sqlite")
	_ = os.Setenv("APP_SQLITE_PATH", "test.db")
	t.Cleanup(func() {
		_ = os.Unsetenv("APP_HTTP_PORT")
		_ = os.Unsetenv("APP_DB_DRIVER")
		_ = os.Unsetenv("APP_SQLITE_PATH")
	})

	cfg := Load()

	assert.Equal(t, "9090", cfg.HTTPPort)
	assert.Equal(t, "sqlite", cfg.DBDriver)
	assert.Equal(t, "test.db", cfg.SQLitePath)
}

