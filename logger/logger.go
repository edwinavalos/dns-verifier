package logger

import (
	"github.com/rs/zerolog"
)

type Logger struct {
	zerolog.Logger
}

func (logger Logger) Debugf(format string, args ...interface{}) {
	logger.Debug().Msgf(format, args...)
}

func (logger Logger) Infof(format string, args ...interface{}) {
	logger.Info().Msgf(format, args...)
}

func (logger Logger) Warnf(format string, args ...interface{}) {
	logger.Warn().Msgf(format, args...)
}

func (logger Logger) Errorf(format string, args ...interface{}) {
	logger.Error().Msgf(format, args...)
}

func (logger Logger) Fatalf(format string, args ...interface{}) {
	logger.Error().Msgf(format, args...)
}
