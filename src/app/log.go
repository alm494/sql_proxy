package app

import (
	"io"
	"os"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
)

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

var Logger LoggerInterface
var DebugLog bool

// service logger:
type ServiceLogger struct {
	Logger service.Logger
}

func (s *ServiceLogger) Info(args ...interface{}) {
	s.Logger.Info(args...)
}

func (s *ServiceLogger) Infof(format string, args ...interface{}) {
	s.Logger.Infof(format, args...)
}

func (s *ServiceLogger) Error(args ...interface{}) {
	s.Logger.Error(args...)
}

func (s *ServiceLogger) Errorf(format string, args ...interface{}) {
	s.Logger.Errorf(format, args...)
}

func (s *ServiceLogger) Warn(args ...interface{}) {
	s.Logger.Warning(args...)
}

func (s *ServiceLogger) Warnf(format string, args ...interface{}) {
	s.Logger.Warningf(format, args...)
}

func (s *ServiceLogger) Debug(args ...interface{}) {
	if DebugLog {
		s.Logger.Info(args...)
	}
}

func (s *ServiceLogger) Debugf(format string, args ...interface{}) {
	if DebugLog {
		s.Logger.Infof(format, args...)
	}
}

// Console logger:

type ConsoleLogger struct {
	Logger *logrus.Logger
}

func (c *ConsoleLogger) Info(args ...interface{}) {
	c.Logger.Info(args...)
}

func (c *ConsoleLogger) Infof(format string, args ...interface{}) {
	c.Logger.Infof(format, args...)
}

func (c *ConsoleLogger) Error(args ...interface{}) {
	c.Logger.Error(args...)
}

func (c *ConsoleLogger) Errorf(format string, args ...interface{}) {
	c.Logger.Errorf(format, args...)
}

func (c *ConsoleLogger) Warn(args ...interface{}) {
	c.Logger.Warn(args...)
}

func (c *ConsoleLogger) Warnf(format string, args ...interface{}) {
	c.Logger.Warnf(format, args...)
}

func (c *ConsoleLogger) Debug(args ...interface{}) {
	if DebugLog {
		c.Logger.Info(args...)
	}
}

func (c *ConsoleLogger) Debugf(format string, args ...interface{}) {
	if DebugLog {
		c.Logger.Infof(format, args...)
	}
}

// Create logger variants
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

func NewServiceLogger(svcLogger service.Logger) *ServiceLogger {
	return &ServiceLogger{Logger: svcLogger}
}

// Init
func InitLogger(logger LoggerInterface) {
	Logger = logger
}
