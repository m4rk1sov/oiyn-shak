package storage

import (
	"context"
	"errors"
	"profile/internal/domain"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrProfileAlreadyExists = errors.New("profile already exists")
)

type ProfileStorage interface {
	CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (*domain.Profile, error)
	GetProfile(ctx context.Context, id string) (*domain.Profile, error)
	GetProfileByUserID(ctx context.Context, userID int64) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, req domain.UpdateProfileRequest) (*domain.Profile, error)
	DeleteProfile(ctx context.Context, id string) error
	ListProfiles(ctx context.Context, filter domain.ListProfilesFilter) ([]*domain.Profile, int64, error)
	Close() error
}
