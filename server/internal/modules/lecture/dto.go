package lecture

import "time"

type CreateLectureRequest struct {
	BatchID  int  `json:"batchId" binding:"required"`
	ModuleID *int `json:"moduleId"`
	TutorID  *int `json:"tutorId"`

	Title       string `json:"title" binding:"required,max=255"`
	Description string `json:"description"`

	RecordingEnabled bool `json:"recordingEnabled"`

	StartTime time.Time  `json:"startTime" binding:"required"`
	EndTime   *time.Time `json:"endTime"`
}

type UpdateLectureRequest struct {
	ModuleID *int `json:"moduleId"`
	TutorID  *int `json:"tutorId"`

	Title       string `json:"title" binding:"required,max=255"`
	Description string `json:"description"`

	RecordingEnabled bool `json:"recordingEnabled"`

	StartTime time.Time  `json:"startTime" binding:"required"`
	EndTime   *time.Time `json:"endTime"`
}

type ListLectureRequest struct {
	BatchID *int
	Status  string
	Search  string

	Page     int
	PageSize int
}

type LectureResponse struct {
	ID       int64 `json:"id"`
	BatchID  int   `json:"batchId"`
	ModuleID *int  `json:"moduleId"`
	TutorID  *int  `json:"tutorId"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Status string `json:"status"`

	RoomName *string `json:"roomName,omitempty"`

	RecordingEnabled bool    `json:"recordingEnabled"`
	RecordingURL     *string `json:"recordingUrl,omitempty"`

	StartTime string  `json:"startTime"`
	EndTime   *string `json:"endTime,omitempty"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ListLectureResponse struct {
	Items []LectureResponse `json:"items"`
	Total int               `json:"total"`
	Page  int               `json:"page"`
	Limit int               `json:"limit"`
}
