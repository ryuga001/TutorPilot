package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"
)

var ErrInvalidRecipient = errors.New("invalid recipient address")

type Mailer struct {
	host    string
	port    string
	from    string
	timeout time.Duration
	auth    smtp.Auth
}

type Config struct {
	Host string
	Port string
	From string

	Timeout time.Duration

	Username string
	Password string
}

func New(cfg Config) *Mailer {
	m := &Mailer{host: cfg.Host, port: cfg.Port, from: cfg.From, timeout: cfg.Timeout}
	if m.timeout <= 0 {
		m.timeout = 10 * time.Second
	}
	if cfg.Username != "" {
		m.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return m
}

func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidRecipient, to)
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	addr := net.JoinHostPort(m.host, m.port)
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	c, err := smtp.NewClient(conn, m.host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer func() { _ = c.Close() }()

	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: m.host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if m.auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(m.auth); err != nil {
				return err
			}
		}
	}

	envelopeFrom := m.from
	if parsed, err := mail.ParseAddress(m.from); err == nil {
		envelopeFrom = parsed.Address
	}
	if err := c.Mail(envelopeFrom); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}

	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(m.message(to, subject, body))); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func (m *Mailer) message(to, subject, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", m.from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInvalidRecipient) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if code, ok := smtpCode(err); ok {
		switch code {
		case 421, 450, 451, 452, 454, 455, 471:
			return true
		case 550, 551, 552, 553, 554:

			return false
		}
		if code >= 500 && code < 600 {
			return false
		}
		return true
	}
	return true
}

func smtpCode(err error) (int, bool) {
	var protoErr *textproto.Error
	if errors.As(err, &protoErr) {
		return protoErr.Code, true
	}
	return 0, false
}
