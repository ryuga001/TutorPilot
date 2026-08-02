package lecture

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dto "tutorpilot/internal/modules/admin/dto/lecture"
	model "tutorpilot/internal/modules/admin/model/lecture"
	"tutorpilot/internal/modules/admin/scope"
	"tutorpilot/internal/pkg/pg"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Pool() *pgxpool.Pool { return r.db }

const lectureCols = `
	l.id, l.customer_id, l.batch_id, l.module_id, l.tutor_id,
	l.title, l.description, l.room_name, l.egress_id, l.status,
	l.recording_enabled, l.recording_status, l.recording_url, l.recording_object_key,
	l.recording_node_id, l.recording_duration_seconds, l.recording_size_bytes,
	l.start_time, l.actual_start_at, l.end_time,
	l.created_by, l.created_at, l.updated_at`

const lectureContextCols = `
	b.name, c.title, cm.title, NULLIF(TRIM(COALESCE(tdu.first_name, '') || ' ' || COALESCE(tdu.last_name, '')), '')`

const lectureFrom = `
	FROM lectures l
	JOIN batches b ON b.id = l.batch_id
	JOIN courses c ON c.id = b.course_id
	LEFT JOIN course_modules cm ON cm.id = l.module_id
	LEFT JOIN tutors t ON t.dashboard_user_id = l.tutor_id
	LEFT JOIN dashboard_users tdu ON tdu.id = t.dashboard_user_id`

func scanLecture(row pgx.Row) (*model.Lecture, error) {
	l := &model.Lecture{}
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
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *Repository) Create(
	ctx context.Context,
	sc scope.Scope,
	createdBy int,
	roomName string,
	req dto.CreateLectureRequest,
) (*model.Lecture, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	pred, args := sc.BatchPredicate("b.id", 3)
	batchQ := `SELECT EXISTS(SELECT 1 FROM batches b WHERE b.id = $1 AND b.customer_id = $2` + pred + `)`
	var ok bool
	if err := tx.QueryRow(ctx, batchQ, append([]any{req.BatchID, sc.CustomerID}, args...)...).Scan(&ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, model.ErrNotFound
	}

	if req.TutorID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $1 AND EXISTS(SELECT 1 FROM dashboard_users du WHERE du.id = t.dashboard_user_id AND du.customer_id = $2))`,
			*req.TutorID, sc.CustomerID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, model.ErrNotFound
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
			return nil, model.ErrNotFound
		}
	}

	recordingStatus := model.RecordingNone
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

func (r *Repository) Get(ctx context.Context, sc scope.Scope, id int64) (*model.Lecture, error) {
	pred, args := sc.BatchPredicate("l.batch_id", 3)
	q := `SELECT ` + lectureCols + `, ` + lectureContextCols + lectureFrom +
		` WHERE l.customer_id = $1 AND l.id = $2` + pred
	return scanLecture(r.db.QueryRow(ctx, q, append([]any{sc.CustomerID, id}, args...)...))
}

func (r *Repository) List(
	ctx context.Context,
	sc scope.Scope,
	f dto.ListLectureFilter,
	limit, offset int,
) ([]model.Lecture, int, error) {
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

	out := make([]model.Lecture, 0, limit)
	total := 0
	for rows.Next() {
		var l model.Lecture
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

func (r *Repository) Update(ctx context.Context, sc scope.Scope, id int64, req dto.UpdateLectureRequest) (*model.Lecture, error) {
	if req.TutorID != nil {
		var ok bool
		if err := r.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tutors t WHERE t.dashboard_user_id = $1 AND EXISTS(SELECT 1 FROM dashboard_users du WHERE du.id = t.dashboard_user_id AND du.customer_id = $2))`,
			*req.TutorID, sc.CustomerID).Scan(&ok); err != nil {
			return nil, err
		}
		if !ok {
			return nil, model.ErrNotFound
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
			return nil, model.ErrNotFound
		}
	}

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
		if _, err := r.Get(ctx, sc, id); err != nil {
			return nil, err
		}
		return nil, model.ErrInvalidTransition
	}
	return r.Get(ctx, sc, id)
}

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
		return model.ErrInvalidTransition
	}
	return nil
}

func (r *Repository) Transition(ctx context.Context, sc scope.Scope, id int64, from, to string) (*model.Lecture, error) {
	if !model.CanTransition(from, to) {
		return nil, model.ErrInvalidTransition
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
		return nil, model.ErrInvalidTransition
	}
	return r.Get(ctx, sc, id)
}

func (r *Repository) SetRecordingTarget(ctx context.Context, customerID int, id int64, egressID, objectKey string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lectures
		SET egress_id = $3, recording_object_key = $4, recording_status = $5, updated_at = now()
		WHERE customer_id = $1 AND id = $2`,
		customerID, id, egressID, objectKey, model.RecordingRecording)
	return err
}

func (r *Repository) SetRecordingStatus(ctx context.Context, q pg.Querier, id int64, status string) error {
	if q == nil {
		q = r.db
	}
	_, err := q.Exec(ctx,
		`UPDATE lectures SET recording_status = $2, updated_at = now() WHERE id = $1`, id, status)
	return err
}

func (r *Repository) ByRoomName(ctx context.Context, q pg.Querier, roomName string, forUpdate bool) (*model.Lecture, error) {
	if q == nil {
		q = r.db
	}
	sql := `SELECT ` + lectureCols + `, ` + lectureContextCols + lectureFrom + ` WHERE l.room_name = $1`
	if forUpdate {
		sql += ` FOR UPDATE OF l`
	}
	return scanLecture(q.QueryRow(ctx, sql, roomName))
}

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
		id, nodeID, url, durationSeconds, sizeBytes, model.RecordingReady)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) ForceEnd(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lectures
		SET status = 'ended', end_time = COALESCE(end_time, now()), updated_at = now()
		WHERE id = $1 AND status = 'live'`, id)
	return err
}

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

func (r *Repository) CloseOpenAttendance(ctx context.Context, lectureID int64) error {
	_, err := r.db.Exec(ctx, `
		UPDATE lecture_attendance SET
			left_at = now(),
			seconds_present = GREATEST(0, EXTRACT(EPOCH FROM (now() - joined_at))::int)
		WHERE lecture_id = $1 AND left_at IS NULL`, lectureID)
	return err
}

func (r *Repository) ListAttendance(ctx context.Context, lectureID int64) ([]model.Attendance, error) {
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

	out := make([]model.Attendance, 0)
	for rows.Next() {
		var a model.Attendance
		if err := rows.Scan(&a.ID, &a.LectureID, &a.UserID, &a.SubjectType, &a.SubjectID,
			&a.DisplayName, &a.JoinedAt, &a.LeftAt, &a.SecondsPresent); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

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
