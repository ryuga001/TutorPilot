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
	ErrNotFound   = errors.New("not found")
	ErrEmailTaken = errors.New("a tutor with this email already exists")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const tutorCols = `id, customer_id, first_name, last_name, email, phone_no, designation,
	profile_image_key, address_id, created_by, created_at, updated_at`

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

func (r *Repository) Create(ctx context.Context, customerID, createdBy int, req CreateTutorRequest) (*Tutor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var addressID *int
	if req.Address != nil {
		id, err := address.Create(ctx, tx, customerID, *req.Address)
		if err != nil {
			return nil, err
		}
		addressID = &id
	}

	t, err := scanTutor(tx.QueryRow(ctx,
		`INSERT INTO tutors (customer_id, first_name, last_name, email, phone_no, designation, address_id, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+tutorCols,
		customerID, req.FirstName, req.LastName, req.Email, req.PhoneNo, req.Designation, addressID, createdBy))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) List(ctx context.Context, customerID int, search string, limit, offset int) ([]Tutor, int, error) {
	const countQ = `SELECT COUNT(*) FROM tutors
		WHERE customer_id = $1
		  AND ($2 = '' OR first_name ILIKE '%'||$2||'%' OR last_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')`
	var total int
	if err := r.db.QueryRow(ctx, countQ, customerID, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `SELECT ` + tutorCols + ` FROM tutors
		WHERE customer_id = $1
		  AND ($2 = '' OR first_name ILIKE '%'||$2||'%' OR last_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')
		ORDER BY created_at DESC
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
	return scanTutor(r.db.QueryRow(ctx, `SELECT `+tutorCols+` FROM tutors WHERE customer_id = $1 AND id = $2`, customerID, id))
}

func (r *Repository) Update(ctx context.Context, customerID, id int, req UpdateTutorRequest) (*Tutor, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	current, err := scanTutor(tx.QueryRow(ctx, `SELECT `+tutorCols+` FROM tutors WHERE customer_id = $1 AND id = $2 FOR UPDATE`, customerID, id))
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

	t, err := scanTutor(tx.QueryRow(ctx,
		`UPDATE tutors SET first_name = $3, last_name = $4, email = $5, phone_no = $6,
		 designation = $7, address_id = $8, updated_at = now()
		 WHERE customer_id = $1 AND id = $2 RETURNING `+tutorCols,
		customerID, id, req.FirstName, req.LastName, req.Email, req.PhoneNo, req.Designation, addressID))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) SetProfileImage(ctx context.Context, customerID, id int, key string) (*Tutor, error) {
	const q = `UPDATE tutors SET profile_image_key = $3, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + tutorCols
	return scanTutor(r.db.QueryRow(ctx, q, customerID, id, key))
}

func (r *Repository) Delete(ctx context.Context, customerID, id int) (*string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var profileImageKey *string
	var addressID *int
	err = tx.QueryRow(ctx,
		`DELETE FROM tutors WHERE customer_id = $1 AND id = $2 RETURNING profile_image_key, address_id`,
		customerID, id).Scan(&profileImageKey, &addressID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
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
