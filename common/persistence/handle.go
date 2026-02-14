package persistence

import (
	"context"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Handle struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, config *config.Persistence) (*Handle, error) {
	conf, err := pgxpool.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	))
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Handle{
		db: pool,
	}, nil
}

func (h *Handle) Close() error {
	h.db.Close()
	return nil
}
