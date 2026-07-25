package students

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
	ErrEmailTaken = errors.New("a student with this email already exists")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const studentCols = `id, customer_id, first_name, last_name, email, phone_no,
	profile_image_key, address_id, created_by, created_at, updated_at`

func scanStudent(row pgx.Row) (*Student, error) {
	s := &Student{}
	err := row.Scan(&s.ID, &s.CustomerID, &s.FirstName, &s.LastName, &s.Email, &s.PhoneNo,
		&s.ProfileImageKey, &s.AddressID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *Repository) Create(ctx context.Context, customerID, createdBy int, req CreateStudentRequest) (*Student, error) {
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

	s, err := scanStudent(tx.QueryRow(ctx,
		`INSERT INTO students (customer_id, first_name, last_name, email, phone_no, address_id, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+studentCols,
		customerID, req.FirstName, req.LastName, req.Email, req.PhoneNo, addressID, createdBy))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) List(ctx context.Context, customerID int, search string, limit, offset int) ([]Student, int, error) {
	const countQ = `SELECT COUNT(*) FROM students
		WHERE customer_id = $1
		  AND ($2 = '' OR first_name ILIKE '%'||$2||'%' OR last_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')`
	var total int
	if err := r.db.QueryRow(ctx, countQ, customerID, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `SELECT ` + studentCols + ` FROM students
		WHERE customer_id = $1
		  AND ($2 = '' OR first_name ILIKE '%'||$2||'%' OR last_name ILIKE '%'||$2||'%' OR email ILIKE '%'||$2||'%')
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, q, customerID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Student, 0)
	for rows.Next() {
		s := Student{}
		if err := rows.Scan(&s.ID, &s.CustomerID, &s.FirstName, &s.LastName, &s.Email, &s.PhoneNo,
			&s.ProfileImageKey, &s.AddressID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, customerID, id int) (*Student, error) {
	return scanStudent(r.db.QueryRow(ctx, `SELECT `+studentCols+` FROM students WHERE customer_id = $1 AND id = $2`, customerID, id))
}

func (r *Repository) Update(ctx context.Context, customerID, id int, req UpdateStudentRequest) (*Student, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	current, err := scanStudent(tx.QueryRow(ctx, `SELECT `+studentCols+` FROM students WHERE customer_id = $1 AND id = $2 FOR UPDATE`, customerID, id))
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

	s, err := scanStudent(tx.QueryRow(ctx,
		`UPDATE students SET first_name = $3, last_name = $4, email = $5, phone_no = $6,
		 address_id = $7, updated_at = now()
		 WHERE customer_id = $1 AND id = $2 RETURNING `+studentCols,
		customerID, id, req.FirstName, req.LastName, req.Email, req.PhoneNo, addressID))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) SetProfileImage(ctx context.Context, customerID, id int, key string) (*Student, error) {
	const q = `UPDATE students SET profile_image_key = $3, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + studentCols
	return scanStudent(r.db.QueryRow(ctx, q, customerID, id, key))
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
		`DELETE FROM students WHERE customer_id = $1 AND id = $2 RETURNING profile_image_key, address_id`,
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
