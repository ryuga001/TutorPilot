package security

import (
	"strings"
	"testing"
)

const pepper = "test-pepper"

func TestHashPasswordVerifies(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	hash, err := HashPassword("correct horse", salt, pepper)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !CheckPassword(hash, "correct horse", salt, pepper) {
		t.Error("the correct password did not verify")
	}
	if CheckPassword(hash, "wrong", salt, pepper) {
		t.Error("a wrong password verified")
	}
}

func TestHashIsSaltDependent(t *testing.T) {
	s1, _ := NewSalt()
	s2, _ := NewSalt()
	h1, _ := HashPassword("same", s1, pepper)

	if CheckPassword(h1, "same", s2, pepper) {
		t.Error("a hash verified under a different salt")
	}
}

func TestHashIsPepperDependent(t *testing.T) {
	salt, _ := NewSalt()
	hash, _ := HashPassword("same", salt, pepper)

	if CheckPassword(hash, "same", salt, "other-pepper") {
		t.Error("a hash verified under a different pepper; the pepper adds nothing")
	}
}

func TestSameInputProducesDifferentHashes(t *testing.T) {
	salt, _ := NewSalt()
	h1, _ := HashPassword("same", salt, pepper)
	h2, _ := HashPassword("same", salt, pepper)

	if h1 == h2 {
		t.Error("bcrypt produced an identical hash twice; its internal salt is not being applied")
	}
	if !CheckPassword(h1, "same", salt, pepper) || !CheckPassword(h2, "same", salt, pepper) {
		t.Error("both hashes must still verify")
	}
}

func TestNewSaltIsUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		s, err := NewSalt()
		if err != nil {
			t.Fatalf("NewSalt: %v", err)
		}
		if s == "" {
			t.Fatal("NewSalt returned empty")
		}
		if seen[s] {
			t.Fatalf("duplicate salt after %d draws", i)
		}
		seen[s] = true
	}
}

func TestGenerateOTPShapeAndSpread(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		otp, err := GenerateOTP(6)
		if err != nil {
			t.Fatalf("GenerateOTP: %v", err)
		}
		if len(otp) != 6 {
			t.Fatalf("otp %q has length %d, want 6", otp, len(otp))
		}
		if strings.Trim(otp, "0123456789") != "" {
			t.Fatalf("otp %q is not all digits", otp)
		}
		seen[otp] = true
	}
	if len(seen) < 100 {
		t.Errorf("only %d distinct OTPs in 200 draws; entropy looks wrong", len(seen))
	}
}

func TestHashTokenIsStableAndNotTheInput(t *testing.T) {
	raw := "a-secret-token"
	h := HashToken(raw)

	if h == raw {
		t.Fatal("HashToken returned the input; the store would hold plaintext")
	}
	if h != HashToken(raw) {
		t.Error("HashToken is not deterministic; lookups would miss")
	}
	if h == HashToken(raw+"x") {
		t.Error("two different tokens hashed identically")
	}
}

func TestGenerateRefreshTokenReturnsMatchingHash(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}
	if raw == "" || hash == "" {
		t.Fatal("empty token or hash")
	}
	if hash != HashToken(raw) {
		t.Error("returned hash does not match HashToken(raw); refresh lookups would fail")
	}
}

func TestGenerateTempPasswordIsNonTrivial(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		p, err := GenerateTempPassword()
		if err != nil {
			t.Fatalf("GenerateTempPassword: %v", err)
		}
		if len(p) < 8 {
			t.Fatalf("temp password %q is only %d chars", p, len(p))
		}
		if seen[p] {
			t.Fatalf("duplicate temp password after %d draws", i)
		}
		seen[p] = true
	}
}
