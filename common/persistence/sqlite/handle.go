package sqlite

import (
	"context"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/tursodatabase/go-libsql"
	"sync"
	"sync/atomic"
)

type Handle struct {
	dbPtr   atomic.Pointer[sqlx.DB]
	running atomic.Bool
	mu      sync.Mutex
}

func NewHandle(ctx context.Context, config *config.Persistence) (*Handle, error) {
	db, err := sqlx.Open("libsql", config.DSN)
	if err != nil {
		return nil, err
	}

	handle := &Handle{}

	handle.dbPtr.Store(db)
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
			return db.Close()
		}
	}
	return nil
}

func (h *Handle) db() (*sqlx.DB, error) {
	if db := h.dbPtr.Load(); db != nil {
		return db, nil
	}

	return nil, errors.New("no usable database connection found")
}
