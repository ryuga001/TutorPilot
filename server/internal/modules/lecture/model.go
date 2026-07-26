package lecture

import "time"

const (
	LectureStatusScheduled = "scheduled"
	LectureStatusLive      = "live"
	LectureStatusEnded     = "ended"
)

type Lecture struct {
	ID         int64 `json:"id"`
	CustomerID int   `json:"customerId"`

	BatchID  int  `json:"batchId"`
	ModuleID *int `json:"moduleId"`
	TutorID  *int `json:"tutorId"`

	Title       string `json:"title"`
	Description string `json:"description"`

	RoomName *string `json:"roomName"`

	EgressID *string `json:"-"`

	Status string `json:"status"`

	RecordingEnabled bool    `json:"recordingEnabled"`
	RecordingURL     *string `json:"recordingUrl"`

	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`

	CreatedBy *int `json:"createdBy"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
