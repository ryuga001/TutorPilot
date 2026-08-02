package notification

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SystemTenantID = 1

var ErrTemplateNotFound = errors.New("email template not found")

type Template struct {
	Subject string
	Body    string
}

type TemplateStore struct {
	db *pgxpool.Pool
}

func NewTemplateStore(db *pgxpool.Pool) *TemplateStore {
	return &TemplateStore{db: db}
}

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
