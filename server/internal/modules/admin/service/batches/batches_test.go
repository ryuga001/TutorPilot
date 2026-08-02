package batches

import (
	"encoding/json"
	"testing"
	"time"

	model "tutorpilot/internal/modules/admin/model/batches"
	"tutorpilot/internal/modules/notification"
	"tutorpilot/internal/pkg/events"
)

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty(nil); got != "" {
		t.Errorf("firstNonEmpty(nil) = %q, want empty", got)
	}
	v := "Ada"
	if got := firstNonEmpty(&v); got != "Ada" {
		t.Errorf("firstNonEmpty(&\"Ada\") = %q, want %q", got, "Ada")
	}
	empty := ""
	if got := firstNonEmpty(&empty); got != "" {
		t.Errorf("firstNonEmpty(&\"\") = %q, want empty", got)
	}
}

func decodeEmail(t *testing.T, evt events.Event) events.EmailRequested {
	t.Helper()
	var p events.EmailRequested
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return p
}

func TestStudentEnrolmentEventsOnePerStudent(t *testing.T) {
	b := &model.Batch{ID: 1, CustomerID: 4, Name: "Batch A"}
	students := []model.StudentSummary{
		{Email: "a@example.com", FirstName: "Ann"},
		{Email: "b@example.com", FirstName: "Ben"},
	}

	evts, err := studentEnrolmentEvents(b, "Algebra", students, time.Now())
	if err != nil {
		t.Fatalf("studentEnrolmentEvents: %v", err)
	}

	if len(evts) != 2 {
		t.Fatalf("got %d events, want one per student", len(evts))
	}
	for i, evt := range evts {
		if evt.CustomerID != b.CustomerID {
			t.Errorf("event %d: CustomerID = %d, want %d", i, evt.CustomerID, b.CustomerID)
		}
		if evt.Type != events.TypeEmailRequested {
			t.Errorf("event %d: Type = %q, want an ordinary (replayable) email", i, evt.Type)
		}
		p := decodeEmail(t, evt)
		if p.To != students[i].Email {
			t.Errorf("event %d: To = %q, want %q", i, p.To, students[i].Email)
		}
		if p.TemplateName != notification.TmplBatchStudentEnrollment {
			t.Errorf("event %d: TemplateName = %q", i, p.TemplateName)
		}
		if p.Vars["batch_name"] != b.Name || p.Vars["course_name"] != "Algebra" {
			t.Errorf("event %d: vars = %v", i, p.Vars)
		}
	}
}

func TestStudentEnrolmentEventsPayloadIsSelfContained(t *testing.T) {
	b := &model.Batch{ID: 1, CustomerID: 4, Name: "Batch A"}
	students := []model.StudentSummary{{ID: 99, Email: "a@example.com", FirstName: "Ann"}}

	evts, err := studentEnrolmentEvents(b, "Algebra", students, time.Now())
	if err != nil {
		t.Fatalf("studentEnrolmentEvents: %v", err)
	}

	p := decodeEmail(t, evts[0])
	if p.To == "" {
		t.Error("payload has no recipient; the worker cannot look one up post-split")
	}
	for _, k := range []string{"name", "batch_name", "course_name"} {
		if _, ok := p.Vars[k]; !ok {
			t.Errorf("payload is missing var %q the template needs", k)
		}
	}
}

func TestStudentEnrolmentEventsEmptyRoster(t *testing.T) {
	evts, err := studentEnrolmentEvents(&model.Batch{}, "", nil, time.Now())
	if err != nil {
		t.Fatalf("studentEnrolmentEvents: %v", err)
	}
	if len(evts) != 0 {
		t.Errorf("got %d events for an empty roster, want 0", len(evts))
	}
}

func TestStudentEnrolmentEventsHaveDistinctIDs(t *testing.T) {
	students := []model.StudentSummary{
		{Email: "a@example.com"}, {Email: "b@example.com"}, {Email: "c@example.com"},
	}

	evts, err := studentEnrolmentEvents(&model.Batch{CustomerID: 1}, "", students, time.Now())
	if err != nil {
		t.Fatalf("studentEnrolmentEvents: %v", err)
	}

	seen := map[string]bool{}
	for _, e := range evts {
		id := e.ID.String()
		if seen[id] {
			t.Fatalf("duplicate idempotency key %s; the worker would drop all but one", id)
		}
		seen[id] = true
	}
}
