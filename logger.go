package miio

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

type LogLevel int

const (
	LogLevelFatal LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

func parseLogLevel(lvl LogLevel) (logrus.Level, error) {
	switch lvl {
	case LogLevelDebug:
		return logrus.DebugLevel, nil
	case LogLevelInfo:
		return logrus.InfoLevel, nil
	case LogLevelWarn:
		return logrus.WarnLevel, nil
	case LogLevelError:
		return logrus.ErrorLevel, nil
	case LogLevelFatal:
		return logrus.FatalLevel, nil
	}

	var l logrus.Level
	return l, fmt.Errorf("miio: invalid LogLevel '%d'", lvl)
}

var (
	// LOGGER implementation.
	LOGGER = newLogger()
)

// ILogger defines a logger interface.
type ILogger interface {
	SetLogLevel(lvl LogLevel) error
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

	l := &defaultLogger{}
	l.SetLogLevel(LogLevelDebug)
	return l
}

func (*defaultLogger) SetLogLevel(lvl LogLevel) error {
	l, err := parseLogLevel(lvl)
	if err != nil {
		return err
	}

	logrus.SetLevel(l)

	return nil
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
