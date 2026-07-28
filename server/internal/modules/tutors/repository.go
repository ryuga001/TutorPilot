package tutors

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/address"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmailTaken  = errors.New("a tutor with this email already exists")
	ErrNoTutorRole = errors.New("this organization has no Tutor role configured")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// tutorCols/tutorFrom join the tutor's extra fields to the identity fields that
// now live on dashboard_users. t.dashboard_user_id is the id everywhere else in
// the system knows the tutor by (JWT uid, batch_tutors.tutor_id, ...).
const tutorCols = `t.dashboard_user_id, du.customer_id, du.first_name, du.last_name, du.email,
	t.phone_no, t.designation, t.profile_image_key, t.address_id, t.created_by,
	t.created_at, t.updated_at`

const tutorFrom = `FROM tutors t JOIN dashboard_users du ON du.id = t.dashboard_user_id`

func scanTutor(row pgx.Row) (*Tutor, error) {
	t := &Tutor{}
	err := row.Scan(&t.ID, &t.CustomerID, &t.FirstName, &t.LastName, &t.Email, &t.PhoneNo,
		&t.Designation, &t.ProfileImageKey, &t.AddressID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateInput carries an already-generated, already-hashed password: the
// repository only persists, the service owns password generation (pkg/security)
// so hashing details stay out of SQL.
type CreateInput struct {
	CreateTutorRequest
	PasswordHash string
	PasswordSalt string
}

// Create inserts the dashboard_users row (identity + credentials) and the tutors
// row (extras) in one transaction. A tutor is never created without a login —
// their name and email live on dashboard_users, so there is nothing to create
// without it.
func (r *Repository) Create(ctx context.Context, customerID, createdBy int, in CreateInput) (*Tutor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var roleID int
	err = tx.QueryRow(ctx,
		`SELECT id FROM roles WHERE customer_id = $1 AND name = 'Tutor'`, customerID,
	).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoTutorRole
	}
	if err != nil {
		return nil, err
	}

	var userID int
	err = tx.QueryRow(ctx,
		`INSERT INTO dashboard_users
			(customer_id, role_id, email, password_hash, password_salt, first_name, last_name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		customerID, roleID, in.Email, in.PasswordHash, in.PasswordSalt, in.FirstName, in.LastName,
	).Scan(&userID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	var addressID *int
	if in.Address != nil {
		id, err := address.Create(ctx, tx, customerID, *in.Address)
		if err != nil {
			return nil, err
		}
		addressID = &id
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO tutors (dashboard_user_id, phone_no, designation, address_id, created_by)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, in.PhoneNo, in.Designation, addressID, createdBy,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, customerID, userID)
}

func (r *Repository) List(ctx context.Context, customerID int, search string, limit, offset int) ([]Tutor, int, error) {
	const countQ = `SELECT COUNT(*) ` + tutorFrom + `
		WHERE du.customer_id = $1
		  AND ($2 = '' OR du.first_name ILIKE '%'||$2||'%' OR du.last_name ILIKE '%'||$2||'%' OR du.email ILIKE '%'||$2||'%')`
	var total int
	if err := r.db.QueryRow(ctx, countQ, customerID, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `SELECT ` + tutorCols + ` ` + tutorFrom + `
		WHERE du.customer_id = $1
		  AND ($2 = '' OR du.first_name ILIKE '%'||$2||'%' OR du.last_name ILIKE '%'||$2||'%' OR du.email ILIKE '%'||$2||'%')
		ORDER BY t.created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, q, customerID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Tutor, 0)
	for rows.Next() {
		t := Tutor{}
		if err := rows.Scan(&t.ID, &t.CustomerID, &t.FirstName, &t.LastName, &t.Email, &t.PhoneNo,
			&t.Designation, &t.ProfileImageKey, &t.AddressID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, customerID, id int) (*Tutor, error) {
	q := `SELECT ` + tutorCols + ` ` + tutorFrom + ` WHERE du.customer_id = $1 AND t.dashboard_user_id = $2`
	return scanTutor(r.db.QueryRow(ctx, q, customerID, id))
}

func (r *Repository) Update(ctx context.Context, customerID, id int, req UpdateTutorRequest) (*Tutor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	current, err := scanTutor(tx.QueryRow(ctx,
		`SELECT `+tutorCols+` `+tutorFrom+`
		 WHERE du.customer_id = $1 AND t.dashboard_user_id = $2 FOR UPDATE`,
		customerID, id))
	if err != nil {
		return nil, err
	}

	addressID := current.AddressID
	if req.Address != nil {
		if addressID != nil {
			if err := address.Update(ctx, tx, customerID, *addressID, *req.Address); err != nil {
				return nil, err
			}
		} else {
			newID, err := address.Create(ctx, tx, customerID, *req.Address)
			if err != nil {
				return nil, err
			}
			addressID = &newID
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE dashboard_users SET first_name = $3, last_name = $4, email = $5
		 WHERE id = $2 AND customer_id = $1`,
		customerID, id, req.FirstName, req.LastName, req.Email,
	); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	if _, err := tx.Exec(ctx,
		`UPDATE tutors SET phone_no = $2, designation = $3, address_id = $4, updated_at = now()
		 WHERE dashboard_user_id = $1`,
		id, req.PhoneNo, req.Designation, addressID,
	); err != nil {
		return nil, err
	}

	t, err := scanTutor(tx.QueryRow(ctx,
		`SELECT `+tutorCols+` `+tutorFrom+` WHERE du.customer_id = $1 AND t.dashboard_user_id = $2`,
		customerID, id))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) SetProfileImage(ctx context.Context, customerID, id int, key string) (*Tutor, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE tutors SET profile_image_key = $2, updated_at = now()
		 WHERE dashboard_user_id = $1
		   AND EXISTS (SELECT 1 FROM dashboard_users du WHERE du.id = $1 AND du.customer_id = $3)`,
		id, key, customerID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, customerID, id)
}

// Delete removes the tutor's login, which cascades to remove the tutors extras
// row too (tutors.dashboard_user_id REFERENCES dashboard_users(id) ON DELETE
// CASCADE). Their address, if any, is cleaned up alongside.
func (r *Repository) Delete(ctx context.Context, customerID, id int) (*string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var profileImageKey *string
	var addressID *int
	err = tx.QueryRow(ctx,
		`SELECT profile_image_key, address_id FROM tutors WHERE dashboard_user_id = $1`, id,
	).Scan(&profileImageKey, &addressID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM dashboard_users WHERE id = $1 AND customer_id = $2`, id, customerID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	if addressID != nil {
		if err := address.Delete(ctx, tx, customerID, *addressID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return profileImageKey, nil
}
