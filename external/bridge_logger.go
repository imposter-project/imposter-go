package external

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// bridgeLogger implements hclog.Logger, forwarding all output through the
// main application logger so that external-plugin messages use the same
// format as the rest of the application.
type bridgeLogger struct {
	name        string
	impliedArgs []interface{}
}

func newBridgeLogger() hclog.Logger {
	return &bridgeLogger{}
}

// formatMsg builds a single log line from the structured hclog arguments.
func (b *bridgeLogger) formatMsg(msg string, args ...interface{}) string {
	all := append(b.impliedArgs, args...)

	var sb strings.Builder
	if b.name != "" {
		sb.WriteString(b.name)
		sb.WriteString(": ")
	}
	sb.WriteString(msg)

	// Append key=value pairs, skipping "timestamp" as the main logger
	// already prepends one.
	for i := 0; i+1 < len(all); i += 2 {
		if all[i] == "timestamp" {
			continue
		}
		sb.WriteString(fmt.Sprintf(" %v=%v", all[i], all[i+1]))
	}
	return sb.String()
}

func (b *bridgeLogger) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace:
		b.Trace(msg, args...)
	case hclog.Debug:
		b.Debug(msg, args...)
	case hclog.Info:
		b.Info(msg, args...)
	case hclog.Warn:
		b.Warn(msg, args...)
	case hclog.Error:
		b.Error(msg, args...)
	}
}

func (b *bridgeLogger) Trace(msg string, args ...interface{}) {
	logger.Tracef("%s", b.formatMsg(msg, args...))
}

func (b *bridgeLogger) Debug(msg string, args ...interface{}) {
	logger.Debugf("%s", b.formatMsg(msg, args...))
}

func (b *bridgeLogger) Info(msg string, args ...interface{}) {
	logger.Infof("%s", b.formatMsg(msg, args...))
}

func (b *bridgeLogger) Warn(msg string, args ...interface{}) {
	logger.Warnf("%s", b.formatMsg(msg, args...))
}

func (b *bridgeLogger) Error(msg string, args ...interface{}) {
	logger.Errorf("%s", b.formatMsg(msg, args...))
}

func (b *bridgeLogger) IsTrace() bool { return logger.IsTraceEnabled() }
func (b *bridgeLogger) IsDebug() bool { return logger.IsDebugEnabled() }
func (b *bridgeLogger) IsInfo() bool  { return logger.IsInfoEnabled() }
func (b *bridgeLogger) IsWarn() bool  { return logger.IsWarnEnabled() }
func (b *bridgeLogger) IsError() bool { return logger.IsErrorEnabled() }

func (b *bridgeLogger) ImpliedArgs() []interface{} {
	return b.impliedArgs
}

func (b *bridgeLogger) With(args ...interface{}) hclog.Logger {
	newArgs := make([]interface{}, len(b.impliedArgs)+len(args))
	copy(newArgs, b.impliedArgs)
	copy(newArgs[len(b.impliedArgs):], args)
	return &bridgeLogger{
		name:        b.name,
		impliedArgs: newArgs,
	}
}

func (b *bridgeLogger) Name() string {
	return b.name
}

func (b *bridgeLogger) Named(name string) hclog.Logger {
	newName := name
	if b.name != "" {
		newName = b.name + "." + name
	}
	return &bridgeLogger{
		name:        newName,
		impliedArgs: b.impliedArgs,
	}
}

func (b *bridgeLogger) ResetNamed(name string) hclog.Logger {
	return &bridgeLogger{
		name:        name,
		impliedArgs: b.impliedArgs,
	}
}

func (b *bridgeLogger) SetLevel(level hclog.Level) {
	// Level is controlled by the main logger; ignore.
}

func (b *bridgeLogger) GetLevel() hclog.Level {
	return getHcLogLevel()
}

func (b *bridgeLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(b.StandardWriter(opts), "", 0)
}

func (b *bridgeLogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return &bridgeWriter{logger: b}
}

// bridgeWriter implements io.Writer for StandardWriter/StandardLogger.
type bridgeWriter struct {
	logger *bridgeLogger
}

func (w *bridgeWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}
