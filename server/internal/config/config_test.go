package config

import (
	"strings"
	"testing"
	"time"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PASSWORD_PEPPER", "pepper")
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PASSWORD_PEPPER", "pepper")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("err = %v, want a DATABASE_URL requirement", err)
	}
}

func TestLoadRequiresPasswordPepper(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PASSWORD_PEPPER", "")

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "PASSWORD_PEPPER") {
		t.Errorf("err = %v, want a PASSWORD_PEPPER requirement", err)
	}
}

func TestLoadRejectsSMTPTimeoutBeyondHalfTheReclaimWindow(t *testing.T) {
	setRequired(t)
	t.Setenv("WORKER_CLAIM_MIN_IDLE", "1m")
	t.Setenv("SMTP_TIMEOUT", "40s")

	_, err := Load()

	if err == nil {
		t.Fatal("a send that outlives the reclaim window lets two workers deliver the same email; Load must refuse it")
	}
	if !strings.Contains(err.Error(), "SMTP_TIMEOUT") {
		t.Errorf("err = %v, want it to name SMTP_TIMEOUT", err)
	}
}

func TestLoadAcceptsSMTPTimeoutInsideTheWindow(t *testing.T) {
	setRequired(t)
	t.Setenv("WORKER_CLAIM_MIN_IDLE", "5m")
	t.Setenv("SMTP_TIMEOUT", "10s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SMTPTimeout != 10*time.Second {
		t.Errorf("SMTPTimeout = %s, want 10s", cfg.SMTPTimeout)
	}
}

func TestLoadRejectsTooShortRetention(t *testing.T) {
	setRequired(t)
	t.Setenv("EVENT_RETENTION_AUTH", "10s")

	_, err := Load()

	if err == nil || !strings.Contains(err.Error(), "retention") {
		t.Errorf("err = %v, want a retention floor: trimming ignores pending entries, so a short window silently drops events", err)
	}
}

func TestLoadRejectsMalformedDuration(t *testing.T) {
	setRequired(t)
	t.Setenv("OTP_TTL", "not-a-duration")

	if _, err := Load(); err == nil {
		t.Error("want an error for an unparseable duration")
	}
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.EventStreamNotifications == cfg.EventStreamAuth {
		t.Error("auth and notification streams must differ, or bulk mail can delay a login code")
	}
	if !cfg.RelayEnabled {
		t.Error("RelayEnabled should default true; otherwise nothing publishes the outbox")
	}
	if cfg.ImportMaxRows <= 0 {
		t.Error("ImportMaxRows must be positive; the import loop holds locks per row")
	}
}

func TestGetIntRejectsNonPositive(t *testing.T) {
	t.Setenv("SOME_INT", "0")
	if got := getInt("SOME_INT", 7); got != 7 {
		t.Errorf("getInt = %d, want the fallback 7 for a non-positive value", got)
	}
	t.Setenv("SOME_INT", "-2")
	if got := getInt("SOME_INT", 7); got != 7 {
		t.Errorf("getInt = %d, want the fallback 7 for a negative value", got)
	}
	t.Setenv("SOME_INT", "12")
	if got := getInt("SOME_INT", 7); got != 12 {
		t.Errorf("getInt = %d, want 12", got)
	}
}

func TestGetBoolAcceptsCommonTruthyForms(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("SOME_BOOL", v)
		if !getBool("SOME_BOOL", false) {
			t.Errorf("getBool(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"0", "false", "no", "off", "nonsense"} {
		t.Setenv("SOME_BOOL", v)
		if getBool("SOME_BOOL", true) {
			t.Errorf("getBool(%q) = true, want false", v)
		}
	}
}

func TestSplitCSVTrimsAndDropsEmpties(t *testing.T) {
	got := splitCSV(" a , ,b,, c ")

	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("splitCSV = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitCSV[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
