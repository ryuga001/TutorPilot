package batches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/pg"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrNameTaken = errors.New("a file or folder with this name already exists here")

	// ErrSystemNode guards folders the application created and manages, such as
	// the one lecture recordings are filed into.
	ErrSystemNode = errors.New("this folder is managed automatically and cannot be changed")

	// ErrNoStudentRole means the tenant has no 'Student' role to assign an
	// imported student's login to (see migration 000012).
	ErrNoStudentRole = errors.New("this organization has no Student role configured")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const batchCols = `id, customer_id, course_id, name, status, published_at, created_by, created_at, updated_at`

func scanBatch(row pgx.Row) (*Batch, error) {
	b := &Batch{}
	err := row.Scan(&b.ID, &b.CustomerID, &b.CourseID, &b.Name, &b.Status,
		&b.PublishedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Repository) CreateBatch(ctx context.Context, customerID, createdBy, courseID int, name string) (*Batch, error) {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND customer_id = $2)`,
		courseID, customerID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	const q = `INSERT INTO batches (customer_id, course_id, name, created_by)
		VALUES ($1, $2, $3, $4) RETURNING ` + batchCols
	return scanBatch(r.db.QueryRow(ctx, q, customerID, courseID, name, createdBy))
}

func (r *Repository) ListBatches(ctx context.Context, customerID int, courseID *int, status, search string, limit, offset int) ([]Batch, int, error) {
	const countQ = `SELECT COUNT(*) FROM batches
		WHERE customer_id = $1
		  AND ($2::int IS NULL OR course_id = $2)
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR name ILIKE '%' || $4 || '%')`
	var total int
	if err := r.db.QueryRow(ctx, countQ, customerID, courseID, status, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `SELECT ` + batchCols + ` FROM batches
		WHERE customer_id = $1
		  AND ($2::int IS NULL OR course_id = $2)
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR name ILIKE '%' || $4 || '%')
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6`
	rows, err := r.db.Query(ctx, q, customerID, courseID, status, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Batch, 0)
	for rows.Next() {
		b := Batch{}
		if err := rows.Scan(&b.ID, &b.CustomerID, &b.CourseID, &b.Name, &b.Status,
			&b.PublishedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetBatch(ctx context.Context, customerID, id int) (*Batch, error) {
	const q = `SELECT ` + batchCols + ` FROM batches WHERE customer_id = $1 AND id = $2`
	return scanBatch(r.db.QueryRow(ctx, q, customerID, id))
}

func (r *Repository) UpdateBatch(ctx context.Context, customerID, id int, name string) (*Batch, error) {
	const q = `UPDATE batches SET name = $3, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + batchCols
	return scanBatch(r.db.QueryRow(ctx, q, customerID, id, name))
}

