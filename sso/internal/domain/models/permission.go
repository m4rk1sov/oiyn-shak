package models

type Permission struct {
	ID   int64
	Code string
}

type UserPermission struct {
	UserID       int64
	PermissionID int64
}
