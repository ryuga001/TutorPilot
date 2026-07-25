package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const userCols = `id, name, email, password_hash, salt, email_verified, created_at, updated_at`

func scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Salt,
		&u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) CreateUser(ctx context.Context, name, email, passwordHash, salt string) (*User, error) {
	const q = `
		INSERT INTO users (name, email, password_hash, salt, email_verified)
		VALUES ($1, $2, $3, $4, true)
		RETURNING ` + userCols
	return scanUser(r.db.QueryRow(ctx, q, name, email, passwordHash, salt))
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email))
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

func (r *Repository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	const q = `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`
	_, err := r.db.Exec(ctx, q, passwordHash, userID)
	return err
}
