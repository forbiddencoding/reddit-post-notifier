package mail

import (
	"context"
	"fmt"
	"github.com/forbiddencoding/reddit-post-notifier/common/config"
	"net/smtp"
	"strings"
)

type gMailClient struct {
	config *config.Mailer
}

func newGMailClient(config *config.Mailer) (Mailer, error) {
	return &gMailClient{
		config: config,
	}, nil
}

func (g *gMailClient) SendMail(ctx context.Context, to []string, subject string, content string) error {
	auth := smtp.PlainAuth("", g.config.SenderEmail, g.config.GMail.AppPassword, "smtp.gmail.com")

	headers := make(map[string]string)
	headers["From"] = g.config.SenderEmail
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = fmt.Sprintf("%s %s", g.config.SubjectPrefix, subject)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = `text/html; charset="UTF-8"`

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + content

	if err := smtp.SendMail("smtp.gmail.com:587", auth, g.config.SenderEmail, to, []byte(message)); err != nil {
		return err
	}
	return nil
}
