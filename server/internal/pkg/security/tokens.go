package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/big"
)

func GenerateRefreshToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func NewSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateOTP(length int) (string, error) {
	const digits = "0123456789"
	return randomString(digits, length)
}

// GenerateActivationToken returns a single-use invitation token and its hash. Only
// the hash is stored, so the raw token exists solely in the invitation email.
func GenerateActivationToken() (raw string, hash string, err error) {
	return GenerateRefreshToken()
}

// tempPasswordAlphabet omits characters that are easily confused when a member
// retypes a password out of an email: 0/O, 1/l/I, and similar.
const tempPasswordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// GenerateTempPassword returns the initial password for an admin-created member.
// It is stored hashed like any other, and the account is flagged to force a change
// before it can be used for anything but changing it.
func GenerateTempPassword() (string, error) {
	return randomString(tempPasswordAlphabet, 14)
}

func randomString(alphabet string, length int) (string, error) {
	out := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out), nil
}
