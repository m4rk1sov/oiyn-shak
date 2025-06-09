package validator

import (
	"unicode/utf8"
)

func ValidateUserEmail(v *Validator, email string) {
	v.Check(email != "", "email", "email is required")
	v.Check(utf8.RuneCountInString(email) <= 255, "email", "email must be less than 255 bytes")
	v.Check(Matches(email, EmailRX), "email", "emails must be valid")
}

func ValidateUserPassword(v *Validator, password string) {
	v.Check(password != "", "password", "password is required")
	v.Check(utf8.RuneCountInString(password) >= 8, "password", "password must be at least 8 characters long")
	v.Check(utf8.RuneCountInString(password) <= 72, "password", "password must be less than 72 characters long")
}
