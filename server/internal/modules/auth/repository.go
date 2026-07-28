package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/jwtutil"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// roleSpec describes a role created for every new tenant and the privileges it
// receives. It mirrors the grants seeded for the TutorPilot tenant, and must stay
// in step with migration 000012_tutor_student_dashboard_user_link, which
// backfills the same roles onto tenants that already existed.
type roleSpec struct {
	name   string
	typ    string
	all    bool     // grant every privilege...
	except []string // ...except these
	only   []string // or grant only these
}

// The three portal roles are strictly nested: student ⊂ tutor ⊂ admin. Expressing
// that by composition rather than three hand-written lists is what keeps the
// invariant true — adding a student privilege cannot accidentally leave tutors
// without it. Migration 000012 composes the same sets in SQL.

// studentPrivileges is the base set: look at your own courses and batches, join
// the lectures you are enrolled in, watch the recordings, edit your own profile.
var studentPrivileges = []string{
	"portal.access",
	"self.profile.view", "self.profile.edit",
	"course.view", "batch.view",
	"lecture.view", "lecture.join", "recording.view",
}

// tutorOnlyPrivileges is what a tutor adds on top of a student: they run the
// lectures rather than only attending them, and can see the roster and upload
// material for the batches they teach.
var tutorOnlyPrivileges = []string{
	"view_dashboard", "student.view", "drive.upload",
	"lecture.create", "lecture.edit", "lecture.control", "lecture.publish",
}

// tutorPrivileges is the student set plus the tutor additions. Building it by
// concatenation rather than writing a third list is what guarantees the nesting.
var tutorPrivileges = append(append([]string{}, studentPrivileges...), tutorOnlyPrivileges...)

var newTenantRoles = []roleSpec{
	{name: RoleNameAdmin, typ: "admin", all: true, except: []string{"manage_privileges"}},
	{name: "User", typ: "user", only: []string{"view_dashboard"}},
	{name: RoleNameTutor, typ: jwtutil.SubjectTutor, only: tutorPrivileges},
	{name: RoleNameStudent, typ: jwtutil.SubjectStudent, only: studentPrivileges},
}

// Role names are stable per tenant, so provisioning can look a member's role up
// by name rather than threading ids around.
const (
	RoleNameAdmin   = "Admin"
	RoleNameTutor   = "Tutor"
	RoleNameStudent = "Student"
)

const adminRoleType = "admin"

const userSelect = `
	SELECT du.id, du.customer_id, du.role_id, COALESCE(r.type, ''),
	       du.email, du.password_hash, du.password_salt,
	       du.first_name, du.last_name, du.created_at
	FROM dashboard_users du
	LEFT JOIN roles r ON r.id = du.role_id`

func scanUser(row pgx.Row) (*DashboardUser, error) {
	u := &DashboardUser{}
	err := row.Scan(&u.ID, &u.CustomerID, &u.RoleID, &u.RoleType,
		&u.Email, &u.PasswordHash, &u.PasswordSalt,
		&u.FirstName, &u.LastName, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByEmail is the only login lookup. Email is unique across the whole
// installation (dashboard_users.email UNIQUE), so no tenant id is needed to find
// the account — the account itself says which tenant it belongs to.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*DashboardUser, error) {
	return scanUser(r.db.QueryRow(ctx, userSelect+` WHERE du.email = $1`, NormalizeEmail(email)))
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*DashboardUser, error) {
	return scanUser(r.db.QueryRow(ctx, userSelect+` WHERE du.id = $1`, id))
}

// EmailTaken reports whether an address is already registered, for a friendly
// pre-flight message before signup goes further. The UNIQUE constraint on
// dashboard_users.email is what actually enforces it.
func (r *Repository) EmailTaken(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM dashboard_users WHERE email = $1)`,
		NormalizeEmail(email)).Scan(&exists)
	return exists, err
}

func (r *Repository) SetPassword(ctx context.Context, userID int, passwordHash string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE dashboard_users SET password_hash = $2 WHERE id = $1`, userID, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetCustomerJWTSecret(ctx context.Context, customerID int) (string, error) {
	var secret string
	err := r.db.QueryRow(ctx, `SELECT jwt_secret FROM customers WHERE id = $1`, customerID).Scan(&secret)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return secret, err
}

// GetUserPrivileges returns the privilege names a user holds.
func (r *Repository) GetUserPrivileges(ctx context.Context, userID int) ([]string, error) {
	const q = `
		SELECT p.name
		FROM dashboard_users du
		JOIN role_privileges rp ON rp.role_id = du.role_id
		JOIN privileges p ON p.id = rp.privilege_id
		WHERE du.id = $1`
	rows, err := r.db.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		privs = append(privs, name)
	}
	return privs, rows.Err()
}

// CreateTenantWithAdmin creates a customer, its default roles and grants, and
// the first dashboard user (admin) — all in one transaction.
func (r *Repository) CreateTenantWithAdmin(
	ctx context.Context,
	orgName, firstName, lastName, email, passwordHash, salt string,
) (*DashboardUser, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var customerID int
	err = tx.QueryRow(ctx,
		`INSERT INTO customers (org_name, first_name, last_name) VALUES ($1, $2, $3) RETURNING id`,
		orgName, firstName, lastName,
	).Scan(&customerID)
	if err != nil {
		return nil, mapConstraintErr(err)
	}

	var adminRoleID int
	for _, spec := range newTenantRoles {
		var roleID int
		err = tx.QueryRow(ctx,
			`INSERT INTO roles (name, type, customer_id) VALUES ($1, $2, $3) RETURNING id`,
			spec.name, spec.typ, customerID,
		).Scan(&roleID)
		if err != nil {
			return nil, err
		}
		if err := grantPrivileges(ctx, tx, roleID, spec); err != nil {
			return nil, err
		}
		if spec.typ == adminRoleType {
			adminRoleID = roleID
		}
	}

	user := &DashboardUser{
		CustomerID: customerID,
		RoleID:     &adminRoleID,
		RoleType:   adminRoleType,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO dashboard_users (customer_id, role_id, email, password_hash, password_salt, first_name, last_name)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		customerID, adminRoleID, email, passwordHash, salt, firstName, lastName,
	).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return nil, mapConstraintErr(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return user, nil
}

// NormalizeEmail is applied on every write and lookup. dashboard_users.email is a
// plain UNIQUE constraint, which is case-sensitive, so the case-insensitive
// guarantee has to come from here.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func grantPrivileges(ctx context.Context, tx pgx.Tx, roleID int, spec roleSpec) error {
	switch {
	case spec.all && len(spec.except) == 0:
		_, err := tx.Exec(ctx,
			`INSERT INTO role_privileges (role_id, privilege_id)
			 SELECT $1, id FROM privileges`,
			roleID)
		return err
	case spec.all:
		_, err := tx.Exec(ctx,
			`INSERT INTO role_privileges (role_id, privilege_id)
			 SELECT $1, id FROM privileges WHERE NOT (name = ANY($2))`,
			roleID, spec.except)
		return err
	case len(spec.only) > 0:
		_, err := tx.Exec(ctx,
			`INSERT INTO role_privileges (role_id, privilege_id)
			 SELECT $1, id FROM privileges WHERE name = ANY($2)`,
			roleID, spec.only)
		return err
	default:
		return nil
	}
}

func mapConstraintErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "customers_org_name_key":
			return ErrOrgTaken
		case "dashboard_users_email_key":
			return ErrEmailTaken
		}
	}
	return err
}
