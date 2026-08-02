package batches

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	model "tutorpilot/internal/modules/admin/model/batches"
	"tutorpilot/internal/pkg/events"
	"tutorpilot/internal/pkg/outbox"
	"tutorpilot/internal/pkg/pg"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const batchCols = `id, customer_id, course_id, name, status, published_at, created_by, created_at, updated_at`

func scanBatch(row pgx.Row) (*model.Batch, error) {
	b := &model.Batch{}
	err := row.Scan(&b.ID, &b.CustomerID, &b.CourseID, &b.Name, &b.Status,
		&b.PublishedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Repository) CreateBatch(ctx context.Context, customerID, createdBy, courseID int, name string) (*model.Batch, error) {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND customer_id = $2)`,
		courseID, customerID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, model.ErrNotFound
	}
	const q = `INSERT INTO batches (customer_id, course_id, name, created_by)
		VALUES ($1, $2, $3, $4) RETURNING ` + batchCols
	return scanBatch(r.db.QueryRow(ctx, q, customerID, courseID, name, createdBy))
}

func (r *Repository) ListBatches(ctx context.Context, customerID int, courseID *int, status, search string, limit, offset int) ([]model.Batch, int, error) {
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

	out := make([]model.Batch, 0)
	for rows.Next() {
		b := model.Batch{}
		if err := rows.Scan(&b.ID, &b.CustomerID, &b.CourseID, &b.Name, &b.Status,
			&b.PublishedAt, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, b)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetBatch(ctx context.Context, customerID, id int) (*model.Batch, error) {
	const q = `SELECT ` + batchCols + ` FROM batches WHERE customer_id = $1 AND id = $2`
	return scanBatch(r.db.QueryRow(ctx, q, customerID, id))
}

func (r *Repository) UpdateBatch(ctx context.Context, customerID, id int, name string) (*model.Batch, error) {
	const q = `UPDATE batches SET name = $3, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + batchCols
	return scanBatch(r.db.QueryRow(ctx, q, customerID, id, name))
}

func (r *Repository) SetStatus(ctx context.Context, customerID, id int, status string, publishedAt *time.Time) (*model.Batch, error) {
	return r.SetStatusTx(ctx, r.db, customerID, id, status, publishedAt)
}

func (r *Repository) SetStatusTx(ctx context.Context, db pg.Querier, customerID, id int, status string, publishedAt *time.Time) (*model.Batch, error) {
	const q = `UPDATE batches SET status = $3, published_at = $4, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + batchCols
	return scanBatch(db.QueryRow(ctx, q, customerID, id, status, publishedAt))
}

func (r *Repository) DeleteBatch(ctx context.Context, customerID, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM batches WHERE customer_id = $1 AND id = $2`, customerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *Repository) LoadModuleAssignments(ctx context.Context, batchID, courseID int) ([]model.ModuleAssignment, error) {
	return r.LoadModuleAssignmentsTx(ctx, r.db, batchID, courseID)
}

