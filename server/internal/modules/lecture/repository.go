package lecture

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/pg"
	"tutorpilot/internal/pkg/scope"
)

var (
	ErrNotFound = errors.New("not found")

	// ErrInvalidTransition is returned when a lecture is not in the state the
	// requested change requires — starting one that already ended, for instance.
	ErrInvalidTransition = errors.New("lecture is not in a state that allows this")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Pool exposes the pool so the service can open a transaction spanning this
// repository and the batch drive.
func (r *Repository) Pool() *pgxpool.Pool { return r.db }

const lectureCols = `
	l.id, l.customer_id, l.batch_id, l.module_id, l.tutor_id,
	l.title, l.description, l.room_name, l.egress_id, l.status,
	l.recording_enabled, l.recording_status, l.recording_url, l.recording_object_key,
	l.recording_node_id, l.recording_duration_seconds, l.recording_size_bytes,
	l.start_time, l.actual_start_at, l.end_time,
	l.created_by, l.created_at, l.updated_at`

// lectureContext joins the names a listing needs. Every join rides a primary key.
// A tutor's name lives on dashboard_users now (see migration 000012); tutors.id
// was replaced by tutors.dashboard_user_id, which is what l.tutor_id references.
const lectureContextCols = `
	b.name, c.title, cm.title, NULLIF(TRIM(COALESCE(tdu.first_name, '') || ' ' || COALESCE(tdu.last_name, '')), '')`

const lectureFrom = `
	FROM lectures l
	JOIN batches b ON b.id = l.batch_id
	JOIN courses c ON c.id = b.course_id
	LEFT JOIN course_modules cm ON cm.id = l.module_id
	LEFT JOIN tutors t ON t.dashboard_user_id = l.tutor_id
	LEFT JOIN dashboard_users tdu ON tdu.id = t.dashboard_user_id`

func scanLecture(row pgx.Row) (*Lecture, error) {
	l := &Lecture{}
	err := row.Scan(
		&l.ID, &l.CustomerID, &l.BatchID, &l.ModuleID, &l.TutorID,
		&l.Title, &l.Description, &l.RoomName, &l.EgressID, &l.Status,
		&l.RecordingEnabled, &l.RecordingStatus, &l.RecordingURL, &l.RecordingObjectKey,
		&l.RecordingNodeID, &l.RecordingDuration, &l.RecordingSize,
		&l.StartTime, &l.ActualStartAt, &l.EndTime,
		&l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
		&l.BatchName, &l.CourseTitle, &l.ModuleTitle, &l.TutorName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Create validates that the batch, tutor and module all belong to the tenant, then
// inserts. It no longer touches LiveKit: the room is created when the lecture
// starts, because a room made now would be reaped by its empty-timeout long before
// a lecture scheduled for tomorrow begins.
func (r *Repository) Create(
	ctx context.Context,
	sc scope.Scope,
	createdBy int,
	roomName string,
	req CreateLectureRequest,
) (*Lecture, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// The batch must belong to the tenant and, for a tutor, to them.
	pred, args := sc.BatchPredicate("b.id", 3)
	batchQ := `SELECT EXISTS(SELECT 1 FROM batches b WHERE b.id = $1 AND b.customer_id = $2` + pred + `)`
	var ok bool
	if err := tx.QueryRow(ctx, batchQ, append([]any{req.BatchID, sc.CustomerID}, args...)...).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	if req.TutorID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $1 AND EXISTS(SELECT 1 FROM dashboard_users du WHERE du.id = t.dashboard_user_id AND du.customer_id = $2))`,
			*req.TutorID, sc.CustomerID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}

	if req.ModuleID != nil {
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM batches b
				JOIN course_modules cm ON cm.course_id = b.course_id
				WHERE b.id = $1 AND b.customer_id = $2 AND cm.id = $3
			)`, req.BatchID, sc.CustomerID, *req.ModuleID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}

	recordingStatus := RecordingNone
	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO lectures (
			customer_id, batch_id, module_id, tutor_id, title, description,
			room_name, recording_enabled, recording_status, start_time, end_time, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id`,
		sc.CustomerID, req.BatchID, req.ModuleID, req.TutorID, req.Title, req.Description,
		roomName, req.RecordingEnabled, recordingStatus, req.StartTime, req.EndTime, createdBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Get(ctx, sc, id)
}

// Get loads one lecture the principal is allowed to see.
func (r *Repository) Get(ctx context.Context, sc scope.Scope, id int64) (*Lecture, error) {
	pred, args := sc.BatchPredicate("l.batch_id", 3)
	q := `SELECT ` + lectureCols + `, ` + lectureContextCols + lectureFrom +
		` WHERE l.customer_id = $1 AND l.id = $2` + pred
	return scanLecture(r.db.QueryRow(ctx, q, append([]any{sc.CustomerID, id}, args...)...))
}

// List returns a page of lectures with the total, in a single round trip. The count
// comes from a window function rather than a separate COUNT query, and the joins
// supply the batch, course, module and tutor names the client would otherwise have
// to fetch per row.
func (r *Repository) List(
	ctx context.Context,
	sc scope.Scope,
	f ListLectureFilter,
	limit, offset int,
) ([]Lecture, int, error) {
	args := []any{sc.CustomerID, f.BatchID, f.Status, f.Search}
	pred, scopeArgs := sc.BatchPredicate("l.batch_id", len(args)+1)
	args = append(args, scopeArgs...)

	q := `
		SELECT ` + lectureCols + `, ` + lectureContextCols + `, COUNT(*) OVER() AS total` +
		lectureFrom + `
		WHERE l.customer_id = $1
		  AND ($2::int IS NULL OR l.batch_id = $2::int)
		  AND ($3::text = '' OR l.status = $3::text)
		  AND ($4::text = '' OR l.title ILIKE '%' || $4::text || '%')` + pred + `
		ORDER BY l.start_time DESC, l.id DESC
		LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Lecture, 0, limit)
	total := 0
	for rows.Next() {
		var l Lecture
		if err := rows.Scan(
			&l.ID, &l.CustomerID, &l.BatchID, &l.ModuleID, &l.TutorID,
			&l.Title, &l.Description, &l.RoomName, &l.EgressID, &l.Status,
			&l.RecordingEnabled, &l.RecordingStatus, &l.RecordingURL, &l.RecordingObjectKey,
			&l.RecordingNodeID, &l.RecordingDuration, &l.RecordingSize,
			&l.StartTime, &l.ActualStartAt, &l.EndTime,
			&l.CreatedBy, &l.CreatedAt, &l.UpdatedAt,
			&l.BatchName, &l.CourseTitle, &l.ModuleTitle, &l.TutorName,
			&total,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

func (r *Repository) Update(ctx context.Context, sc scope.Scope, id int64, req UpdateLectureRequest) (*Lecture, error) {
	if req.TutorID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $1 AND EXISTS(SELECT 1 FROM dashboard_users du WHERE du.id = t.dashboard_user_id AND du.customer_id = $2))`,
			*req.TutorID, sc.CustomerID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}
	if req.ModuleID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM lectures l
				JOIN batches b ON b.id = l.batch_id
				JOIN course_modules cm ON cm.course_id = b.course_id
				WHERE l.id = $1 AND l.customer_id = $2 AND cm.id = $3
			)`, id, sc.CustomerID, *req.ModuleID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}

	// Editing details is only meaningful before a lecture has run its course.
	tag, err := r.db.Exec(ctx, `
		UPDATE lectures SET
			module_id = $3, tutor_id = $4, title = $5, description = $6,
			recording_enabled = $7, start_time = $8, end_time = $9, updated_at = now()
		WHERE customer_id = $1 AND id = $2 AND status IN ('scheduled', 'live')`,
		sc.CustomerID, id, req.ModuleID, req.TutorID, req.Title, req.Description,
		req.RecordingEnabled, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		// Either it does not exist, or it has already ended.
		if _, err := r.Get(ctx, sc, id); err != nil {
			return nil, err
		}
		return nil, ErrInvalidTransition
	}
	return r.Get(ctx, sc, id)
}

// Delete refuses to remove a lecture that is currently live.
func (r *Repository) Delete(ctx context.Context, sc scope.Scope, id int64) error {
	if _, err := r.Get(ctx, sc, id); err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx,
		`DELETE FROM lectures WHERE customer_id = $1 AND id = $2 AND status <> 'live'`,
		sc.CustomerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidTransition
	}
	return nil
}

// Transition moves a lecture from one status to another, and is the concurrency
// guard for the whole module: the `status = $3` predicate means two simultaneous
// starts result in exactly one winner, so only one egress job is ever launched.
// A caller seeing ErrInvalidTransition knows it lost the race or the state was wrong.
func (r *Repository) Transition(ctx context.Context, sc scope.Scope, id int64, from, to string) (*Lecture, error) {
	if !canTransition(from, to) {
		return nil, ErrInvalidTransition
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE lectures SET
			status = $4::text,
			actual_start_at = CASE WHEN $4::text = 'live' THEN now() ELSE actual_start_at END,
			end_time = CASE WHEN $4::text IN ('ended', 'cancelled') THEN now() ELSE end_time END,
			updated_at = now()
		WHERE customer_id = $1 AND id = $2 AND status = $3::text`,
		sc.CustomerID, id, from, to)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrInvalidTransition
	}
	return r.Get(ctx, sc, id)
}

// SetRecordingTarget stores the real egress id and the object key the recording
// will be written to, so the webhook can find the lecture and the file later.
func (r *Repository) SetRecordingTarget(ctx context.Context, customerID int, id int64, egressID, objectKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lectures
		SET egress_id = $3, recording_object_key = $4, recording_status = $5, updated_at = now()
		WHERE customer_id = $1 AND id = $2`,
		customerID, id, egressID, objectKey, RecordingRecording)
	return err
}

// SetRecordingStatus records progress or failure of the recording pipeline.
func (r *Repository) SetRecordingStatus(ctx context.Context, q pg.Querier, id int64, status string) error {
	if q == nil {
		q = r.db
	}
	_, err := q.Exec(ctx,
		`UPDATE lectures SET recording_status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

// --- Webhook support ---------------------------------------------------------

// ByRoomName looks a lecture up from a LiveKit event. room_name already carries a
// UNIQUE constraint, so this needs no new index. Unlike Get it is not scoped: the
// caller is LiveKit, not a user.
func (r *Repository) ByRoomName(ctx context.Context, q pg.Querier, roomName string, forUpdate bool) (*Lecture, error) {
	if q == nil {
		q = r.db
	}
	sql := `SELECT ` + lectureCols + `, ` + lectureContextCols + lectureFrom + ` WHERE l.room_name = $1`
	if forUpdate {
		// Only the lectures row is locked; the joined tables are read-only here.
		sql += ` FOR UPDATE OF l`
	}
	return scanLecture(q.QueryRow(ctx, sql, roomName))
}

// AttachRecording links a finished recording to the drive node holding it. The
// recording_node_id IS NULL predicate makes the whole webhook idempotent, which
// matters because LiveKit retries deliveries.
func (r *Repository) AttachRecording(
	ctx context.Context,
	q pg.Querier,
	id int64,
	nodeID int,
	url string,
	durationSeconds int,
	sizeBytes int64,
) (bool, error) {
	if q == nil {
		q = r.db
	}
	tag, err := q.Exec(ctx, `
		UPDATE lectures SET
			recording_node_id = $2, recording_url = $3,
			recording_duration_seconds = $4, recording_size_bytes = $5,
			recording_status = $6, updated_at = now()
		WHERE id = $1 AND recording_node_id IS NULL`,
		id, nodeID, url, durationSeconds, sizeBytes, RecordingReady)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ForceEnd closes a lecture whose room finished without anyone pressing End.
func (r *Repository) ForceEnd(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lectures
		SET status = 'ended', end_time = COALESCE(end_time, now()), updated_at = now()
		WHERE id = $1 AND status = 'live'`, id)
	return err
}

// --- Attendance --------------------------------------------------------------

// OpenAttendance records a participant joining. A reconnect opens a new session
// rather than overwriting the first, so the history stays honest.
//
// subject_type is derived here rather than trusted from the caller: a tutor or
// student is a dashboard_users row whose id is also their tutors/students
// primary key (see migration 000012), so "what kind of principal is this" is a
// two-table EXISTS check away and never needs to travel through LiveKit metadata.
func (r *Repository) OpenAttendance(ctx context.Context, lectureID int64, userID int, displayName string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO lecture_attendance (lecture_id, user_id, subject_type, subject_id, display_name)
		VALUES ($1, $2,
			CASE
				WHEN EXISTS (SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $2) THEN 'tutor'
				WHEN EXISTS (SELECT 1 FROM students s WHERE s.dashboard_user_id = $2) THEN 'student'
				ELSE 'admin'
			END,
			$2, $3)`,
		lectureID, userID, displayName)
	return err
}

// CloseAttendance closes the participant's most recent open session.
func (r *Repository) CloseAttendance(ctx context.Context, lectureID int64, userID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lecture_attendance SET
			left_at = now(),
			seconds_present = GREATEST(0, EXTRACT(EPOCH FROM (now() - joined_at))::int)
		WHERE id = (
			SELECT id FROM lecture_attendance
			WHERE lecture_id = $1 AND user_id = $2 AND left_at IS NULL
			ORDER BY joined_at DESC
			LIMIT 1
		)`, lectureID, userID)
	return err
}