func (r *Repository) SetStatus(ctx context.Context, customerID, id int, status string, publishedAt *time.Time) (*Batch, error) {
	const q = `UPDATE batches SET status = $3, published_at = $4, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + batchCols
	return scanBatch(r.db.QueryRow(ctx, q, customerID, id, status, publishedAt))
}

func (r *Repository) DeleteBatch(ctx context.Context, customerID, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM batches WHERE customer_id = $1 AND id = $2`, customerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) LoadModuleAssignments(ctx context.Context, batchID, courseID int) ([]ModuleAssignment, error) {
	// A tutor's name lives on dashboard_users now (see migration 000012);
	// tutors.id was replaced by tutors.dashboard_user_id, which is what
	// batch_module_tutors.tutor_id references.
	const q = `
		SELECT cm.id, cm.title, cm.position,
		       bmt.tutor_id, tdu.first_name, tdu.last_name, tdu.email,
		       bmt.start_date, bmt.expected_end_date
		FROM course_modules cm
		LEFT JOIN batch_module_tutors bmt ON bmt.course_module_id = cm.id AND bmt.batch_id = $1
		LEFT JOIN tutors t ON t.dashboard_user_id = bmt.tutor_id
		LEFT JOIN dashboard_users tdu ON tdu.id = t.dashboard_user_id
		WHERE cm.course_id = $2
		ORDER BY cm.position, cm.id`
	rows, err := r.db.Query(ctx, q, batchID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ModuleAssignment, 0)
	for rows.Next() {
		m := ModuleAssignment{}
		if err := rows.Scan(&m.CourseModuleID, &m.ModuleTitle, &m.ModulePosition,
			&m.TutorID, &m.TutorFirstName, &m.TutorLastName, &m.TutorEmail,
			&m.StartDate, &m.ExpectedEndDate); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) AssignModuleTutor(ctx context.Context, customerID, batchID, courseID, courseModuleID, tutorID int, start, end time.Time) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM course_modules WHERE id = $1 AND course_id = $2)`,
		courseModuleID, courseID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM tutors t
			JOIN dashboard_users du ON du.id = t.dashboard_user_id
			WHERE t.dashboard_user_id = $1 AND du.customer_id = $2
		)`,
		tutorID, customerID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var oldTutorID *int
	err = tx.QueryRow(ctx,
		`SELECT tutor_id FROM batch_module_tutors WHERE batch_id = $1 AND course_module_id = $2`,
		batchID, courseModuleID).Scan(&oldTutorID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO batch_module_tutors (batch_id, course_module_id, tutor_id, start_date, expected_end_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (batch_id, course_module_id) DO UPDATE SET
			tutor_id = EXCLUDED.tutor_id,
			start_date = EXCLUDED.start_date,
			expected_end_date = EXCLUDED.expected_end_date,
			updated_at = now()`,
		batchID, courseModuleID, tutorID, start, end)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO batch_tutors (batch_id, tutor_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		batchID, tutorID); err != nil {
		return err
	}

	if oldTutorID != nil && *oldTutorID != tutorID {
		if err := pruneBatchTutor(ctx, tx, batchID, *oldTutorID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) UnassignModuleTutor(ctx context.Context, batchID, courseModuleID int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var tutorID int
	err = tx.QueryRow(ctx,
		`DELETE FROM batch_module_tutors WHERE batch_id = $1 AND course_module_id = $2 RETURNING tutor_id`,
		batchID, courseModuleID).Scan(&tutorID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if err := pruneBatchTutor(ctx, tx, batchID, tutorID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func pruneBatchTutor(ctx context.Context, tx pgx.Tx, batchID, tutorID int) error {
	var stillAssigned bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM batch_module_tutors WHERE batch_id = $1 AND tutor_id = $2)`,
		batchID, tutorID).Scan(&stillAssigned); err != nil {
		return err
	}
	if stillAssigned {
		return nil
	}
	_, err := tx.Exec(ctx, `DELETE FROM batch_tutors WHERE batch_id = $1 AND tutor_id = $2`, batchID, tutorID)
	return err
}

func (r *Repository) LoadTutors(ctx context.Context, batchID int) ([]TutorSummary, error) {
	// A tutor's name lives on dashboard_users now (see migration 000012).
	const q = `
		SELECT t.dashboard_user_id, du.first_name, du.last_name, du.email
		FROM batch_tutors bt
		JOIN tutors t ON t.dashboard_user_id = bt.tutor_id
		JOIN dashboard_users du ON du.id = t.dashboard_user_id
		WHERE bt.batch_id = $1 ORDER BY du.first_name, du.last_name`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TutorSummary, 0)
	for rows.Next() {
		t := TutorSummary{}
		if err := rows.Scan(&t.ID, &t.FirstName, &t.LastName, &t.Email); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) CountTutors(ctx context.Context, batchID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM batch_tutors WHERE batch_id = $1`, batchID).Scan(&n)
	return n, err
}

func (r *Repository) CountStudents(ctx context.Context, batchID int) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM batch_students WHERE batch_id = $1`, batchID).Scan(&n)
	return n, err
}

