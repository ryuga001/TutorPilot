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

func (n *Notifier) SendBatchTutorAssignment(ctx context.Context, toEmail string, vars map[string]string) error {
	return n.send(ctx, toEmail, tmplBatchTutorAssignment, vars)
}

func (n *Notifier) SendBatchStudentEnrollment(ctx context.Context, toEmail string, vars map[string]string) error {
	return n.send(ctx, toEmail, tmplBatchStudentEnrollment, vars)
}

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