// CloseOpenAttendance closes every session still open for a lecture, for when it
// ends while participants are connected.
func (r *Repository) CloseOpenAttendance(ctx context.Context, lectureID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lecture_attendance SET
			left_at = now(),
			seconds_present = GREATEST(0, EXTRACT(EPOCH FROM (now() - joined_at))::int)
		WHERE lecture_id = $1 AND left_at IS NULL`, lectureID)
	return err
}

func (r *Repository) ListAttendance(ctx context.Context, lectureID int64) ([]Attendance, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, lecture_id, user_id, subject_type, subject_id,
		       display_name, joined_at, left_at, seconds_present
		FROM lecture_attendance
		WHERE lecture_id = $1
		ORDER BY joined_at`, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Attendance, 0)
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.LectureID, &a.UserID, &a.SubjectType, &a.SubjectID,
			&a.DisplayName, &a.JoinedAt, &a.LeftAt, &a.SecondsPresent); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// DisplayNameForUser is used to label attendance rows. Every principal's name
// lives directly on dashboard_users (see migration 000012), so this needs no
// join to tutors/students at all.
func (r *Repository) DisplayNameForUser(ctx context.Context, customerID, userID int) string {
	var name string
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(TRIM(first_name || ' ' || last_name), ''), email)
		FROM dashboard_users
		WHERE customer_id = $1 AND id = $2`, customerID, userID).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}
