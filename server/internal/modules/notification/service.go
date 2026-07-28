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
	mail         *mailer.Mailer
	templates    *TemplateStore
	verifyURL    string
	portalURLTpl string
	otpTTL       time.Duration
	inviteTTL    time.Duration
	tenantID     int
}

type Config struct {
	VerifyURL string
	// PortalURLTemplate contains {slug}, replaced with the organization's slug so
	// invitation links land on that tenant's own address.
	PortalURLTemplate string
	OTPTTL            time.Duration
	InviteTTL         time.Duration
	SystemTenantID    int
}

func New(mail *mailer.Mailer, templates *TemplateStore, cfg Config) *Notifier {
	return &Notifier{
		mail:         mail,
		templates:    templates,
		verifyURL:    cfg.VerifyURL,
		portalURLTpl: cfg.PortalURLTemplate,
		otpTTL:       cfg.OTPTTL,
		inviteTTL:    cfg.InviteTTL,
		tenantID:     cfg.SystemTenantID,
	}
}

// PortalURL is the browser address of a tenant's portal.
func (n *Notifier) PortalURL(orgSlug string) string {
	return strings.ReplaceAll(n.portalURLTpl, "{slug}", orgSlug)
}

// ActivationURL is where an invited member sets their first password.
func (n *Notifier) ActivationURL(orgSlug, token string) string {
	return fmt.Sprintf("%s/activate?token=%s", strings.TrimRight(n.PortalURL(orgSlug), "/"), url.QueryEscape(token))
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

// MemberInvite is everything an invitation email needs. Both the activation link
// and the temporary password are included: the link is the normal path, the
// password is the fallback when a member never receives the mail.
type MemberInvite struct {
	Name            string
	OrgName         string
	OrgSlug         string
	Email           string
	TempPassword    string
	ActivationToken string
}

func (n *Notifier) SendMemberInvite(ctx context.Context, invite MemberInvite) error {
	return n.send(ctx, invite.Email, tmplMemberInvite, n.inviteVars(invite))
}

// SendMemberCredentialsReset tells a member their credentials were reissued.
func (n *Notifier) SendMemberCredentialsReset(ctx context.Context, invite MemberInvite) error {
	return n.send(ctx, invite.Email, tmplMemberCredentialsReset, n.inviteVars(invite))
}

func (n *Notifier) inviteVars(invite MemberInvite) map[string]string {
	return map[string]string{
		"name":           invite.Name,
		"org_name":       invite.OrgName,
		"email":          invite.Email,
		"temp_password":  invite.TempPassword,
		"portal_url":     n.PortalURL(invite.OrgSlug),
		"activation_url": n.ActivationURL(invite.OrgSlug, invite.ActivationToken),
		"expires_in":     humanizeDuration(n.inviteTTL),
	}
}

func humanizeDuration(d time.Duration) string {
	if hours := int(d.Hours()); hours >= 24 && hours%24 == 0 {
		days := hours / 24
		if days == 1 {
			return "1 day"
		}
		return strconv.Itoa(days) + " days"
	}
	if hours := int(d.Hours()); hours >= 1 {
		if hours == 1 {
			return "1 hour"
		}
		return strconv.Itoa(hours) + " hours"
	}
	return strconv.Itoa(int(d.Minutes())) + " minutes"
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
