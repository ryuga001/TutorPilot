package lecture

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrInvalidTransition  = errors.New("lecture is not in a state that allows this")
	ErrLiveKitUnavailable = errors.New("live video is not configured")
	ErrNotLive            = errors.New("this lecture is not live")
	ErrNoRecording        = errors.New("this lecture has no recording")
)

const (
	StatusScheduled = "scheduled"
	StatusLive      = "live"
	StatusEnded     = "ended"
	StatusCancelled = "cancelled"
)

const (
	RecordingNone       = "none"
	RecordingStarting   = "starting"
	RecordingRecording  = "recording"
	RecordingProcessing = "processing"
	RecordingReady      = "ready"
	RecordingFailed     = "failed"
)

var transitions = map[string][]string{
	StatusScheduled: {StatusLive, StatusCancelled},
	StatusLive:      {StatusEnded},
	StatusEnded:     nil,
	StatusCancelled: nil,
}

func CanTransition(from, to string) bool {
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

type Lecture struct {
	ID         int64
	CustomerID int

	BatchID  int
	ModuleID *int
	TutorID  *int

	Title       string
	Description string

	RoomName *string
	EgressID *string

	Status string

	RecordingEnabled   bool
	RecordingStatus    string
	RecordingURL       *string
	RecordingObjectKey *string
	RecordingNodeID    *int
	RecordingDuration  *int
	RecordingSize      *int64

	StartTime     time.Time
	ActualStartAt *time.Time
	EndTime       *time.Time

	CreatedBy *int
	CreatedAt time.Time
	UpdatedAt time.Time

	BatchName   string
	CourseTitle string
	ModuleTitle *string
	TutorName   *string
}

type Attendance struct {
	ID             int
	LectureID      int64
	UserID         int
	SubjectType    string
	SubjectID      *int
	DisplayName    string
	JoinedAt       time.Time
	LeftAt         *time.Time
	SecondsPresent *int
}
