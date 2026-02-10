package persistence

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Handle struct {
	db *pgxpool.Pool
}

func New(ctx context.Context, config *config.Persistence) (*Handle, error) {
	conf, err := pgxpool.ParseConfig(config.DSN)
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

func (h *Handle) Close(ctx context.Context) error {
	h.db.Close()
	return nil
}
