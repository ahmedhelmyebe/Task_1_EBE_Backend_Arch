package redislog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry is a structured log object saved into Redis as JSON.
type Entry struct {
	Level string            `json:"level"`
	Msg   string            `json:"msg"`
	Time  string            `json:"time"`
	Meta  map[string]string `json:"meta,omitempty"`
}

// Logger pushes logs to a Redis LIST (e.g., "logs:app") and trims to a max length.
type Logger struct {
	rdb       *redis.Client
	key       string        // list key, e.g. "logs:app"
	max       int64         // keep last N entries
	retention time.Duration // optional expire for the list key
}

// New creates a Redis logger using a LIST. You’ll see this key in your Redis Desktop Manager.
func New(rdb *redis.Client, key string, max int64, retention time.Duration) *Logger {
	return &Logger{rdb: rdb, key: key, max: max, retention: retention}
}

// log pushes a log entry as JSON -> LPUSH; then LTRIM; then EXPIRE.
func (l *Logger) log(level, msg string, meta map[string]string) {
	if l == nil || l.rdb == nil {
		return // no-op if logger not initialized
	}
	en := Entry{
		Level: level,
		Msg:   msg,
		Time:  time.Now().UTC().Format(time.RFC3339),
		Meta:  meta,
	}
	b, _ := json.Marshal(en)
	ctx := context.Background()
	_ = l.rdb.LPush(ctx, l.key, b).Err()
	_ = l.rdb.LTrim(ctx, l.key, 0, l.max-1).Err()
	if l.retention > 0 {
		_ = l.rdb.Expire(ctx, l.key, l.retention).Err()
	}
}

// Convenience helpers

//Log severity = normal information (not an error, not a warning).
func (l *Logger) Info(msg string, meta map[string]string)  { l.log("info", msg, meta) }


func (l *Logger) Warn(msg string, meta map[string]string)  { l.log("warn", msg, meta) }
func (l *Logger) Error(msg string, meta map[string]string) { l.log("error", msg, meta) }

// Formatted variants
func (l *Logger) Infof(format string, meta map[string]string, args ...any)  { l.Info(fmt.Sprintf(format, args...), meta) }
func (l *Logger) Warnf(format string, meta map[string]string, args ...any)  { l.Warn(fmt.Sprintf(format, args...), meta) }
func (l *Logger) Errorf(format string, meta map[string]string, args ...any) { l.Error(fmt.Sprintf(format, args...), meta) }
