package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
)

// writerProxy is a Writer that forwards write calls to the underlying writer
type writerProxy struct {
	mu     sync.RWMutex
	writer io.Writer
}

func (w *writerProxy) Write(p []byte) (n int, err error) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.writer.Write(p)
}

func (w *writerProxy) getWriter() io.Writer {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.writer
}

func (w *writerProxy) setWriter(writer io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer = writer
}

var (
	outputSink = &writerProxy{writer: os.Stdout}
	errorSink  = &writerProxy{writer: os.Stderr}

	Trace *log.Logger
	Debug *log.Logger
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger

	currentLevel LogLevel
)

func init() {
	Trace = log.New(outputSink, "[TRACE] ", log.Ldate|log.Ltime)
	Debug = log.New(outputSink, "[DEBUG] ", log.Ldate|log.Ltime)
	Info = log.New(outputSink, "[INFO] ", log.Ldate|log.Ltime)
	Warn = log.New(outputSink, "[WARN] ", log.Ldate|log.Ltime)
	Error = log.New(errorSink, "[ERROR] ", log.Ldate|log.Ltime)

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

// GetSinks returns both the current output and error sink writers
func GetSinks() (outputWriter, errorWriter io.Writer) {
	return outputSink.getWriter(), errorSink.getWriter()
}

// SetOutputSink sets a custom writer for the output sink
func SetOutputSink(w io.Writer) {
	if w == nil {
		fmt.Fprintln(os.Stderr, "[WARN] Attempted to set output sink to nil, using io.Discard instead")
		w = io.Discard // prevent nil writers
	}
	outputSink.setWriter(w)
}

// SetErrorSink sets a custom writer for the error sink
func SetErrorSink(w io.Writer) {
	if w == nil {
		fmt.Fprintln(os.Stderr, "[WARN] Attempted to set error sink to nil, using io.Discard instead")
		w = io.Discard // prevent nil writers
	}
	errorSink.setWriter(w)
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