func (r *Repository) LoadModuleAssignmentsTx(ctx context.Context, db pg.Querier, batchID, courseID int) ([]model.ModuleAssignment, error) {
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
	rows, err := db.Query(ctx, q, batchID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.ModuleAssignment, 0)
	for rows.Next() {
		m := model.ModuleAssignment{}
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
		return model.ErrNotFound
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
		return model.ErrNotFound
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
		return model.ErrNotFound
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

func (r *Repository) LoadTutors(ctx context.Context, batchID int) ([]model.TutorSummary, error) {
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
	out := make([]model.TutorSummary, 0)
	for rows.Next() {
		t := model.TutorSummary{}
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

func (r *Repository) CourseTitle(ctx context.Context, courseID int) (string, error) {
	return r.CourseTitleTx(ctx, r.db, courseID)
}

func (r *Repository) CourseTitleTx(ctx context.Context, db pg.Querier, courseID int) (string, error) {
	var title string
	err := db.QueryRow(ctx, `SELECT title FROM courses WHERE id = $1`, courseID).Scan(&title)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", model.ErrNotFound
	}
	return title, err
}

func (r *Repository) LoadStudents(ctx context.Context, batchID, limit, offset int) ([]model.StudentSummary, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM batch_students WHERE batch_id = $1`, batchID).Scan(&total); err != nil {
		return nil, 0, err
	}

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
	out := make([]model.StudentSummary, 0)
	for rows.Next() {
		s := model.StudentSummary{}
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Phone); err != nil {
			return nil, 0, err
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

func (r *Repository) LoadAllStudents(ctx context.Context, batchID int) ([]model.StudentSummary, error) {
	return r.LoadAllStudentsTx(ctx, r.db, batchID)
}

func (r *Repository) LoadAllStudentsTx(ctx context.Context, db pg.Querier, batchID int) ([]model.StudentSummary, error) {
	const q = `
		SELECT s.dashboard_user_id, du.first_name, du.last_name, du.email, s.phone_no
		FROM batch_students bs
		JOIN students s ON s.dashboard_user_id = bs.student_id
		JOIN dashboard_users du ON du.id = s.dashboard_user_id
		WHERE bs.batch_id = $1 ORDER BY du.first_name, du.last_name`
	rows, err := db.Query(ctx, q, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.StudentSummary, 0)
	for rows.Next() {
		s := model.StudentSummary{}
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) EnrollStudentIDs(
	ctx context.Context,
	q pg.Querier,
	customerID, batchID int,
	studentIDs []int,
) (enrolled int, newlyEnrolled []int, notFound []int, err error) {
	notFound = make([]int, 0)
	newlyEnrolled = make([]int, 0)

	for _, sid := range studentIDs {
		var exists bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM students s
				JOIN dashboard_users du ON du.id = s.dashboard_user_id
				WHERE s.dashboard_user_id = $1 AND du.customer_id = $2
			)`,
			sid, customerID).Scan(&exists); err != nil {
			return 0, nil, nil, err
		}
		if !exists {
			notFound = append(notFound, sid)
			continue
		}
		tag, err := q.Exec(ctx,
			`INSERT INTO batch_students (batch_id, student_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			batchID, sid)
		if err != nil {
			return 0, nil, nil, err
		}
		if tag.RowsAffected() > 0 {
			newlyEnrolled = append(newlyEnrolled, sid)
		}
		enrolled++
	}
	return enrolled, newlyEnrolled, notFound, nil
}

func (r *Repository) LoadStudentsByIDs(ctx context.Context, q pg.Querier, ids []int) ([]model.StudentSummary, error) {
	const sql = `
		SELECT s.dashboard_user_id, du.first_name, du.last_name, du.email, s.phone_no
		FROM students s
		JOIN dashboard_users du ON du.id = s.dashboard_user_id
		WHERE s.dashboard_user_id = ANY($1)
		ORDER BY du.first_name, du.last_name`
	rows, err := q.Query(ctx, sql, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.StudentSummary, 0, len(ids))
	for rows.Next() {
		var s model.StudentSummary
		if err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) RemoveStudent(ctx context.Context, batchID, studentID int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM batch_students WHERE batch_id = $1 AND student_id = $2`, batchID, studentID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

type StudentRow struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

type ImportRow struct {
	StudentRow
	Row          int
	PasswordHash string
	PasswordSalt string

	InviteEvent events.Event

	EnrolmentEvent events.Event
}

type ImportOutcome struct {
	Email   string
	Created bool
}

type ImportParams struct {
	CustomerID int
	BatchID    int
	Stream     string

	NotifyEnrolment bool
}

func (r *Repository) ImportStudents(
	ctx context.Context,
	p ImportParams,
	rows []ImportRow,
) ([]ImportOutcome, []model.SkippedRow, error) {
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
		`SELECT id FROM roles WHERE customer_id = $1 AND name = 'Student'`, p.CustomerID,
	).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, model.ErrNoStudentRole
	}
	if err != nil {
		return nil, nil, err
	}

	outcomes := make([]ImportOutcome, 0, len(rows))
	var skipped []model.SkippedRow
	var queued []events.Event

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
				p.CustomerID, roleID, row.Email, row.PasswordHash, row.PasswordSalt, row.FirstName, row.LastName,
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
				p.BatchID, newID); err != nil {
				return nil, nil, err
			}
			outcomes = append(outcomes, ImportOutcome{Email: row.Email, Created: true})
			queued = append(queued, row.InviteEvent)
			if p.NotifyEnrolment {
				queued = append(queued, row.EnrolmentEvent)
			}

		case err != nil:
			return nil, nil, err

		default:
			if existingCustomerID != p.CustomerID {
				skipped = append(skipped, model.SkippedRow{Row: row.Row, Reason: "email already belongs to a different organization"})
				continue
			}
			var isStudent bool
			if err := tx.QueryRow(ctx,
				`SELECT EXISTS(SELECT 1 FROM students WHERE dashboard_user_id = $1)`, existingID,
			).Scan(&isStudent); err != nil {
				return nil, nil, err
			}
			if !isStudent {
				skipped = append(skipped, model.SkippedRow{Row: row.Row, Reason: "email belongs to an existing account that is not a student"})
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
				p.BatchID, existingID); err != nil {
				return nil, nil, err
			}
			outcomes = append(outcomes, ImportOutcome{Email: row.Email, Created: false})
			if p.NotifyEnrolment {
				queued = append(queued, row.EnrolmentEvent)
			}
		}
	}

	if p.Stream != "" && len(queued) > 0 {
		if err := outbox.InsertBatch(ctx, tx, p.Stream, queued); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return outcomes, skipped, nil
}

