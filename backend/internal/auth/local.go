package auth

import (
	"os"

	"golang.org/x/crypto/bcrypt"
)

func AuthenticateLocal(username, password string) bool {
	localUser := os.Getenv("LOCAL_ADMIN_USER")
	localHash := os.Getenv("LOCAL_ADMIN_PASSWORD_HASH")

	if localUser == "" || localHash == "" {
		return false
	}

	if username != localUser {
		return false
	}

	err := bcrypt.CompareHashAndPassword(
		[]byte(localHash),
		[]byte(password),
	)

	return err == nil
}
