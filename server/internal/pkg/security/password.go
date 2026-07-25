package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

func prehash(password, salt, pepper string) []byte {
	mac := hmac.New(sha256.New, []byte(pepper))
	mac.Write([]byte(salt))
	mac.Write([]byte{0})
	mac.Write([]byte(password))
	sum := mac.Sum(nil)

	out := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(out, sum)
	return out
}

func HashPassword(password, salt, pepper string) (string, error) {
	b, err := bcrypt.GenerateFromPassword(prehash(password, salt, pepper), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password, salt, pepper string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), prehash(password, salt, pepper)) == nil
}
