package mail

import (
	"context"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"github.com/resend/resend-go/v2"
)

type resendClient struct {
	config *config.Mailer
	client *resend.Client
}

func newResendClient(config *config.Mailer) (Mailer, error) {
	return &resendClient{
		client: resend.NewClient(config.Resend.APIKey),
		config: config,
	}, nil
}

func (r *resendClient) SendMail(ctx context.Context, to []string, subject string, content string) error {
	params := &resend.SendEmailRequest{
		From:    r.config.SenderEmail,
		To:      to,
		Html:    content,
		Subject: subject,
	}

	sent, err := r.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return err
	}
	_ = sent
	return nil
}
