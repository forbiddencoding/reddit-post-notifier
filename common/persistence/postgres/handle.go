package postgres

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"sync"
	"sync/atomic"
)

type Handle struct {
	db      atomic.Pointer[pgxpool.Pool]
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

	handle := &Handle{}

	handle.db.Store(pool)
	handle.running.Store(true)

	return handle, nil
}

func (h *Handle) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running.Load() {
		h.running.Swap(false)
		db := h.db.Swap(nil)
		if db != nil {
			db.Close()
		}
	}
	return nil
}
