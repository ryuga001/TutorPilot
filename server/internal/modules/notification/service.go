package notification

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tutorpilot/internal/pkg/mailer"
)

type Notifier struct {
	mail      *mailer.Mailer
	templates *TemplateStore
	verifyURL string
	otpTTL    time.Duration
	tenantID  int
}

func New(mail *mailer.Mailer, templates *TemplateStore, verifyURL string, otpTTL time.Duration, systemTenantID int) *Notifier {
	return &Notifier{
		mail:      mail,
		templates: templates,
		verifyURL: verifyURL,
		otpTTL:    otpTTL,
		tenantID:  systemTenantID,
	}
}

func (n *Notifier) SendEmailVerification(ctx context.Context, toEmail, otp string) error {
	link := fmt.Sprintf("%s?email=%s", n.verifyURL, url.QueryEscape(toEmail))
	return n.send(ctx, toEmail, tmplEmailVerification, map[string]string{
		"name":           "there",
		"otp":            otp,
		"expiry_minutes": strconv.Itoa(int(n.otpTTL.Minutes())),
		"verify_link":    link,
	})
}

func (n *Notifier) SendPasswordReset(ctx context.Context, name, toEmail, otp string) error {
	return n.send(ctx, toEmail, tmplPasswordReset, map[string]string{
		"name":           name,
		"otp":            otp,
		"expiry_minutes": strconv.Itoa(int(n.otpTTL.Minutes())),
	})
}

// send loads the saved template (subject + HTML body), substitutes {{var}}
// placeholders and mails it. If the template row is missing, the error is
// returned to the caller — there is no fallback.
func (n *Notifier) send(ctx context.Context, toEmail, name string, vars map[string]string) error {
	tmpl, err := n.templates.Get(ctx, n.tenantID, name)
	if err != nil {
		return err
	}
	subject := fillPlaceholders(tmpl.Subject, vars)
	body := fillPlaceholders(tmpl.Body, vars)
	return n.mail.Send(toEmail, subject, body)
}

func fillPlaceholders(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
