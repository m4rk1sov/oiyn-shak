package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"profile/internal/app/auth"
	
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
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
	if req.UserID != userCtx.UserID {
		log.Warn("user trying to create profile for another user",
			slog.Int64("jwt_user_id", userCtx.UserID),
			slog.Int64("requested_user_id", req.UserID))
		return nil, fmt.Errorf("%s: can only create own profile", op)
	}
	
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
	
	if req.Email == "" {
		req.Email = userCtx.Email
	}
	if req.Name == "" {
		req.Name = userCtx.Name
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
	
	log.Info("Profile created successfully",
		slog.String("profile_id", profile.ID),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return profile, nil
}

func (s *Service) GetProfile(ctx context.Context, id string) (*domain.Profile, error) {
	const op = "services.profile.GetProfile"
	
	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", id),
	)
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
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
	
	if profile.UserID != userCtx.UserID && !auth.HasPermission(ctx, "admin") {
		log.Warn("user trying to access another user's profile",
			slog.Int64("jwt_user_id", userCtx.UserID),
			slog.Int64("profile_user_id", profile.UserID))
		return nil, fmt.Errorf("%s: access denied", op)
	}
	
	log.Info("Profile retrieved successfully",
		slog.String("profile_id", profile.ID),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return profile, nil
}

func (s *Service) GetProfileByUserID(ctx context.Context, userID int64) (*domain.Profile, error) {
	const op = "services.profile.GetProfileByUserID"
	
	log := s.log.With(
		slog.String("op", op),
		slog.Int64("user_id", userID),
	)
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
	if userID <= 0 {
		log.Warn("Invalid user ID")
		return nil, fmt.Errorf("%s: invalid user ID", op)
	}
	
	if userID != userCtx.UserID && !auth.HasPermission(ctx, "admin") {
		log.Warn("user trying to access another user's profile",
			slog.Int64("jwt_user_id", userCtx.UserID),
			slog.Int64("requested_user_id", userID))
		return nil, fmt.Errorf("%s: access denied", op)
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
	
	log.Info("Profile retrieved successfully",
		slog.Int64("user_id", profile.UserID),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, req domain.UpdateProfileRequest) (*domain.Profile, error) {
	const op = "services.profile.UpdateProfile"
	
	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", req.ID),
	)
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
	if req.ID == "" {
		log.Warn("Profile ID is required")
		return nil, fmt.Errorf("%s: profile ID is required", op)
	}
	
	existingProfile, err := s.storage.GetProfile(ctx, req.ID)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to get existing profile", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
	if existingProfile.UserID != userCtx.UserID && !auth.HasPermission(ctx, "admin") {
		log.Warn("user trying to update another user's profile",
			slog.Int64("jwt_user_id", userCtx.UserID),
			slog.Int64("profile_user_id", existingProfile.UserID))
		return nil, fmt.Errorf("%s: access denied", op)
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
	
	log.Info("Profile updated successfully",
		slog.String("profile_id", profile.ID),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return profile, nil
}

func (s *Service) DeleteProfile(ctx context.Context, id string) error {
	const op = "services.profile.DeleteProfile"
	
	log := s.log.With(
		slog.String("op", op),
		slog.String("profile_id", id),
	)
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}
	
	if id == "" {
		log.Warn("Profile ID is required")
		return fmt.Errorf("%s: profile ID is required", op)
	}
	
	existingProfile, err := s.storage.GetProfile(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to get existing profile", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}
	
	if existingProfile.UserID != userCtx.UserID && !auth.HasPermission(ctx, "admin") {
		log.Warn("user trying to delete another user's profile",
			slog.Int64("jwt_user_id", userCtx.UserID),
			slog.Int64("profile_user_id", existingProfile.UserID))
		return fmt.Errorf("%s: access denied", op)
	}
	
	err = s.storage.DeleteProfile(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			log.Warn("Profile not found")
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Error("Failed to delete profile", slog.String("error", err.Error()))
		return fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("Profile deleted successfully",
		slog.String("profile_id", id),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return nil
}

func (s *Service) ListProfiles(ctx context.Context, filter domain.ListProfilesFilter) ([]*domain.Profile, int64, error) {
	const op = "services.profile.ListProfiles"
	
	log := s.log.With(
		slog.String("op", op),
		slog.Int64("page", filter.Page),
		slog.Int64("limit", filter.Limit),
	)
	
	auth.DebugUserContext(ctx)
	
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	
	if !auth.HasPermission(ctx, "admin") {
		log.Warn("user trying to list all profiles without admin permission",
			slog.Int64("jwt_user_id", userCtx.UserID))
		return nil, 0, fmt.Errorf("%s: access denied", op)
	}
	
	profiles, total, err := s.storage.ListProfiles(ctx, filter)
	if err != nil {
		log.Error("Failed to list profiles", slog.String("error", err.Error()))
		return nil, 0, fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("Profiles listed successfully",
		slog.Int("count", len(profiles)),
		slog.Int64("total", total),
		slog.Int64("jwt_user_id", userCtx.UserID))
	
	return profiles, total, nil
}

func (s *Service) GetMyProfile(ctx context.Context) (*domain.Profile, error) {
	const op = "services.profile.GetMyProfile"
	
	log := s.log.With(slog.String("op", op))
	
	// Получаем пользователя из JWT контекста
	userCtx, err := auth.GetUserFromContext(ctx)
	if err != nil {
		log.Error("failed to get user from context", slog.String("error", err.Error()))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	
	log.Info("getting current user profile", slog.Int64("user_id", userCtx.UserID))
	
	return s.GetProfileByUserID(ctx, userCtx.UserID)
}
