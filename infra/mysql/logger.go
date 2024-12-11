package mysql

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"time"
)

type GormLogger struct {
	logger        *zap.Logger
	SlowThreshold time.Duration
}

func NewGormLogger(logger *zap.Logger) *GormLogger {
	return &GormLogger{
		logger:        logger,
		SlowThreshold: 200 * time.Millisecond,
	}
}

func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Info(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warn(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Error(fmt.Sprintf(msg, data...))
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		fields = append(fields, zap.Error(err))
		l.logger.Error("sql query error", fields...)
		return
	}

	if elapsed > l.SlowThreshold {
		l.logger.Warn("slow sql query", fields...)
		return
	}

	l.logger.Debug("sql query", fields...)
}
