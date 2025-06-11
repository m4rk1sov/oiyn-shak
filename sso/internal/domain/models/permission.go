package models

type Permission struct {
	ID   int64  `json:"id"`
	Code string `json:"code"`
}

type UserPermission struct {
	UserID       int64 `json:"user_id"`
	PermissionID int64 `json:"permission_id"`
}
