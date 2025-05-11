package storage

import "errors"

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrAppNotFound        = errors.New("app not found")
	ErrTokenNotFound      = errors.New("token not found")
	ErrPermissionNotFound = errors.New("permission not found")
)
