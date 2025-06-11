package profile

import (
	"context"
	"errors"
	"log/slog"

	"profile/internal/domain"
	"profile/internal/services/profile"
	"profile/internal/storage"

	profilev1 "github.com/m4rk1sov/protos/gen/go/profile"
	rpc "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type serverAPI struct {
	profilev1.UnimplementedProfileServiceServer
	profileService *profile.Service
	log            *slog.Logger
}

func Register(gRPCServer *grpc.Server, profileService *profile.Service, log *slog.Logger) {
	profilev1.RegisterProfileServiceServer(gRPCServer, &serverAPI{
		profileService: profileService,
		log:            log,
	})

	reflection.Register(gRPCServer)
	log.Info("Profile service registered")
}

type Profile interface {
	CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (*domain.Profile, error)
	GetProfile(ctx context.Context, id string) (*domain.Profile, error)
	GetProfileByUserID(ctx context.Context, userID int64) (*domain.Profile, error)
	UpdateProfile(ctx context.Context, req domain.UpdateProfileRequest) (*domain.Profile, error)
	DeleteProfile(ctx context.Context, id string) error
	ListProfiles(ctx context.Context, filter domain.ListProfilesFilter) ([]*domain.Profile, int64, error)
}

func (s *serverAPI) CreateProfile(ctx context.Context, req *profilev1.CreateProfileRequest) (*profilev1.ProfileResponse, error) {
	if req.GetUserId() <= 0 {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "user_id is required",
				},
			},
		}, nil
	}

	if req.GetName() == "" {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "name is required",
				},
			},
		}, nil
	}

	if req.GetEmail() == "" {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "email is required",
				},
			},
		}, nil
	}

	domainReq := domain.CreateProfileRequest{
		UserID:  req.GetUserId(),
		Name:    req.GetName(),
		Phone:   req.GetPhone(),
		Address: req.GetAddress(),
		Email:   req.GetEmail(),
	}

	profileData, err := s.profileService.CreateProfile(ctx, domainReq)
	if err != nil {
		if errors.Is(err, storage.ErrProfileAlreadyExists) {
			return &profilev1.ProfileResponse{
				Result: &profilev1.ProfileResponse_Error{
					Error: &rpc.Status{
						Code:    int32(codes.AlreadyExists),
						Message: "profile already exists",
					},
				},
			}, nil
		}

		s.log.Error("Failed to create profile", slog.String("error", err.Error()))
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.Internal),
					Message: "internal server error",
				},
			},
		}, nil
	}

	return &profilev1.ProfileResponse{
		Result: &profilev1.ProfileResponse_Profile{
			Profile: &profilev1.UserProfile{
				Id:      profileData.ID,
				UserId:  profileData.UserID,
				Name:    profileData.Name,
				Phone:   profileData.Phone,
				Address: profileData.Address,
				Email:   profileData.Email,
			},
		},
	}, nil
}

func (s *serverAPI) GetProfile(ctx context.Context, req *profilev1.GetProfileRequest) (*profilev1.ProfileResponse, error) {
	if req.GetId() == "" {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "id is required",
				},
			},
		}, nil
	}

	profileData, err := s.profileService.GetProfile(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return &profilev1.ProfileResponse{
				Result: &profilev1.ProfileResponse_Error{
					Error: &rpc.Status{
						Code:    int32(codes.NotFound),
						Message: "profile not found",
					},
				},
			}, nil
		}

		s.log.Error("Failed to get profile", slog.String("error", err.Error()))
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.Internal),
					Message: "internal server error",
				},
			},
		}, nil
	}

	return &profilev1.ProfileResponse{
		Result: &profilev1.ProfileResponse_Profile{
			Profile: &profilev1.UserProfile{
				Id:      profileData.ID,
				UserId:  profileData.UserID,
				Name:    profileData.Name,
				Phone:   profileData.Phone,
				Address: profileData.Address,
				Email:   profileData.Email,
			},
		},
	}, nil
}

