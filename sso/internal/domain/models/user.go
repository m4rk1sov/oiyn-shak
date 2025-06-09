package models

type User struct {
	ID           int64
	Email        string
	PasswordHash []byte
	Name         string
	Phone        string
	Address      string
	Activated    bool
}
