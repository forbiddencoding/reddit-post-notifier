package postgres

import (
	"context"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"sync"
	"sync/atomic"
	"time"
)

type Handle struct {
	dbPtr   atomic.Pointer[pgxpool.Pool]
	running atomic.Bool
	mu      sync.Mutex
}

func NewHandle(ctx context.Context, config *config.Persistence) (*Handle, error) {
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

	handle := &Handle{}

	handle.dbPtr.Store(pool)
	handle.running.Store(true)

	return handle, nil
}

func (h *Handle) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running.Load() {
		h.running.Swap(false)
		db := h.dbPtr.Swap(nil)
		if db != nil {
			db.Close()
		}
	}
	return nil
}

func (h *Handle) db() (*pgxpool.Pool, error) {
	if db := h.dbPtr.Load(); db != nil {
		return db, nil
	}

	return nil, errors.New("no usable database connection found")
}
