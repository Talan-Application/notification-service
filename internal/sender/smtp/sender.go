package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/Talan-Application/notification-service/internal/config"
	"github.com/Talan-Application/notification-service/internal/sender"
)

type Sender struct {
	cfg config.SMTPConfig
}

func NewSender(cfg config.SMTPConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) Send(_ context.Context, msg sender.Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	from := s.formatFrom()
	raw := buildMessage(from, msg.To, msg.Subject, msg.Body, msg.HTML)

	if s.cfg.TLS {
		return s.sendWithTLS(addr, msg.To, raw)
	}

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	return smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, []byte(raw))
}

func (s *Sender) formatFrom() string {
	if s.cfg.FromName != "" {
		return fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.From)
	}
	return s.cfg.From
}

func (s *Sender) sendWithTLS(addr, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	defer w.Close()

	_, err = fmt.Fprint(w, msg)
	return err
}

const boundary = "==TALAN_NOTIFICATION_BOUNDARY=="

func buildMessage(from, to, subject, body, html string) string {
	if html == "" {
		return fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
			from, to, subject, body,
		)
	}
	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n\r\n--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n\r\n--%s--\r\n",
		from, to, subject, boundary,
		boundary, body,
		boundary, html,
		boundary,
	)
}
