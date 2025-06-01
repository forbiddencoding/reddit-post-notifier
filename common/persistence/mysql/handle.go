package mysql

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"sync"
	"sync/atomic"
	"time"
)

type Handle struct {
	db      atomic.Pointer[sqlx.DB]
	running atomic.Bool
	mu      sync.Mutex
}

func NewHandle(ctx context.Context, config *config.Persistence) (*Handle, error) {
	db, err := sqlx.Open("mysql", config.DSN)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	handle := &Handle{}

	handle.db.Store(db)
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
			return db.Close()
		}
	}
	return nil
}
