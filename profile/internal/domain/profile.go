package domain

type Profile struct {
	ID      string `json:"id" db:"id"`
	UserID  int64  `json:"user_id" db:"user_id"`
	Name    string `json:"name" db:"name"`
	Phone   string `json:"phone" db:"phone"`
	Address string `json:"address" db:"address"`
	Email   string `json:"email" db:"email"`
}

type CreateProfileRequest struct {
	UserID  int64  `json:"user_id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Email   string `json:"email"`
}

type UpdateProfileRequest struct {
	ID      string  `json:"id"`
	Name    *string `json:"name,omitempty"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
}

type ListProfilesFilter struct {
	Page      int64  `json:"page"`
	Limit     int64  `json:"limit"`
	SortBy    string `json:"sort_by"`
	SortOrder string `json:"sort_order"`
}