const driveCols = `id, batch_id, parent_id, name, node_type, object_key, content_type, size_bytes, is_system, created_at, updated_at`

func scanNode(row pgx.Row) (*model.DriveNode, error) {
	n := &model.DriveNode{}
	err := row.Scan(&n.ID, &n.BatchID, &n.ParentID, &n.Name, &n.NodeType,
		&n.ObjectKey, &n.ContentType, &n.SizeBytes, &n.IsSystem, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return n, nil
}

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
		  AND node_type = '`+model.NodeFolder+`'`,
		batchID, parentID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	err = q.QueryRow(ctx, `
		INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, is_system)
		VALUES ($1, $2, $3, '`+model.NodeFolder+`', $4)
		RETURNING id`,
		batchID, parentID, name, isSystem).Scan(&id)
	return id, err
}

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
		VALUES ($1, $2, $3, '`+model.NodeFile+`', $4, $5, $6)
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
	return nodeType == model.NodeFolder, nil
}

func (r *Repository) CreateFolder(ctx context.Context, batchID int, parentID *int, name string, createdBy int) (*model.DriveNode, error) {
	ok, err := r.parentIsFolder(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}
	taken, err := r.nameTaken(ctx, batchID, parentID, name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, model.ErrNameTaken
	}
	const q = `INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, created_by)
		VALUES ($1, $2, $3, '` + model.NodeFolder + `', $4) RETURNING ` + driveCols
	return scanNode(r.db.QueryRow(ctx, q, batchID, parentID, name, createdBy))
}

func (r *Repository) CreateFile(ctx context.Context, batchID int, parentID *int, name, objectKey, contentType string, size int64, createdBy int) (*model.DriveNode, error) {
	ok, err := r.parentIsFolder(ctx, batchID, parentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}
	taken, err := r.nameTaken(ctx, batchID, parentID, name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, model.ErrNameTaken
	}
	const q = `INSERT INTO batch_drive_nodes (batch_id, parent_id, name, node_type, object_key, content_type, size_bytes, created_by)
		VALUES ($1, $2, $3, '` + model.NodeFile + `', $4, $5, $6, $7) RETURNING ` + driveCols
	return scanNode(r.db.QueryRow(ctx, q, batchID, parentID, name, objectKey, contentType, size, createdBy))
}

func (r *Repository) ListChildren(ctx context.Context, batchID int, parentID *int) ([]model.DriveNode, error) {
	const q = `SELECT ` + driveCols + ` FROM batch_drive_nodes
		WHERE batch_id = $1 AND ((parent_id IS NULL AND $2::int IS NULL) OR parent_id = $2)
		ORDER BY node_type DESC, name`
	rows, err := r.db.Query(ctx, q, batchID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]model.DriveNode, 0)
	for rows.Next() {
		n := model.DriveNode{}
		if err := rows.Scan(&n.ID, &n.BatchID, &n.ParentID, &n.Name, &n.NodeType,
			&n.ObjectKey, &n.ContentType, &n.SizeBytes, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) GetNode(ctx context.Context, batchID, nodeID int) (*model.DriveNode, error) {
	const q = `SELECT ` + driveCols + ` FROM batch_drive_nodes WHERE batch_id = $1 AND id = $2`
	return scanNode(r.db.QueryRow(ctx, q, batchID, nodeID))
}

func (r *Repository) RenameNode(ctx context.Context, batchID, nodeID int, name string) (*model.DriveNode, error) {
	current, err := r.GetNode(ctx, batchID, nodeID)
	if err != nil {
		return nil, err
	}
	if current.IsSystem {
		return nil, model.ErrSystemNode
	}
	taken, err := r.nameTaken(ctx, batchID, current.ParentID, name, nodeID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, model.ErrNameTaken
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
		return nil, model.ErrSystemNode
	}

	const subtreeQ = `
		WITH RECURSIVE subtree AS (
			SELECT id, node_type, object_key FROM batch_drive_nodes WHERE id = $1 AND batch_id = $2
			UNION ALL
			SELECT n.id, n.node_type, n.object_key
			FROM batch_drive_nodes n JOIN subtree s ON n.parent_id = s.id
		)
		SELECT object_key FROM subtree WHERE node_type = '` + model.NodeFile + `' AND object_key IS NOT NULL`
	rows, err := r.db.Query(ctx, subtreeQ, nodeID, batchID)
	if err != nil {
		return nil, err
	}

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
		return nil, model.ErrNotFound
	}
	return keys, nil
}
