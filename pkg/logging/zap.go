package logging

import (
	"mempool/pkg/util"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger LoggingSystem = nil

type LoggingSystem interface {
	Sugar() *zap.SugaredLogger
	Named(string) *zap.Logger
	WithOptions(...zap.Option) *zap.Logger
	With(...zap.Field) *zap.Logger
	Check(zapcore.Level, string) *zapcore.CheckedEntry
	Debug(string, ...zap.Field)
	Info(string, ...zap.Field)
	Warn(string, ...zap.Field)
	Error(string, ...zap.Field)
	DPanic(string, ...zap.Field)
	Panic(string, ...zap.Field)
	Fatal(string, ...zap.Field)
	Sync() error
	Core() zapcore.Core
}

func Logger() (LoggingSystem, error) {
	var err error
	if logger == nil {
		if util.DevelopmentEnvironment() {
			logger, err = zap.NewDevelopment(zap.AddStacktrace(zapcore.ErrorLevel))
		} else {
			logger, err = zap.NewProduction()
		}
	}
	return logger.Named("mempool"), err
}
