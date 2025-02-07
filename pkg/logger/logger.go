package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
)

var (
	Trace *log.Logger
	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger

	currentLevel LogLevel
)

func init() {
	Trace = log.New(os.Stdout, "[TRACE] ", log.Ldate|log.Ltime)
	Debug = log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime)
	Info = log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime)
	Warn = log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime)
	Error = log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime)

	setLogLevel()
}

func setLogLevel() {
	lvl := os.Getenv("IMPOSTER_LOG_LEVEL")
	if lvl == "" {
		currentLevel = DEBUG // Default level
		return
	}

	switch strings.ToUpper(lvl) {
	case "TRACE":
		currentLevel = TRACE
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = DEBUG
	}

	// Set output to discard for disabled levels
	if !IsTraceEnabled() {
		Trace.SetOutput(io.Discard)
	}
	if !IsDebugEnabled() {
		Debug.SetOutput(io.Discard)
	}
	if !IsInfoEnabled() {
		Info.SetOutput(io.Discard)
	}
	if !IsWarnEnabled() {
		Warn.SetOutput(io.Discard)
	}
	if !IsErrorEnabled() {
		Error.SetOutput(io.Discard)
	}
}

// Level check functions
func IsTraceEnabled() bool {
	return currentLevel <= TRACE
}

func IsDebugEnabled() bool {
	return currentLevel <= DEBUG
}

func IsInfoEnabled() bool {
	return currentLevel <= INFO
}

func IsWarnEnabled() bool {
	return currentLevel <= WARN
}

func IsErrorEnabled() bool {
	return currentLevel <= ERROR
}

// Trace level logging
func Tracef(format string, v ...interface{}) {
	if IsTraceEnabled() {
		Trace.Printf(format, v...)
	}
}

func Traceln(msg string) {
	if IsTraceEnabled() {
		Trace.Println(msg)
	}
}

// Debug level logging
func Debugf(format string, v ...interface{}) {
	if IsDebugEnabled() {
		Debug.Printf(format, v...)
	}
}

func Debugln(msg string) {
	if IsDebugEnabled() {
		Debug.Println(msg)
	}
}

// Info level logging
func Infof(format string, v ...interface{}) {
	if IsInfoEnabled() {
		Info.Printf(format, v...)
	}
}

func Infoln(msg string) {
	if IsInfoEnabled() {
		Info.Println(msg)
	}
}

// Warn level logging
func Warnf(format string, v ...interface{}) {
	if IsWarnEnabled() {
		Warn.Printf(format, v...)
	}
}

func Warnln(msg string) {
	if IsWarnEnabled() {
		Warn.Println(msg)
	}
}

// Error level logging
func Errorf(format string, v ...interface{}) {
	if IsErrorEnabled() {
		Error.Printf(format, v...)
	}
}

func Errorln(msg string) {
	if IsErrorEnabled() {
		Error.Println(msg)
	}
}

// GetCurrentLevel returns the current log level
func GetCurrentLevel() LogLevel {
	return currentLevel
}
