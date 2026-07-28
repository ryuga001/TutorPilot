package lecture

import "time"

type CreateLectureRequest struct {
	BatchID  int  `json:"batch_id" binding:"required"`
	ModuleID *int `json:"module_id"`
	TutorID  *int `json:"tutor_id"`

	Title       string `json:"title" binding:"required,max=255"`
	Description string `json:"description"`

	RecordingEnabled bool `json:"recording_enabled"`

	StartTime time.Time  `json:"start_time" binding:"required"`
	EndTime   *time.Time `json:"end_time"`
}

type UpdateLectureRequest struct {
	ModuleID *int `json:"module_id"`
	TutorID  *int `json:"tutor_id"`

	Title       string `json:"title" binding:"required,max=255"`
	Description string `json:"description"`

	RecordingEnabled bool `json:"recording_enabled"`

	StartTime time.Time  `json:"start_time" binding:"required"`
	EndTime   *time.Time `json:"end_time"`
}

type ListLectureFilter struct {
	BatchID *int
	Status  string
	Search  string
}

type LectureView struct {
	ID       int64 `json:"id"`
	BatchID  int   `json:"batch_id"`
	ModuleID *int  `json:"module_id,omitempty"`
	TutorID  *int  `json:"tutor_id,omitempty"`

	Title       string `json:"title"`
	Description string `json:"description"`

	Status string `json:"status"`

	RoomName *string `json:"room_name,omitempty"`

	RecordingEnabled  bool    `json:"recording_enabled"`
	RecordingStatus   string  `json:"recording_status"`
	RecordingURL      *string `json:"recording_url,omitempty"`
	RecordingDuration *int    `json:"recording_duration_seconds,omitempty"`
	RecordingSize     *int64  `json:"recording_size_bytes,omitempty"`

	StartTime     time.Time  `json:"start_time"`
	ActualStartAt *time.Time `json:"actual_start_at,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	BatchName   string  `json:"batch_name,omitempty"`
	CourseTitle string  `json:"course_title,omitempty"`
	ModuleTitle *string `json:"module_title,omitempty"`
	TutorName   *string `json:"tutor_name,omitempty"`

	CanPublish bool `json:"can_publish"`
}

type JoinView struct {
	Token      string `json:"token"`
	RoomName   string `json:"room_name"`
	Identity   string `json:"identity"`
	CanPublish bool   `json:"can_publish"`
}

type AttendanceView struct {
	UserID         int        `json:"user_id"`
	SubjectType    string     `json:"subject_type"`
	SubjectID      *int       `json:"subject_id,omitempty"`
	DisplayName    string     `json:"display_name"`
	JoinedAt       time.Time  `json:"joined_at"`
	LeftAt         *time.Time `json:"left_at,omitempty"`
	SecondsPresent *int       `json:"seconds_present,omitempty"`
}

func view(l *Lecture) LectureView {
	return LectureView{
		ID:                l.ID,
		BatchID:           l.BatchID,
		ModuleID:          l.ModuleID,
		TutorID:           l.TutorID,
		Title:             l.Title,
		Description:       l.Description,
		Status:            l.Status,
		RoomName:          l.RoomName,
		RecordingEnabled:  l.RecordingEnabled,
		RecordingStatus:   l.RecordingStatus,
		RecordingURL:      l.RecordingURL,
		RecordingDuration: l.RecordingDuration,
		RecordingSize:     l.RecordingSize,
		StartTime:         l.StartTime,
		ActualStartAt:     l.ActualStartAt,
		EndTime:           l.EndTime,
		CreatedAt:         l.CreatedAt,
		UpdatedAt:         l.UpdatedAt,
		BatchName:         l.BatchName,
		CourseTitle:       l.CourseTitle,
		ModuleTitle:       l.ModuleTitle,
		TutorName:         l.TutorName,
	}
}

func attendanceView(a *Attendance) AttendanceView {
	return AttendanceView{
		UserID:         a.UserID,
		SubjectType:    a.SubjectType,
		SubjectID:      a.SubjectID,
		DisplayName:    a.DisplayName,
		JoinedAt:       a.JoinedAt,
		LeftAt:         a.LeftAt,
		SecondsPresent: a.SecondsPresent,
	}
}
