package courses

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

type Course struct {
	ID            int
	CustomerID    int
	Title         string
	Slug          string
	Summary       string
	DescriptionMD string
	ThumbnailKey  *string
	Status        string
	PublishedAt   *time.Time
	CreatedBy     *int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CourseModule struct {
	ID       int
	CourseID int
	Title    string
	Position int
	Lessons  []Lesson
}

type Lesson struct {
	ID        int
	ModuleID  int
	Title     string
	ContentMD string
	Position  int
}

type Resource struct {
	ID          int
	CourseID    int
	LessonID    *int
	Name        string
	ObjectKey   string
	ContentType string
	SizeBytes   int64
	CreatedAt   time.Time
}

type CourseView struct {
	ID            int          `json:"id"`
	Title         string       `json:"title"`
	Slug          string       `json:"slug"`
	Summary       string       `json:"summary"`
	DescriptionMD string       `json:"description_md"`
	ThumbnailURL  string       `json:"thumbnail_url,omitempty"`
	Status        string       `json:"status"`
	PublishedAt   *time.Time   `json:"published_at,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	Modules       []ModuleView `json:"modules,omitempty"`
}

type ModuleView struct {
	ID       int          `json:"id"`
	Title    string       `json:"title"`
	Position int          `json:"position"`
	Lessons  []LessonView `json:"lessons"`
}

type LessonView struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
	Position  int    `json:"position"`
}

type ResourceView struct {
	ID          int       `json:"id"`
	LessonID    *int      `json:"lesson_id,omitempty"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}
