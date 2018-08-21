package miio

import (
	"os"

	"github.com/sirupsen/logrus"
)

var (
	// LOGGER implementation.
	LOGGER = newLogger()
)

// ILogger defines a logger interface.
type ILogger interface {
	Debug(format string, v ...interface{})
	Info(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
	Fatal(format string, v ...interface{})
}

// Default logger.
type defaultLogger struct {
}

// Creates a new default logger.
func newLogger() ILogger {
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.DebugLevel)
	return &defaultLogger{}
}

// Debug prints debug lvl message.
func (*defaultLogger) Debug(format string, v ...interface{}) {
	logrus.Debugf(format, v...)
}

// Info prints info lvl message.
func (*defaultLogger) Info(format string, v ...interface{}) {
	logrus.Infof(format, v...)
}

// Warn prints warn lvl message.
func (*defaultLogger) Warn(format string, v ...interface{}) {
	logrus.Warnf(format, v...)
}

// Error prints error lvl message.
func (*defaultLogger) Error(format string, v ...interface{}) {
	logrus.Errorf(format, v...)
}

// Fatal prints fatal lvl message.
func (*defaultLogger) Fatal(format string, v ...interface{}) {
	logrus.Fatalf(format, v...)
}
