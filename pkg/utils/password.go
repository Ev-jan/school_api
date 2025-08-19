package utils

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/argon2"
)

func VerifyPassword(password, encodedHash string) error {
	parts := strings.Split(encodedHash, ".")
	if len(parts) != 2 {
		return ErrorHandler(errors.New("invalid encoded hash format"), "internal error")
	}
	saltBase64 := parts[0]
	hashePasswordBase64 := parts[1]

	hashedSalt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return ErrorHandler(err, "internal error")
	}

	hashedPassword, err := base64.StdEncoding.DecodeString(hashePasswordBase64)
	if err != nil {
		return ErrorHandler(err, "internal error")
	}

	receivedPasswordHash := argon2.IDKey([]byte(password), hashedSalt, 1, 64*1024, 4, 32)
	// compare hash lengths first as a fast primary measure
	if len(receivedPasswordHash) != len(hashedPassword) {
		return ErrorHandler(errors.New("hash length mismatch"), "incorrect pasword")
	}

	if subtle.ConstantTimeCompare(receivedPasswordHash, hashedPassword) == 1 {
		return nil
	}
	return ErrorHandler(errors.New("incorrect pasword"), "incorrect pasword")
}
