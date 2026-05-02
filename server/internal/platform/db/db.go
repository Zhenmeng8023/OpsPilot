package db

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"opspilot/server/internal/config"
)

func Open(ctx context.Context, cfg config.DatabaseConfig) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, errors.New("DATABASE_DSN is empty")
	}

	var (
		handle *gorm.DB
		err    error
	)
	switch cfg.Driver {
	case "postgres", "postgresql":
		handle, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
	case "mysql":
		handle, err = gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
	default:
		return nil, errors.New("unsupported database driver")
	}
	if err != nil {
		return nil, err
	}

	sqlDB, err := handle.DB()
	if err != nil {
		return nil, err
	}
	if err := ping(ctx, sqlDB); err != nil {
		return nil, err
	}

	return handle, nil
}

func ping(ctx context.Context, db *sql.DB) error {
	done := make(chan error, 1)
	go func() {
		done <- db.PingContext(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
