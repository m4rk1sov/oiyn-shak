package storage

import "errors"

var (
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrAppNotFound          = errors.New("app not found")
	ErrTokenNotFound        = errors.New("token not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrPermissionNotFound   = errors.New("permission not found")
	ErrTokenExists          = errors.New("token already exists")
)
