package events

import (
	"encoding/json"
	"testing"
	"time"
)

func mustEmail(t *testing.T, typ string, p EmailRequested) Event {
	t.Helper()
	evt, err := NewEmail(typ, 7, time.Unix(0, 0).UTC(), p)
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	return evt
}

func decode(t *testing.T, evt Event) EmailRequested {
	t.Helper()
	var p EmailRequested
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return p
}

func TestNewEmailStampsIdentityAndVersion(t *testing.T) {
	evt := mustEmail(t, TypeEmailRequested, EmailRequested{To: "a@example.com"})

	if evt.ID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("want a generated idempotency key, got the nil UUID")
	}
	if evt.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", evt.Version, CurrentVersion)
	}
	if evt.CustomerID != 7 {
		t.Errorf("CustomerID = %d, want 7", evt.CustomerID)
	}

	other := mustEmail(t, TypeEmailRequested, EmailRequested{To: "a@example.com"})
	if evt.ID == other.ID {
		t.Error("two events share an idempotency key; dedupe would drop the second")
	}
}

func TestRedactBlanksEveryVar(t *testing.T) {
	evt := mustEmail(t, TypeOTPEmailRequested, EmailRequested{
		To:           "user@example.com",
		TemplateName: "password_reset",
		Vars: map[string]string{
			"otp":            "482915",
			"name":           "Ada",
			"expiry_minutes": "5",
		},
	})

	got := decode(t, Redact(evt))

	if got.To != "user@example.com" {
		t.Errorf("To = %q, want it preserved so the row stays triageable", got.To)
	}
	if got.TemplateName != "password_reset" {
		t.Errorf("TemplateName = %q, want it preserved", got.TemplateName)
	}
	for k, v := range got.Vars {
		if v != redactedPlaceholder {
			t.Errorf("Vars[%q] = %q, want %q", k, v, redactedPlaceholder)
		}
	}
}

func TestRedactDoesNotMutateInput(t *testing.T) {
	evt := mustEmail(t, TypeOTPEmailRequested, EmailRequested{
		Vars: map[string]string{"otp": "482915"},
	})

	_ = Redact(evt)

	if got := decode(t, evt).Vars["otp"]; got != "482915" {
		t.Errorf("input event was mutated: otp = %q, want %q", got, "482915")
	}
}

func TestRedactDropsUnparseablePayload(t *testing.T) {
	evt := Event{Type: TypeOTPEmailRequested, Payload: json.RawMessage(`not json`)}

	got := string(Redact(evt).Payload)

	if got == "not json" {
		t.Fatal("unparseable payload passed through unredacted")
	}
	if got == "" {
		t.Fatal("want a marker payload, got empty")
	}
}

func TestRedactableOnlyCoversOneTimeCodes(t *testing.T) {
	if !Redactable(TypeOTPEmailRequested) {
		t.Error("one-time-code events must be redacted before hitting dead_events")
	}
	if Redactable(TypeEmailRequested) {
		t.Error("ordinary emails must stay intact, or an invite could never be replayed")
	}
}

func TestForDLQKeepsInvitePayloadIntact(t *testing.T) {
	evt := mustEmail(t, TypeEmailRequested, EmailRequested{
		To:           "tutor@example.com",
		TemplateName: "member_invite",
		Vars:         map[string]string{"temp_password": "hunter2"},
	})

	if got := decode(t, ForDLQ(evt)).Vars["temp_password"]; got != "hunter2" {
		t.Errorf("temp_password = %q, want it kept: replay cannot regenerate it", got)
	}
}

func TestForDLQRedactsOneTimeCode(t *testing.T) {
	evt := mustEmail(t, TypeOTPEmailRequested, EmailRequested{
		Vars: map[string]string{"otp": "482915"},
	})

	if got := decode(t, ForDLQ(evt)).Vars["otp"]; got == "482915" {
		t.Error("one-time code survived into the dead-letter payload")
	}
}
