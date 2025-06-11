package storage

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserExists           = errors.New("user already exists")
	ErrAppNotFound          = errors.New("app not found")
	ErrTokenExists          = errors.New("token already exists")
	ErrTokenNotFound        = errors.New("token not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrPermissionNotFound   = errors.New("permission not found")
)
