package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"profile/internal/domain"
	"profile/internal/storage"
)

type Service struct {
	log     *slog.Logger
	storage storage.ProfileStorage
}

func New(log *slog.Logger, storage storage.ProfileStorage) *Service {
	return &Service{
		log:     log,
		storage: storage,
	}
}

func (s *Service) CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (*domain.Profile, error) {
	const op = "services.profile.CreateProfile"

	log := s.log.With(
		slog.String("op", op),
		slog.Int64("user_id", req.UserID),
	)

	// Validate input
	if req.UserID <= 0 {
		log.Warn("Invalid user ID")
		return nil, fmt.Errorf("%s: invalid user ID", op)
	}

	if req.Name == "" {
		log.Warn("Name is required")
		return nil, fmt.Errorf("%s: name is required", op)
	}

	if req.Email == "" {
		log.Warn("Email is required")
		return nil, fmt.Errorf("%s: email is required", op)
	}

	profile, err := s.storage.CreateProfile(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrProfileAlreadyExists) {
			log.Warn("Profile already exists")
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to create profile", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Profile created successfully", slog.String("profile_id", profile.ID))
	return profile, nil
}

func (s *Service) GetProfile(ctx context.Context, id string) (*domain.Profile, error) {
	const op = "services.profile.GetProfile"

	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", id),
	)

	if id == "" {
		log.Warn("Profile ID is required")
		return nil, fmt.Errorf("%s: profile ID is required", op)
	}

	profile, err := s.storage.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to get profile", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return profile, nil
}

func (s *Service) GetProfileByUserID(ctx context.Context, userID int64) (*domain.Profile, error) {
	const op = "services.profile.GetProfileByUserID"

	log := s.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)

	if userID <= 0 {
		log.Warn("Invalid user ID")
		return nil, fmt.Errorf("%s: invalid user ID", op)
	}

	profile, err := s.storage.GetProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to get profile", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, req domain.UpdateProfileRequest) (*domain.Profile, error) {
	const op = "services.profile.UpdateProfile"

	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", req.ID),
	)

	if req.ID == "" {
		log.Warn("Profile ID is required")
		return nil, fmt.Errorf("%s: profile ID is required", op)
	}

	profile, err := s.storage.UpdateProfile(ctx, req)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to update profile", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Profile updated successfully", slog.String("profile_id", profile.ID))
	return profile, nil
}

func (s *Service) DeleteProfile(ctx context.Context, id string) error {
	const op = "services.profile.DeleteProfile"

	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", id),
	)

	if id == "" {
		log.Warn("Profile ID is required")
		return fmt.Errorf("%s: profile ID is required", op)
	}

	err := s.storage.DeleteProfile(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to delete profile", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Profile deleted successfully", slog.String("profile_id", id))
	return nil
}

func (s *Service) ListProfiles(ctx context.Context, filter domain.ListProfilesFilter) ([]*domain.Profile, int64, error) {
	const op = "services.profile.ListProfiles"

	log := s.log.With(
		slog.String("op", op),
		slog.Int64("page", filter.Page),
		slog.Int64("limit", filter.Limit),
	)

	profiles, total, err := s.storage.ListProfiles(ctx, filter)
	if err != nil {
		log.Error("Failed to list profiles", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("Profiles listed successfully", slog.Int("count", len(profiles)), slog.Int64("total", total))
	return profiles, total, nil
}
