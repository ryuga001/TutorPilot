package courses

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrSlugTaken = errors.New("slug already exists")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

const courseCols = `id, customer_id, title, slug, summary, description_md,
	thumbnail_key, status, published_at, created_by, created_at, updated_at`

func scanCourse(row pgx.Row) (*Course, error) {
	c := &Course{}
	err := row.Scan(&c.ID, &c.CustomerID, &c.Title, &c.Slug, &c.Summary, &c.DescriptionMD,
		&c.ThumbnailKey, &c.Status, &c.PublishedAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}


func (r *Repository) CreateCourse(ctx context.Context, customerID, createdBy int, title, slug, summary, descMD string) (*Course, error) {
	const q = `INSERT INTO courses (customer_id, title, slug, summary, description_md, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING ` + courseCols
	c, err := scanCourse(r.db.QueryRow(ctx, q, customerID, title, slug, summary, descMD, createdBy))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}
	return c, nil
}

func (r *Repository) ListCourses(ctx context.Context, customerID int, status, search string, limit, offset int) ([]Course, int, error) {
	const countQ = `SELECT COUNT(*) FROM courses
		WHERE customer_id = $1 AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR title ILIKE '%' || $3 || '%')`
	var total int
	if err := r.db.QueryRow(ctx, countQ, customerID, status, search).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `SELECT ` + courseCols + ` FROM courses
		WHERE customer_id = $1 AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR title ILIKE '%' || $3 || '%')
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`
	rows, err := r.db.Query(ctx, q, customerID, status, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]Course, 0)
	for rows.Next() {
		c := Course{}
		if err := rows.Scan(&c.ID, &c.CustomerID, &c.Title, &c.Slug, &c.Summary, &c.DescriptionMD,
			&c.ThumbnailKey, &c.Status, &c.PublishedAt, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetCourse(ctx context.Context, customerID, id int) (*Course, error) {
	const q = `SELECT ` + courseCols + ` FROM courses WHERE customer_id = $1 AND id = $2`
	return scanCourse(r.db.QueryRow(ctx, q, customerID, id))
}

func (r *Repository) UpdateCourse(ctx context.Context, customerID, id int, title, summary, descMD string) (*Course, error) {
	const q = `UPDATE courses SET title = $3, summary = $4, description_md = $5, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + courseCols
	return scanCourse(r.db.QueryRow(ctx, q, customerID, id, title, summary, descMD))
}

func (r *Repository) SetStatus(ctx context.Context, customerID, id int, status string, publishedAt *time.Time) (*Course, error) {
	const q = `UPDATE courses SET status = $3, published_at = $4, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + courseCols
	return scanCourse(r.db.QueryRow(ctx, q, customerID, id, status, publishedAt))
}

func (r *Repository) SetThumbnail(ctx context.Context, customerID, id int, key string) (*Course, error) {
	const q = `UPDATE courses SET thumbnail_key = $3, updated_at = now()
		WHERE customer_id = $1 AND id = $2 RETURNING ` + courseCols
	return scanCourse(r.db.QueryRow(ctx, q, customerID, id, key))
}

func (r *Repository) DeleteCourse(ctx context.Context, customerID, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM courses WHERE customer_id = $1 AND id = $2`, customerID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) LoadModules(ctx context.Context, courseID int) ([]CourseModule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, course_id, title, position FROM course_modules WHERE course_id = $1 ORDER BY position, id`,
		courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mods := make([]CourseModule, 0)
	idx := map[int]int{}
	for rows.Next() {
		m := CourseModule{Lessons: []Lesson{}}
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Position); err != nil {
			return nil, err
		}
		idx[m.ID] = len(mods)
		mods = append(mods, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	lrows, err := r.db.Query(ctx,
		`SELECT l.id, l.module_id, l.title, l.content_md, l.position
		 FROM course_lessons l JOIN course_modules m ON m.id = l.module_id
		 WHERE m.course_id = $1 ORDER BY l.position, l.id`, courseID)
	if err != nil {
		return nil, err
	}
	defer lrows.Close()
	for lrows.Next() {
		l := Lesson{}
		if err := lrows.Scan(&l.ID, &l.ModuleID, &l.Title, &l.ContentMD, &l.Position); err != nil {
			return nil, err
		}
		if i, ok := idx[l.ModuleID]; ok {
			mods[i].Lessons = append(mods[i].Lessons, l)
		}
	}
	return mods, lrows.Err()
}

func (r *Repository) moduleOwned(ctx context.Context, customerID, moduleID int) (int, error) {
	var courseID int
	err := r.db.QueryRow(ctx,
		`SELECT m.course_id FROM course_modules m JOIN courses c ON c.id = m.course_id
		 WHERE m.id = $1 AND c.customer_id = $2`, moduleID, customerID).Scan(&courseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return courseID, err
}

func (r *Repository) lessonOwned(ctx context.Context, customerID, lessonID int) error {
	var one int
	err := r.db.QueryRow(ctx,
		`SELECT 1 FROM course_lessons l
		 JOIN course_modules m ON m.id = l.module_id
		 JOIN courses c ON c.id = m.course_id
		 WHERE l.id = $1 AND c.customer_id = $2`, lessonID, customerID).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *Repository) CreateModule(ctx context.Context, courseID int, title string, position int) (*CourseModule, error) {
	m := &CourseModule{Lessons: []Lesson{}}
	err := r.db.QueryRow(ctx,
		`INSERT INTO course_modules (course_id, title, position) VALUES ($1, $2, $3)
		 RETURNING id, course_id, title, position`, courseID, title, position).
		Scan(&m.ID, &m.CourseID, &m.Title, &m.Position)
	return m, err
}

func (r *Repository) UpdateModule(ctx context.Context, moduleID int, title string, position int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE course_modules SET title = $2, position = $3, updated_at = now() WHERE id = $1`,
		moduleID, title, position)
	return err
}

func (r *Repository) DeleteModule(ctx context.Context, moduleID int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM course_modules WHERE id = $1`, moduleID)
	return err
}

func (r *Repository) CreateLesson(ctx context.Context, moduleID int, title, contentMD string, position int) (*Lesson, error) {
	l := &Lesson{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO course_lessons (module_id, title, content_md, position) VALUES ($1, $2, $3, $4)
		 RETURNING id, module_id, title, content_md, position`, moduleID, title, contentMD, position).
		Scan(&l.ID, &l.ModuleID, &l.Title, &l.ContentMD, &l.Position)
	return l, err
}

func (r *Repository) UpdateLesson(ctx context.Context, lessonID int, title, contentMD string, position int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE course_lessons SET title = $2, content_md = $3, position = $4, updated_at = now() WHERE id = $1`,
		lessonID, title, contentMD, position)
	return err
}

func (r *Repository) DeleteLesson(ctx context.Context, lessonID int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM course_lessons WHERE id = $1`, lessonID)
	return err
}

func (r *Repository) CreateResource(ctx context.Context, courseID int, lessonID *int, createdBy int, name, key, contentType string, size int64) (*Resource, error) {
	res := &Resource{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO course_resources (course_id, lesson_id, name, object_key, content_type, size_bytes, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, course_id, lesson_id, name, object_key, content_type, size_bytes, created_at`,
		courseID, lessonID, name, key, contentType, size, createdBy).
		Scan(&res.ID, &res.CourseID, &res.LessonID, &res.Name, &res.ObjectKey, &res.ContentType, &res.SizeBytes, &res.CreatedAt)
	return res, err
}

func (r *Repository) ListResources(ctx context.Context, courseID, limit, offset int) ([]Resource, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM course_resources WHERE course_id = $1`, courseID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, course_id, lesson_id, name, object_key, content_type, size_bytes, created_at
		 FROM course_resources WHERE course_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		courseID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := make([]Resource, 0)
	for rows.Next() {
		res := Resource{}
		if err := rows.Scan(&res.ID, &res.CourseID, &res.LessonID, &res.Name, &res.ObjectKey,
			&res.ContentType, &res.SizeBytes, &res.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, res)
	}
	return out, total, rows.Err()
}

func (r *Repository) GetResource(ctx context.Context, customerID, courseID, resourceID int) (*Resource, error) {
	res := &Resource{}
	err := r.db.QueryRow(ctx,
		`SELECT rs.id, rs.course_id, rs.lesson_id, rs.name, rs.object_key, rs.content_type, rs.size_bytes, rs.created_at
		 FROM course_resources rs JOIN courses c ON c.id = rs.course_id
		 WHERE rs.id = $1 AND rs.course_id = $2 AND c.customer_id = $3`,
		resourceID, courseID, customerID).
		Scan(&res.ID, &res.CourseID, &res.LessonID, &res.Name, &res.ObjectKey, &res.ContentType, &res.SizeBytes, &res.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return res, err
}

func (r *Repository) DeleteResource(ctx context.Context, resourceID int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM course_resources WHERE id = $1`, resourceID)
	return err
}