// CourseTitle returns a course's title (used to populate publish-notification emails).
func (r *Repository) CourseTitle(ctx context.Context, courseID int) (string, error) {
	var title string
	err := r.db.QueryRow(ctx, `SELECT title FROM courses WHERE id = $1`, courseID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return title, err
}

func (r *Repository) LoadStudents(ctx context.Context, batchID, limit, offset int) ([]StudentSummary, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM batch_students WHERE batch_id = $1`, batchID).Scan(&total); err != nil {
		return nil, 0, err
	}
	// A student's name lives on dashboard_users now (see migration 000012).
	const q = `
		SELECT s.dashboard_user_id, du.first_name, du.last_name, du.email, s.phone_no
		FROM batch_students bs
		JOIN students s ON s.dashboard_user_id = bs.student_id
		JOIN dashboard_users du ON du.id = s.dashboard_user_id
		WHERE bs.batch_id = $1 ORDER BY bs.enrolled_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, batchID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]StudentSummary, 0)
	for rows.Next() {
		s := StudentSummary{}
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Phone); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) LoadAllStudents(ctx context.Context, batchID int) ([]StudentSummary, error) {
	const q = `
		SELECT s.dashboard_user_id, du.first_name, du.last_name, du.email, s.phone_no
		FROM batch_students bs
		JOIN students s ON s.dashboard_user_id = bs.student_id
		JOIN dashboard_users du ON du.id = s.dashboard_user_id
		WHERE bs.batch_id = $1 ORDER BY du.first_name, du.last_name`
	rows, err := r.db.Query(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]StudentSummary, 0)
	for rows.Next() {
		s := StudentSummary{}
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) EnrollStudentIDs(ctx context.Context, customerID, batchID int, studentIDs []int) (int, []int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback(ctx)

	enrolled := 0
	notFound := make([]int, 0)
	for _, sid := range studentIDs {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM students s
				JOIN dashboard_users du ON du.id = s.dashboard_user_id
				WHERE s.dashboard_user_id = $1 AND du.customer_id = $2
			)`,
			sid, customerID).Scan(&exists); err != nil {
			return 0, nil, err
		}
		if !exists {
			notFound = append(notFound, sid)
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO batch_students (batch_id, student_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			batchID, sid); err != nil {
			return 0, nil, err
		}
		enrolled++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, err
	}
	return enrolled, notFound, nil
}

func (r *Repository) RemoveStudent(ctx context.Context, batchID, studentID int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM batch_students WHERE batch_id = $1 AND student_id = $2`, batchID, studentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type StudentRow struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// ImportRow is one CSV row ready to persist: the service has already generated
// and hashed a temporary password for it, exactly as a single Create does.
type ImportRow struct {
	StudentRow
	Row          int // original CSV row number, echoed back on skip
	PasswordHash string
	PasswordSalt string
}

// ImportOutcome reports what happened to one row that was not skipped.
// Created is false when the row matched an existing student in this
// organization and was just re-enrolled/updated — no new login, no email to send.
type ImportOutcome struct {
	Email   string
	Created bool
}

// ImportStudents creates a login for each new student (matching the single
// Create flow) and enrolls everyone in the batch. A row whose email already
// belongs to a different organization, or to an account in this organization
// that isn't a student, is skipped rather than silently repurposed.
func (r *Repository) ImportStudents(
	ctx context.Context,
	customerID, batchID int,
	rows []ImportRow,
) ([]ImportOutcome, []SkippedRow, error) {
	if len(rows) == 0 {
		return nil, nil, nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var roleID int
	err = tx.QueryRow(ctx,
		`SELECT id FROM roles WHERE customer_id = $1 AND name = 'Student'`, customerID,
	).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNoStudentRole
	}
	if err != nil {
		return nil, nil, err
	}

	outcomes := make([]ImportOutcome, 0, len(rows))
	var skipped []SkippedRow

	for _, row := range rows {
		var existingID, existingCustomerID int
		err := tx.QueryRow(ctx,
			`SELECT id, customer_id FROM dashboard_users WHERE email = $1`, row.Email,
		).Scan(&existingID, &existingCustomerID)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			var newID int
			if err := tx.QueryRow(ctx,
				`INSERT INTO dashboard_users
					(customer_id, role_id, email, password_hash, password_salt, first_name, last_name)
				 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
				customerID, roleID, row.Email, row.PasswordHash, row.PasswordSalt, row.FirstName, row.LastName,
			).Scan(&newID); err != nil {
				return nil, nil, err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO students (dashboard_user_id, phone_no) VALUES ($1, $2)`,
				newID, row.Phone); err != nil {
				return nil, nil, err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO batch_students (batch_id, student_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				batchID, newID); err != nil {
				return nil, nil, err
			}
			outcomes = append(outcomes, ImportOutcome{Email: row.Email, Created: true})

		case err != nil:
			return nil, nil, err

		default:
			if existingCustomerID != customerID {
				skipped = append(skipped, SkippedRow{Row: row.Row, Reason: "email already belongs to a different organization"})
				continue
			}
			var isStudent bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM students WHERE dashboard_user_id = $1)`, existingID,
			).Scan(&isStudent); err != nil {
				return nil, nil, err
			}
			if !isStudent {
				skipped = append(skipped, SkippedRow{Row: row.Row, Reason: "email belongs to an existing account that is not a student"})
				continue
			}
			if _, err := tx.Exec(ctx,
				`UPDATE dashboard_users SET first_name = $2, last_name = $3 WHERE id = $1`,
				existingID, row.FirstName, row.LastName); err != nil {
				return nil, nil, err
			}
			if _, err := tx.Exec(ctx,
				`UPDATE students SET phone_no = $2, updated_at = now() WHERE dashboard_user_id = $1`,
				existingID, row.Phone); err != nil {
				return nil, nil, err
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO batch_students (batch_id, student_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				batchID, existingID); err != nil {
				return nil, nil, err
			}
			outcomes = append(outcomes, ImportOutcome{Email: row.Email, Created: false})
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return outcomes, skipped, nil
}

const driveCols = `id, batch_id, parent_id, name, node_type, object_key, content_type, size_bytes, is_system, created_at, updated_at`

func scanNode(row pgx.Row) (*DriveNode, error) {
	n := &DriveNode{}
	err := row.Scan(&n.ID, &n.BatchID, &n.ParentID, &n.Name, &n.NodeType,
		&n.ObjectKey, &n.ContentType, &n.SizeBytes, &n.IsSystem, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

// --- Drive writes for the lecture recording pipeline -------------------------
//
// Recordings land in the batch drive, so the lecture module needs to create nodes.
// These take a pg.Querier rather than using the pool directly, so the caller can
// run them inside its own transaction alongside the lectures update.

// EnsureFolder returns the id of a folder, creating it if absent. Concurrent
// deliveries of the same webhook both end up with the same folder rather than two.
func (r *Repository) EnsureFolder(
	ctx context.Context,
	q pg.Querier,
	batchID int,
	parentID *int,
	name string,
	isSystem bool,
) (int, error) {
	if q == nil {
		q = r.db
	}

	var id int
	err := q.QueryRow(ctx, `
		SELECT id FROM batch_drive_nodes
		WHERE batch_id = $1
		  AND ((parent_id IS NULL AND $2::int IS NULL) OR parent_id = $2)
		  AND lower(name) = lower($3)
		  AND node_type = '`+NodeFolder+`'`,
		batchID, parentID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	err = q.QueryRow(ctx, `
		INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, is_system)
		VALUES ($1, $2, $3, '`+NodeFolder+`', $4)
		RETURNING id`,
		batchID, parentID, name, isSystem).Scan(&id)
	return id, err
}

// InsertFile adds a file node for an object already written to storage.
func (r *Repository) InsertFile(
	ctx context.Context,
	q pg.Querier,
	batchID int,
	parentID *int,
	name, objectKey, contentType string,
	sizeBytes int64,
) (int, error) {
	if q == nil {
		q = r.db
	}
	var id int
	err := q.QueryRow(ctx, `
		INSERT INTO batch_drive_nodes
			(batch_id, parent_id, name, node_type, object_key, content_type, size_bytes)
		VALUES ($1, $2, $3, '`+NodeFile+`', $4, $5, $6)
		RETURNING id`,
		batchID, parentID, name, objectKey, contentType, sizeBytes).Scan(&id)
	return id, err
}

func (r *Repository) nameTaken(ctx context.Context, batchID int, parentID *int, name string, excludeID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM batch_drive_nodes
			WHERE batch_id = $1
			  AND ((parent_id IS NULL AND $2::int IS NULL) OR parent_id = $2)
			  AND lower(name) = lower($3)
			  AND id <> $4
		)`, batchID, parentID, name, excludeID).Scan(&exists)
	return exists, err
}

