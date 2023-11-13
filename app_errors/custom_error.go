package app_errors

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Wrap(msg string, err error) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func LogAndError(msg string, err error, logger *zap.Logger, fields ...zapcore.Field) error {
	fields = append(fields, zap.Error(err))
	logger.Error(msg, fields...)

	return Wrap(msg, err)
}
