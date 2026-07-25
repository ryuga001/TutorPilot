package notification

import (
	"fmt"
	"net/url"
	"time"

	"workflow/internal/pkg/mailer"
)

type Notifier struct {
	mail      *mailer.Mailer
	verifyURL string
	otpTTL    time.Duration
}

func New(mail *mailer.Mailer, verifyURL string, otpTTL time.Duration) *Notifier {
	return &Notifier{mail: mail, verifyURL: verifyURL, otpTTL: otpTTL}
}

func (n *Notifier) SendEmailVerification(toEmail, otp string) error {
	link := fmt.Sprintf("%s?email=%s", n.verifyURL, url.QueryEscape(toEmail))
	body := fmt.Sprintf(tmplEmailVerification, "there", otp, link, link, int(n.otpTTL.Minutes()))
	return n.mail.Send(toEmail, subjectEmailVerification, body)
}

func (n *Notifier) SendPasswordReset(name, toEmail, otp string) error {
	body := fmt.Sprintf(tmplPasswordReset, name, otp, int(n.otpTTL.Minutes()))
	return n.mail.Send(toEmail, subjectPasswordReset, body)
}
