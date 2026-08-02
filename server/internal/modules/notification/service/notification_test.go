package notification

import (
	"reflect"
	"strings"
	"testing"
)

func TestFillEscapesBodyValues(t *testing.T) {
	got := fill(`<p>Hi {{name}}</p>`, map[string]string{"name": `<b>Bobby</b>`}, true)

	if strings.Contains(got, "<b>") {
		t.Errorf("body = %q, want the value escaped", got)
	}
	if !strings.Contains(got, "&lt;b&gt;") {
		t.Errorf("body = %q, want escaped markup", got)
	}
}

func TestFillLeavesSubjectValuesUnescaped(t *testing.T) {
	got := fill(`{{org}} & you`, map[string]string{"org": "Ben & Co"}, false)

	if got != "Ben & Co & you" {
		t.Errorf("subject = %q, want %q", got, "Ben & Co & you")
	}
}

func TestFillIsSinglePass(t *testing.T) {
	vars := map[string]string{
		"name":        "{{secret}}",
		"secret":      "hunter2",
		"sign_in_url": "https://example.test",
	}
	for i := 0; i < 50; i++ {
		got := fill(`Hi {{name}} at {{sign_in_url}}`, vars, false)
		if strings.Contains(got, "hunter2") {
			t.Fatalf("iteration %d: a value was re-substituted: %q", i, got)
		}
	}
}

func TestFillLeavesUnknownPlaceholdersIntact(t *testing.T) {
	got := fill(`{{known}} {{unknown}}`, map[string]string{"known": "x"}, false)

	if got != "x {{unknown}}" {
		t.Errorf("got %q, want %q", got, "x {{unknown}}")
	}
}

func TestOrphansReportsUnsuppliedPlaceholders(t *testing.T) {
	tmpl := `Hi {{name}}, {{activation_url}} expires in {{expires_in}}. {{name}} again.`

	got := orphans(tmpl, map[string]string{"name": "Ada"})

	want := []string{"activation_url", "expires_in"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("orphans = %v, want %v", got, want)
	}
}

func TestOrphansEmptyWhenFullySupplied(t *testing.T) {
	if got := orphans(`Hi {{name}}`, map[string]string{"name": "Ada"}); len(got) != 0 {
		t.Errorf("orphans = %v, want none", got)
	}
}