func (s *serverAPI) GetProfileByUserID(ctx context.Context, req *profilev1.GetProfileByUserIDRequest) (*profilev1.ProfileResponse, error) {
	if req.GetUserId() <= 0 {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "user_id is required",
				},
			},
		}, nil
	}

	profileData, err := s.profileService.GetProfileByUserID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return &profilev1.ProfileResponse{
				Result: &profilev1.ProfileResponse_Error{
					Error: &rpc.Status{
						Code:    int32(codes.NotFound),
						Message: "profile not found",
					},
				},
			}, nil
		}

		s.log.Error("Failed to get profile by user ID", slog.String("error", err.Error()))
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.Internal),
					Message: "internal server error",
				},
			},
		}, nil
	}

	return &profilev1.ProfileResponse{
		Result: &profilev1.ProfileResponse_Profile{
			Profile: &profilev1.UserProfile{
				Id:      profileData.ID,
				UserId:  profileData.UserID,
				Name:    profileData.Name,
				Phone:   profileData.Phone,
				Address: profileData.Address,
				Email:   profileData.Email,
			},
		},
	}, nil
}

func (s *serverAPI) UpdateProfile(ctx context.Context, req *profilev1.UpdateProfileRequest) (*profilev1.ProfileResponse, error) {
	if req.GetId() == "" {
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.InvalidArgument),
					Message: "id is required",
				},
			},
		}, nil
	}

	domainReq := domain.UpdateProfileRequest{
		ID: req.GetId(),
	}

	if req.Name != nil {
		domainReq.Name = req.Name
	}
	if req.Phone != nil {
		domainReq.Phone = req.Phone
	}
	if req.Address != nil {
		domainReq.Address = req.Address
	}

	profileData, err := s.profileService.UpdateProfile(ctx, domainReq)
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return &profilev1.ProfileResponse{
				Result: &profilev1.ProfileResponse_Error{
					Error: &rpc.Status{
						Code:    int32(codes.NotFound),
						Message: "profile not found",
					},
				},
			}, nil
		}

		s.log.Error("Failed to update profile", slog.String("error", err.Error()))
		return &profilev1.ProfileResponse{
			Result: &profilev1.ProfileResponse_Error{
				Error: &rpc.Status{
					Code:    int32(codes.Internal),
					Message: "internal server error",
				},
			},
		}, nil
	}

	return &profilev1.ProfileResponse{
		Result: &profilev1.ProfileResponse_Profile{
			Profile: &profilev1.UserProfile{
				Id:      profileData.ID,
				UserId:  profileData.UserID,
				Name:    profileData.Name,
				Phone:   profileData.Phone,
				Address: profileData.Address,
				Email:   profileData.Email,
			},
		},
	}, nil
}

func (s *serverAPI) DeleteProfile(ctx context.Context, req *profilev1.DeleteProfileRequest) (*profilev1.DeleteProfileResponse, error) {
	if req.GetId() == "" {
		return &profilev1.DeleteProfileResponse{
			Success: false,
			Message: "id is required",
		}, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.profileService.DeleteProfile(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return &profilev1.DeleteProfileResponse{
				Success: false,
				Message: "profile not found",
			}, status.Error(codes.NotFound, "profile not found")
		}

		s.log.Error("Failed to delete profile", slog.String("error", err.Error()))
		return &profilev1.DeleteProfileResponse{
			Success: false,
			Message: "internal server error",
		}, status.Error(codes.Internal, "internal server error")
	}

	return &profilev1.DeleteProfileResponse{
		Success: true,
		Message: "profile deleted successfully",
	}, nil
}

func (s *serverAPI) ListProfiles(ctx context.Context, req *profilev1.ListProfilesRequest) (*profilev1.ListProfilesResponse, error) {
	filter := domain.ListProfilesFilter{
		Page:      req.GetPage(),
		Limit:     req.GetLimit(),
		SortBy:    req.GetSortBy(),
		SortOrder: req.GetSortOrder(),
	}

	profiles, total, err := s.profileService.ListProfiles(ctx, filter)
	if err != nil {
		s.log.Error("Failed to list profiles", slog.String("error", err.Error()))
		return nil, status.Error(codes.Internal, "internal server error")
	}

	var protoProfiles []*profilev1.UserProfile
	for _, p := range profiles {
		protoProfiles = append(protoProfiles, &profilev1.UserProfile{
			Id:      p.ID,
			UserId:  p.UserID,
			Name:    p.Name,
			Phone:   p.Phone,
			Address: p.Address,
			Email:   p.Email,
		})
	}

	hasNext := filter.Page*filter.Limit < total

	return &profilev1.ListProfilesResponse{
		Profiles: protoProfiles,
		Total:    total,
		Page:     filter.Page,
		Limit:    filter.Limit,
		HasNext:  hasNext,
	}, nil
}
