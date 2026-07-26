package lecture

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("not found")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

const lectureCols = `
id,
customer_id,
batch_id,
module_id,
tutor_id,
title,
description,
room_name,
egress_id,
status,
recording_enabled,
recording_url,
start_time,
end_time,
created_by,
created_at,
updated_at`

func scanLecture(row pgx.Row) (*Lecture, error) {
	l := &Lecture{}

	err := row.Scan(
		&l.ID,
		&l.CustomerID,
		&l.BatchID,
		&l.ModuleID,
		&l.TutorID,
		&l.Title,
		&l.Description,
		&l.RoomName,
		&l.EgressID,
		&l.Status,
		&l.RecordingEnabled,
		&l.RecordingURL,
		&l.StartTime,
		&l.EndTime,
		&l.CreatedBy,
		&l.CreatedAt,
		&l.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return l, nil
}

func (r *Repository) SetEgressID(ctx context.Context, customerID, lectureID int, egressID string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE lectures SET egress_id = $3, updated_at = now() WHERE customer_id = $1 AND id = $2`,
		customerID, lectureID, egressID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CreateLecture(
	ctx context.Context,
	customerID,
	createdBy int,
	roomName string,
	req CreateLectureRequest,
) (*Lecture, error) {

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var exists bool

	// batch belongs to tenant
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM batches
			WHERE id = $1
			  AND customer_id = $2
		)`,
		req.BatchID,
		customerID,
	).Scan(&exists); err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrNotFound
	}

	// tutor belongs to tenant
	if req.TutorID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1
				FROM tutors
				WHERE id = $1
				  AND customer_id = $2
			)`,
			*req.TutorID,
			customerID,
		).Scan(&exists); err != nil {
			return nil, err
		}

		if !exists {
			return nil, ErrNotFound
		}
	}

	// module belongs to batch's course
	if req.ModuleID != nil {
		if err := tx.QueryRow(ctx,
			`
			SELECT EXISTS(
				SELECT 1
				FROM batches b
				JOIN course_modules cm
					ON cm.course_id = b.course_id
				WHERE b.id = $1
				  AND b.customer_id = $2
				  AND cm.id = $3
			)
			`,
			req.BatchID,
			customerID,
			*req.ModuleID,
		).Scan(&exists); err != nil {
			return nil, err
		}

		if !exists {
			return nil, ErrNotFound
		}
	}

	const q = `
	INSERT INTO lectures (
		customer_id,
		batch_id,
		module_id,
		tutor_id,
		title,
		description,
		room_name,
		recording_enabled,
		start_time,
		end_time,
		created_by
	)
	VALUES (
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11
	)
	RETURNING ` + lectureCols

	lecture, err := scanLecture(tx.QueryRow(
		ctx,
		q,
		customerID,
		req.BatchID,
		req.ModuleID,
		req.TutorID,
		req.Title,
		req.Description,
		roomName,
		req.RecordingEnabled,
		req.StartTime,
		req.EndTime,
		createdBy,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, err
		}
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return lecture, nil
}

func (r *Repository) ListLectures(
	ctx context.Context,
	customerID int,
	batchID *int,
	status, search string,
	limit, offset int,
) ([]Lecture, int, error) {

	const countQ = `
	SELECT COUNT(*)
	FROM lectures
	WHERE customer_id = $1
	  AND ($2::int IS NULL OR batch_id = $2::int)
	  AND ($3::text = '' OR status = $3::text)
	  AND ($4::text = '' OR title ILIKE '%' || $4::text || '%')
	`

	var total int
	if err := r.db.QueryRow(
		ctx,
		countQ,
		customerID,
		batchID,
		status,
		search,
	).Scan(&total); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []Lecture{}, 0, ErrNotFound
		}
		return nil, 0, err
	}

	const q = `
	SELECT ` + lectureCols + `
	FROM lectures
	WHERE customer_id = $1
	  AND ($2::int IS NULL OR batch_id = $2::int)
	  AND ($3::text = '' OR status = $3::text)
	  AND ($4::text = '' OR title ILIKE '%' || $4::text || '%')
	ORDER BY start_time DESC
	LIMIT $5 OFFSET $6
	`

	rows, err := r.db.Query(
		ctx,
		q,
		customerID,
		batchID,
		status,
		search,
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return []Lecture{}, 0, ErrNotFound
		}
		return nil, 0, err
	}
	defer rows.Close()

	lectures := make([]Lecture, 0)

	for rows.Next() {
		var l Lecture

		if err := rows.Scan(
			&l.ID,
			&l.CustomerID,
			&l.BatchID,
			&l.ModuleID,
			&l.TutorID,
			&l.Title,
			&l.Description,
			&l.RoomName,
			&l.EgressID,
			&l.Status,
			&l.RecordingEnabled,
			&l.RecordingURL,
			&l.StartTime,
			&l.EndTime,
			&l.CreatedBy,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}

		lectures = append(lectures, l)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return lectures, total, nil
}

func (r *Repository) GetLecture(
	ctx context.Context,
	customerID,
	lectureID int,
) (*Lecture, error) {

	const q = `
	SELECT ` + lectureCols + `
	FROM lectures
	WHERE customer_id = $1
	  AND id = $2
	`

	return scanLecture(
		r.db.QueryRow(
			ctx,
			q,
			customerID,
			lectureID,
		),
	)
}

func (r *Repository) UpdateLecture(
	ctx context.Context,
	customerID,
	lectureID int,
	req UpdateLectureRequest,
) (*Lecture, error) {

	var exists bool

	// validate tutor
	if req.TutorID != nil {
		if err := r.db.QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1
				FROM tutors
				WHERE id = $1
				  AND customer_id = $2
			)`,
			*req.TutorID,
			customerID,
		).Scan(&exists); err != nil {
			return nil, err
		}

		if !exists {
			return nil, ErrNotFound
		}
	}

	// validate module belongs to lecture's batch course
	if req.ModuleID != nil {
		if err := r.db.QueryRow(ctx,
			`
			SELECT EXISTS(
				SELECT 1
				FROM lectures l
				JOIN batches b
					ON b.id = l.batch_id
				JOIN course_modules cm
					ON cm.course_id = b.course_id
				WHERE l.id = $1
				  AND l.customer_id = $2
				  AND cm.id = $3
			)
			`,
			lectureID,
			customerID,
			*req.ModuleID,
		).Scan(&exists); err != nil {
			return nil, err
		}

		if !exists {
			return nil, ErrNotFound
		}
	}

	const q = `
	UPDATE lectures
	SET
		module_id = $3,
		tutor_id = $4,
		title = $5,
		description = $6,
		recording_enabled = $7,
		start_time = $8,
		end_time = $9,
		updated_at = now()
	WHERE customer_id = $1
	  AND id = $2
	RETURNING ` + lectureCols

	return scanLecture(
		r.db.QueryRow(
			ctx,
			q,
			customerID,
			lectureID,
			req.ModuleID,
			req.TutorID,
			req.Title,
			req.Description,
			req.RecordingEnabled,
			req.StartTime,
			req.EndTime,
		),
	)
}

