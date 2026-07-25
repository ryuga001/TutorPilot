package notification

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SystemTenantID owns the shared system email templates (verification, reset).
const SystemTenantID = 1

// ErrTemplateNotFound is returned when no email_templates row matches.
var ErrTemplateNotFound = errors.New("email template not found")

// Template is a saved email template's subject and HTML body.
type Template struct {
	Subject string
	Body    string
}

// TemplateStore reads email templates from the email_templates table.
type TemplateStore struct {
	db *pgxpool.Pool
}

func NewTemplateStore(db *pgxpool.Pool) *TemplateStore {
	return &TemplateStore{db: db}
}

// Get returns the subject and body of a tenant's template, or
// ErrTemplateNotFound if no such row exists.
func (s *TemplateStore) Get(ctx context.Context, customerID int, name string) (*Template, error) {
	var t Template
	err := s.db.QueryRow(ctx,
		`SELECT subject, body FROM email_templates WHERE customer_id = $1 AND name = $2`,
		customerID, name,
	).Scan(&t.Subject, &t.Body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
