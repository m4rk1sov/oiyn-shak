package models

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	PasswordHash []byte `json:"-"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	Activated    bool   `json:"activated"`
}
