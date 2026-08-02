package auth

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	model "tutorpilot/internal/modules/auth/model"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"lowercases", "Ada@Example.COM", "ada@example.com"},
		{"trims space", "  ada@example.com  ", "ada@example.com"},
		{"trims and lowercases", "  ADA@Example.com\t", "ada@example.com"},
		{"leaves clean address", "ada@example.com", "ada@example.com"},
		{"empty stays empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEmail(tt.in); got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeEmailIsIdempotent(t *testing.T) {
	once := NormalizeEmail("  Ada@Example.COM ")
	if twice := NormalizeEmail(once); twice != once {
		t.Errorf("second pass changed %q to %q", once, twice)
	}
}

func TestMapConstraintErrTranslatesUniqueViolations(t *testing.T) {
	tests := []struct {
		name, constraint string
		want             error
	}{
		{"duplicate org name", "customers_org_name_key", model.ErrOrgTaken},
		{"duplicate email", "dashboard_users_email_key", model.ErrEmailTaken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &pgconn.PgError{Code: "23505", ConstraintName: tt.constraint}
			if got := mapConstraintErr(in); !errors.Is(got, tt.want) {
				t.Errorf("mapConstraintErr(%s) = %v, want %v", tt.constraint, got, tt.want)
			}
		})
	}
}

func TestMapConstraintErrPassesThroughUnknown(t *testing.T) {
	other := errors.New("connection refused")
	if got := mapConstraintErr(other); !errors.Is(got, other) {
		t.Errorf("mapConstraintErr = %v, want the original error", got)
	}
	unknown := &pgconn.PgError{Code: "23505", ConstraintName: "some_other_key"}
	if got := mapConstraintErr(unknown); errors.Is(got, model.ErrOrgTaken) || errors.Is(got, model.ErrEmailTaken) {
		t.Errorf("mapConstraintErr = %v, want it left untranslated", got)
	}
}

func TestRedisKeysNeverCollideAcrossConcerns(t *testing.T) {
	keys := map[string]string{
		"refresh":  refreshKey("abc"),
		"emailOTP": emailOTPKey("ada@example.com"),
		"verified": verifiedKey("ada@example.com"),
		"reset":    resetKey("ada@example.com"),
		"secret":   secretKey(7),
		"priv":     privKey(7),
	}
	seen := map[string]string{}
	for name, k := range keys {
		if prev, dup := seen[k]; dup {
			t.Errorf("%s and %s produce the same key %q", prev, name, k)
		}
		seen[k] = name
	}
}

func TestSignupAndResetOTPKeysDiffer(t *testing.T) {
	const email = "ada@example.com"
	if emailOTPKey(email) == resetKey(email) {
		t.Error("signup and reset OTPs share a key; one flow would consume the other's code")
	}
}

func TestPerSubjectKeysDifferBySubject(t *testing.T) {
	if secretKey(1) == secretKey(2) {
		t.Error("two tenants share a JWT secret cache key")
	}
	if privKey(1) == privKey(2) {
		t.Error("two users share a privilege cache key")
	}
}
