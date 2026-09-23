package logger

import (
	"log/slog"
	"os"
)

// New 创建结构化 JSON 日志。
func New() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
