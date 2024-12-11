package mysql

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
	Debug        bool
}

type Client struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewClient(ctx context.Context, cfg Config, logger *zap.Logger) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	gormConfig := &gorm.Config{
		Logger: NewGormLogger(logger),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify database connection: %w", err)
	}

	if cfg.Debug {
		db = db.Debug()
	}

	return &Client{
		db:     db,
		logger: logger,
	}, nil
}

func (c *Client) DB() *gorm.DB {
	return c.db
}

func (c *Client) WithContext(ctx context.Context) *gorm.DB {
	return c.db.WithContext(ctx)
}

func (c *Client) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tr := otel.Tracer("gorm")
	ctx, span := tr.Start(ctx, "mysql.transaction")
	defer span.End()

	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.Close()
}

func (c *Client) Health(ctx context.Context) error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
