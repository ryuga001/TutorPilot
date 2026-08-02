package lecture

import (
	"testing"

	model "tutorpilot/internal/modules/admin/model/lecture"
)

func TestViewCopiesPersistedFields(t *testing.T) {
	room := "lecture-abc"
	url := "https://example.test/rec.mp4"
	moduleID, tutorID := 3, 9
	title, tutorName := "Module 1", "Ada Lovelace"

	l := &model.Lecture{
		ID: 42, BatchID: 7, ModuleID: &moduleID, TutorID: &tutorID,
		Title: "Intro", Status: model.StatusEnded,
		RoomName: &room, RecordingEnabled: true, RecordingStatus: model.RecordingReady,
		RecordingURL: &url, BatchName: "B1", CourseTitle: "C1",
		ModuleTitle: &title, TutorName: &tutorName,
	}

	v := View(l)

	if v.ID != l.ID || v.BatchID != l.BatchID || v.Title != l.Title || v.Status != l.Status {
		t.Errorf("View dropped a scalar field: %+v", v)
	}
	if v.RoomName == nil || *v.RoomName != room {
		t.Errorf("RoomName = %v, want %q", v.RoomName, room)
	}
	if v.RecordingURL == nil || *v.RecordingURL != url {
		t.Errorf("RecordingURL = %v, want %q", v.RecordingURL, url)
	}
	if v.CanPublish {
		t.Error("CanPublish must default false; it is decided per-caller, not stored")
	}
}

func TestViewKeepsAbsentOptionalsNil(t *testing.T) {
	v := View(&model.Lecture{ID: 1, Status: model.StatusScheduled})
	if v.RoomName != nil || v.RecordingURL != nil || v.ModuleID != nil || v.TutorID != nil {
		t.Errorf("absent fields should stay nil, got %+v", v)
	}
}

func TestAttendanceViewOfCopiesFields(t *testing.T) {
	subjectID, seconds := 5, 120
	a := &model.Attendance{
		UserID: 5, SubjectType: "student", SubjectID: &subjectID,
		DisplayName: "Ada", SecondsPresent: &seconds,
	}

	v := AttendanceViewOf(a)

	if v.UserID != a.UserID || v.SubjectType != a.SubjectType || v.DisplayName != a.DisplayName {
		t.Errorf("AttendanceViewOf dropped a field: %+v", v)
	}
	if v.SecondsPresent == nil || *v.SecondsPresent != seconds {
		t.Errorf("SecondsPresent = %v, want %d", v.SecondsPresent, seconds)
	}
}
