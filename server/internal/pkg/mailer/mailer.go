package mailer

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host string
	port string
	from string
}

func New(host, port, from string) *Mailer {
	return &Mailer{host: host, port: port, from: from}
}

func (m *Mailer) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)

	envelopeFrom := m.from
	if parsed, err := mail.ParseAddress(m.from); err == nil {
		envelopeFrom = parsed.Address
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("From: %s\r\n", m.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return smtp.SendMail(addr, nil, envelopeFrom, []string{to}, []byte(msg.String()))
}
