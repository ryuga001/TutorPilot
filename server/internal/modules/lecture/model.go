package lecture

import "time"

// Lecture lifecycle. A lecture is scheduled, may be cancelled before it starts,
// and once live can only end.
const (
	StatusScheduled = "scheduled"
	StatusLive      = "live"
	StatusEnded     = "ended"
	StatusCancelled = "cancelled"
)

// Recording lifecycle. Ending a lecture does not produce a file: egress finalises
// asynchronously, so the recording passes through processing before it is ready.
const (
	RecordingNone       = "none"
	RecordingStarting   = "starting"
	RecordingRecording  = "recording"
	RecordingProcessing = "processing"
	RecordingReady      = "ready"
	RecordingFailed     = "failed"
)

// transitions is the whole state machine. Anything absent is refused, which is
// what stops an ended lecture from being started again.
var transitions = map[string][]string{
	StatusScheduled: {StatusLive, StatusCancelled},
	StatusLive:      {StatusEnded},
	StatusEnded:     nil,
	StatusCancelled: nil,
}

// canTransition reports whether a lecture may move between two states.
func canTransition(from, to string) bool {
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

	// Denormalised for listings, so the client does not have to fetch the batch,
	// course, module and tutor separately for every row.
	BatchName   string
	CourseTitle string
	ModuleTitle *string
	TutorName   *string
}

// Attendance is one participant's presence in one lecture session.
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
