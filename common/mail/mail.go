package mail

import (
	"context"
	"errors"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
)

type Mailer interface {
	SendMail(ctx context.Context, to []string, subject string, content string) error
}

var ErrUnsupportedMailer = errors.New("unsupported mailer")

func New(ctx context.Context, config *config.Mailer) (Mailer, error) {
	switch config.Provider {
	case "gmail":
		return newGMailClient(config)
	default:
		return nil, ErrUnsupportedMailer
	}
}
