package app

import (
	"io"
	"os"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

// All we ever ask of a logger — both backends below implement exactly this and nothing more
type LoggerInterface interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
}

// Global logger and the debug switch — both are set once at startup, then read from everywhere
var Logger LoggerInterface
var DebugLog bool

// Wraps the systemd service logger — used when we run as a real service
type ServiceLogger struct {
	Logger service.Logger
}

// Info message, straight to the journal
func (s *ServiceLogger) Info(args ...interface{}) {
	s.Logger.Info(args...)
}

// Formatted version of the above, same rules apply
func (s *ServiceLogger) Infof(format string, args ...interface{}) {
	s.Logger.Infof(format, args...)
}

// Errors land here — hope nobody is reading these at 3am
func (s *ServiceLogger) Error(args ...interface{}) {
	s.Logger.Error(args...)
}

// Formatted flavor of the one above
func (s *ServiceLogger) Errorf(format string, args ...interface{}) {
	s.Logger.Errorf(format, args...)
}

// A warning for the log. Yes, it goes to journal level "warning" for some reason
func (s *ServiceLogger) Warn(args ...interface{}) {
	s.Logger.Warning(args...)
}

// Formatted warning, same story
func (s *ServiceLogger) Warnf(format string, args ...interface{}) {
	s.Logger.Warningf(format, args...)
}

// Stays silent unless DEBUG_LOG is on — that's a feature, not a bug
func (s *ServiceLogger) Debug(args ...interface{}) {
	if DebugLog {
		s.Logger.Info(args...)
	}
}

// Formatted variant of the silence above, same rules
func (s *ServiceLogger) Debugf(format string, args ...interface{}) {
	if DebugLog {
		s.Logger.Infof(format, args...)
	}
}

// logrus wrapper for when we run from a plain terminal (dev mode, mostly)
type ConsoleLogger struct {
	Logger *logrus.Logger
}

// Info to stdout with colors — looks nice in a real terminal
func (c *ConsoleLogger) Info(args ...interface{}) {
	c.Logger.Info(args...)
}

// Formatted version of that one above
func (c *ConsoleLogger) Infof(format string, args ...interface{}) {
	c.Logger.Infof(format, args...)
}

// Loud errors — the ones you actually want to see on your screen
func (c *ConsoleLogger) Error(args ...interface{}) {
	c.Logger.Error(args...)
}

// Formatted flavor again, sorry about that
func (c *ConsoleLogger) Errorf(format string, args ...interface{}) {
	c.Logger.Errorf(format, args...)
}

// A warning for humans watching the logs live
func (c *ConsoleLogger) Warn(args ...interface{}) {
	c.Logger.Warn(args...)
}

// Formatted warning
func (c *ConsoleLogger) Warnf(format string, args ...interface{}) {
	c.Logger.Warnf(format, args...)
}

// Hidden unless DEBUG_LOG=true, same as the journal one
func (c *ConsoleLogger) Debug(args ...interface{}) {
	if DebugLog {
		c.Logger.Info(args...)
	}
}

// Formatted variant of that one above, same silence rules
func (c *ConsoleLogger) Debugf(format string, args ...interface{}) {
	if DebugLog {
		c.Logger.Infof(format, args...)
	}
}

// Build the console logger: colors on, timestamps on, Info level and up only
func NewConsoleLogger() *ConsoleLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	l.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	l.SetLevel(logrus.InfoLevel)
	l.SetOutput(os.Stdout)

	return &ConsoleLogger{Logger: l}
}

// Take the service's own logger and wrap it so it speaks our interface
func NewServiceLogger(svcLogger service.Logger) *ServiceLogger {
	return &ServiceLogger{Logger: svcLogger}
}

// Install the active logger for the whole process — call this before anything logs
func InitLogger(logger LoggerInterface) {
	Logger = logger
}
