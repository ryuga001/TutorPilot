package notification

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"time"

	tmpl "tutorpilot/internal/modules/notification"
	"tutorpilot/internal/modules/notification/mailer"
	repository "tutorpilot/internal/modules/notification/repository"
)

type Notifier struct {
	mail      *mailer.Mailer
	templates *repository.TemplateStore
	verifyURL string
	signInURL string
	otpTTL    time.Duration
	tenantID  int
}

type Config struct {
	VerifyURL string

	SignInURL      string
	OTPTTL         time.Duration
	SystemTenantID int
}

func New(mail *mailer.Mailer, templates *repository.TemplateStore, cfg Config) *Notifier {
	return &Notifier{
		mail:      mail,
		templates: templates,
		verifyURL: cfg.VerifyURL,
		signInURL: cfg.SignInURL,
		otpTTL:    cfg.OTPTTL,
		tenantID:  cfg.SystemTenantID,
	}
}

func (n *Notifier) SendTemplate(ctx context.Context, toEmail, templateName string, vars map[string]string) error {
	tmpl, err := n.templates.Get(ctx, n.tenantID, templateName)
	if err != nil {
		return err
	}

	subject := fill(tmpl.Subject, vars, false)
	body := fill(tmpl.Body, vars, true)

	if missing := orphans(tmpl.Subject+tmpl.Body, vars); len(missing) > 0 {
		log.Printf("notification: template %q references variables no caller supplied: %v (they render literally)",
			templateName, missing)
	}
	return n.mail.Send(ctx, toEmail, subject, body)
}

func (n *Notifier) SendEmailVerification(ctx context.Context, toEmail, otp string) error {
	link := fmt.Sprintf("%s?email=%s", n.verifyURL, url.QueryEscape(toEmail))
	return n.SendTemplate(ctx, toEmail, tmpl.TmplEmailVerification, map[string]string{
		"name":           "there",
		"otp":            otp,
		"expiry_minutes": strconv.Itoa(int(n.otpTTL.Minutes())),
		"verify_link":    link,
	})
}

func (n *Notifier) SendPasswordReset(ctx context.Context, name, toEmail, otp string) error {
	return n.SendTemplate(ctx, toEmail, tmpl.TmplPasswordReset, map[string]string{
		"name":           name,
		"otp":            otp,
		"expiry_minutes": strconv.Itoa(int(n.otpTTL.Minutes())),
	})
}

func (n *Notifier) SendBatchTutorAssignment(ctx context.Context, toEmail string, vars map[string]string) error {
	return n.SendTemplate(ctx, toEmail, tmpl.TmplBatchTutorAssignment, vars)
}

func (n *Notifier) SendBatchStudentEnrollment(ctx context.Context, toEmail string, vars map[string]string) error {
	return n.SendTemplate(ctx, toEmail, tmpl.TmplBatchStudentEnrollment, vars)
}

var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}`)

func fill(s string, vars map[string]string, escapeHTML bool) string {
	return placeholderRe.ReplaceAllStringFunc(s, func(match string) string {
		v, ok := vars[match[2:len(match)-2]]
		if !ok {
			return match
		}
		if escapeHTML {
			return html.EscapeString(v)
		}
		return v
	})
}

func orphans(s string, vars map[string]string) []string {
	var missing []string
	seen := map[string]bool{}
	for _, m := range placeholderRe.FindAllStringSubmatch(s, -1) {
		key := m[1]
		if _, ok := vars[key]; ok || seen[key] {
			continue
		}
		seen[key] = true
		missing = append(missing, key)
	}
	sort.Strings(missing)
	return missing
}
