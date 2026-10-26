package utils

import (
	"crypto/subtle"
	"fmt"
	"unicode"
)

func ValidatePasswordComplexity(password string) error {
	if len(password) < 8 || len(password) > 72 {
		return fmt.Errorf("密码长度需为 8 到 72 位")
	}
	var upper, lower, digit, special bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		default:
			if !unicode.IsSpace(r) {
				special = true
			}
		}
	}
	if !upper || !lower || !digit || !special {
		return fmt.Errorf("密码需同时包含大写字母、小写字母、数字和特殊字符")
	}
	return nil
}

func ConstantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