func (r *Repository) parentIsFolder(ctx context.Context, batchID int, parentID *int) (bool, error) {
	if parentID == nil {
		return true, nil
	}
	var nodeType string
	err := r.db.QueryRow(ctx,
		`SELECT node_type FROM batch_drive_nodes WHERE id = $1 AND batch_id = $2`, *parentID, batchID).Scan(&nodeType)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return nodeType == NodeFolder, nil
}

func (r *Repository) CreateFolder(ctx context.Context, batchID int, parentID *int, name string, createdBy int) (*DriveNode, error) {
	ok, err := r.parentIsFolder(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	taken, err := r.nameTaken(ctx, batchID, parentID, name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	const q = `INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, created_by)
		VALUES ($1, $2, $3, '` + NodeFolder + `', $4) RETURNING ` + driveCols
	return scanNode(r.db.QueryRow(ctx, q, batchID, parentID, name, createdBy))
}

func (r *Repository) CreateFile(ctx context.Context, batchID int, parentID *int, name, objectKey, contentType string, size int64, createdBy int) (*DriveNode, error) {
	ok, err := r.parentIsFolder(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	taken, err := r.nameTaken(ctx, batchID, parentID, name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	const q = `INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, object_key, content_type, size_bytes, created_by)
		VALUES ($1, $2, $3, '` + NodeFile + `', $4, $5, $6, $7) RETURNING ` + driveCols
	return scanNode(r.db.QueryRow(ctx, q, batchID, parentID, name, objectKey, contentType, size, createdBy))
}

func (r *Repository) ListChildren(ctx context.Context, batchID int, parentID *int) ([]DriveNode, error) {
	const q = `SELECT ` + driveCols + ` FROM batch_drive_nodes
		WHERE batch_id = $1 AND ((parent_id IS NULL AND $2::int IS NULL) OR parent_id = $2)
		ORDER BY node_type DESC, name`
	rows, err := r.db.Query(ctx, q, batchID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]DriveNode, 0)
	for rows.Next() {
		n := DriveNode{}
		if err := rows.Scan(&n.ID, &n.BatchID, &n.ParentID, &n.Name, &n.NodeType,
			&n.ObjectKey, &n.ContentType, &n.SizeBytes, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) GetNode(ctx context.Context, batchID, nodeID int) (*DriveNode, error) {
	const q = `SELECT ` + driveCols + ` FROM batch_drive_nodes WHERE batch_id = $1 AND id = $2`
	return scanNode(r.db.QueryRow(ctx, q, batchID, nodeID))
}

func (r *Repository) RenameNode(ctx context.Context, batchID, nodeID int, name string) (*DriveNode, error) {
	current, err := r.GetNode(ctx, batchID, nodeID)
	if err != nil {
		return nil, err
	}
	if current.IsSystem {
		return nil, ErrSystemNode
	}
	taken, err := r.nameTaken(ctx, batchID, current.ParentID, name, nodeID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrNameTaken
	}
	const q = `UPDATE batch_drive_nodes SET name = $3, updated_at = now()
		WHERE batch_id = $1 AND id = $2 RETURNING ` + driveCols
	return scanNode(r.db.QueryRow(ctx, q, batchID, nodeID, name))
}

func (r *Repository) DeleteNode(ctx context.Context, batchID, nodeID int) ([]string, error) {
	current, err := r.GetNode(ctx, batchID, nodeID)
	if err != nil {
		return nil, err
	}
	if current.IsSystem {
		return nil, ErrSystemNode
	}

	const subtreeQ = `
		WITH RECURSIVE subtree AS (
			SELECT id, node_type, object_key FROM batch_drive_nodes WHERE id = $1 AND batch_id = $2
			UNION ALL
			SELECT n.id, n.node_type, n.object_key
			FROM batch_drive_nodes n JOIN subtree s ON n.parent_id = s.id
		)
		SELECT object_key FROM subtree WHERE node_type = '` + NodeFile + `' AND object_key IS NOT NULL`
	rows, err := r.db.Query(ctx, subtreeQ, nodeID, batchID)
	if err != nil {
		return nil, err
	}
	//nolint:staticcheck // rows is closed explicitly below, before the DELETE runs.
	keys := make([]string, 0)
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tag, err := r.db.Exec(ctx, `DELETE FROM batch_drive_nodes WHERE id = $1 AND batch_id = $2`, nodeID, batchID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return keys, nil
}
