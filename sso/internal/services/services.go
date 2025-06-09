package services

import "errors"

var (
	ErrInvalidEmail    = errors.New("invalid email (must be in format 'example@mail.com')")
	ErrInvalidPassword = errors.New("invalid password (minimum 8 characters long)")
)
