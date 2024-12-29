package utils

import (
	"fmt"
	"os"
)

// LogInfo logs an informational message
func LogInfo(msg string) {
	fmt.Println("[INFO]:", msg)
}

// define log levels
type LogLevel string

const (
	LEVEL_TRACE LogLevel = "TRACE"
	LEVEL_DEBUG LogLevel = "DEBUG"
	LEVEL_INFO  LogLevel = "INFO"
	LEVEL_WARN  LogLevel = "WARN"
	LEVEL_ERROR LogLevel = "ERROR"
)

var logLevel = func() LogLevel {
	lvl, _ := os.LookupEnv("IMPOSTER_LOG_LEVEL")
	if lvl == "" {
		return LEVEL_DEBUG
	}
	return LogLevel(lvl)
}()

func GetLogLevel() LogLevel {
	return logLevel
}
