package batches

import "time"

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

const (
	NodeFolder = "folder"
	NodeFile   = "file"
)


type Batch struct {
	ID          int
	CustomerID  int
	CourseID    int
	Name        string
	Status      string
	PublishedAt *time.Time
	CreatedBy   *int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ModuleAssignment struct {
	CourseModuleID  int
	ModuleTitle     string
	ModulePosition  int
	TutorID         *int
	TutorFirstName  *string
	TutorLastName   *string
	TutorEmail      *string
	StartDate       *time.Time
	ExpectedEndDate *time.Time
}

type TutorSummary struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
}

type StudentSummary struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

type DriveNode struct {
	ID          int
	BatchID     int
	ParentID    *int
	Name        string
	NodeType    string
	ObjectKey   *string
	ContentType string
	SizeBytes   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// --- Public JSON views -------------------------------------------------------

type BatchView struct {
	ID           int                    `json:"id"`
	CourseID     int                    `json:"course_id"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	PublishedAt  *time.Time             `json:"published_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Modules      []ModuleAssignmentView `json:"modules,omitempty"`
	TutorCount   int                    `json:"tutor_count"`
	StudentCount int                    `json:"student_count"`
}

type ModuleAssignmentView struct {
	CourseModuleID  int               `json:"course_module_id"`
	ModuleTitle     string            `json:"module_title"`
	ModulePosition  int               `json:"module_position"`
	Tutor           *TutorSummaryView `json:"tutor,omitempty"`
	StartDate       string            `json:"start_date,omitempty"`
	ExpectedEndDate string            `json:"expected_end_date,omitempty"`
}

type TutorSummaryView struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type StudentSummaryView struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Phone     string `json:"phone_no"`
}

type DriveNodeView struct {
	ID          int       `json:"id"`
	ParentID    *int      `json:"parent_id,omitempty"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	URL         string    `json:"url,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	SizeBytes   int64     `json:"size_bytes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ImportResult summarizes a CSV student-enrollment import.
type ImportResult struct {
	Imported int          `json:"imported"`
	Skipped  []SkippedRow `json:"skipped"`
}

type SkippedRow struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}
