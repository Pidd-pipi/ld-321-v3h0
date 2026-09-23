package database

import "golang.org/x/crypto/bcrypt"

func hashPwd(pwd string) string {
	b, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(b)
}