func (r *Repository) DeleteLecture(
	ctx context.Context,
	customerID,
	lectureID int,
) error {

	tag, err := r.db.Exec(
		ctx,
		`
		DELETE
		FROM lectures
		WHERE customer_id = $1
		  AND id = $2
		`,
		customerID,
		lectureID,
	)

	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) UpdateStatus(
	ctx context.Context,
	customerID,
	lectureID int,
	status string,
	recordingURL *string,
) (*Lecture, error) {
	const q = `
	UPDATE lectures
	SET
		status = $3,
		end_time = CASE WHEN $3 = 'ended' THEN now() ELSE end_time END,
		recording_url = COALESCE($4, recording_url),
		updated_at = now()
	WHERE customer_id = $1
	  AND id = $2
	RETURNING ` + lectureCols

	return scanLecture(
		r.db.QueryRow(
			ctx,
			q,
			customerID,
			lectureID,
			status,
			recordingURL,
		),
	)
}


func (r *Repository) IsUserAuthorizedForLecture(
	ctx context.Context,
	customerID int,
	lectureID int,
	email string,
	role string,
) (bool, string, error) {
	if role == "admin" || role == "super_admin" || role == "Admin" || role == "Super Admin" {
		var name string
		err := r.db.QueryRow(ctx,
			`SELECT c.first_name || ' ' || c.last_name 
			 FROM dashboard_users du
			 JOIN customers c ON c.id = du.customer_id
			 WHERE du.email = $1`,
			email,
		).Scan(&name)
		if err != nil {
			name = "Admin"
		}
		return true, name, nil
	}

	// check if user is the assigned tutor
	var tutorExists bool
	var tutorName string
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM lectures l
			JOIN tutors t ON t.id = l.tutor_id
			WHERE l.customer_id = $1 AND l.id = $2 AND t.email = $3
		), COALESCE((SELECT t.first_name || ' ' || t.last_name FROM lectures l JOIN tutors t ON t.id = l.tutor_id WHERE l.customer_id = $1 AND l.id = $2 AND t.email = $3 LIMIT 1), '')`,
		customerID,
		lectureID,
		email,
	).Scan(&tutorExists, &tutorName)
	if err != nil {
		return false, "", err
	}
	if tutorExists {
		return true, tutorName, nil
	}

	// check if user is an enrolled student
	var studentExists bool
	var studentName string
	err = r.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM lectures l
			JOIN batch_students bs ON bs.batch_id = l.batch_id
			JOIN students s ON s.id = bs.student_id
			WHERE l.customer_id = $1 AND l.id = $2 AND s.email = $3
		), COALESCE((SELECT s.first_name || ' ' || s.last_name FROM lectures l JOIN batch_students bs ON bs.batch_id = l.batch_id JOIN students s ON s.id = bs.student_id WHERE l.customer_id = $1 AND l.id = $2 AND s.email = $3 LIMIT 1), '')`,
		customerID,
		lectureID,
		email,
	).Scan(&studentExists, &studentName)
	if err != nil {
		return false, "", err
	}
	if studentExists {
		return true, studentName, nil
	}

	return false, "", nil
}
