package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"profile/internal/domain"
	"profile/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db  *pgxpool.Pool
	log *slog.Logger
}

func Config(dsn string) *pgxpool.Config {
	const defaultMaxConns = int32(50)
	const defaultMinConns = int32(0)
	const defaultMinIdleConns = int32(10)
	const defaultMaxConnLifetime = time.Hour
	const defaultMaxConnIdleTime = time.Minute * 30
	const defaultHealthCheckPeriod = time.Minute
	const defaultConnectTimeout = time.Second * 5

	dbConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}

	dbConfig.MaxConns = defaultMaxConns
	dbConfig.MinConns = defaultMinConns
	dbConfig.MaxConnLifetime = defaultMaxConnLifetime
	dbConfig.MaxConnIdleTime = defaultMaxConnIdleTime
	dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
	dbConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout

	dbConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		slog.Info("Before acquiring the connection pool to the database")
		return true
	}

	dbConfig.AfterRelease = func(conn *pgx.Conn) bool {
		slog.Info("After releasing the connection pool to the database")
		return true
	}

	dbConfig.BeforeClose = func(conn *pgx.Conn) {
		slog.Info("Closed the connection pool to the database")
	}

	return dbConfig
}

func New(dsn string, log *slog.Logger) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := pgxpool.NewWithConfig(context.Background(), Config(dsn))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db, log: log}, nil
}

func (s *Storage) Close() error {
	s.db.Close()
	return nil
}

func (s *Storage) CreateProfile(ctx context.Context, req domain.CreateProfileRequest) (*domain.Profile, error) {
	const op = "storage.postgres.CreateProfile"

	// Generate UUID for profile
	profileID := uuid.New().String()

	query := `
		INSERT INTO profiles (id, user_id, name, phone, address, email)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, name, phone, address, email
	`

	row := s.db.QueryRow(ctx, query, profileID, req.UserID, req.Name, req.Phone, req.Address, req.Email)

	var profile domain.Profile
	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Name,
		&profile.Phone,
		&profile.Address,
		&profile.Email,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			if strings.Contains(pgErr.Detail, "user_id") {
				return nil, storage.ErrProfileAlreadyExists
			}
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("Profile created", slog.String("profile_id", profile.ID))
	return &profile, nil
}

func (s *Storage) GetProfile(ctx context.Context, id string) (*domain.Profile, error) {
	const op = "storage.postgres.GetProfile"

	query := `
		SELECT id, user_id, name, phone, address, email
		FROM profiles
		WHERE id = $1
	`

	row := s.db.QueryRow(ctx, query, id)

	var profile domain.Profile
	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Name,
		&profile.Phone,
		&profile.Address,
		&profile.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrProfileNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &profile, nil
}

func (s *Storage) GetProfileByUserID(ctx context.Context, userID int64) (*domain.Profile, error) {
	const op = "storage.postgres.GetProfileByUserID"

	query := `
		SELECT id, user_id, name, phone, address, email
		FROM profiles
		WHERE user_id = $1
	`

	row := s.db.QueryRow(ctx, query, userID)

	var profile domain.Profile
	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Name,
		&profile.Phone,
		&profile.Address,
		&profile.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrProfileNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &profile, nil
}

func (s *Storage) UpdateProfile(ctx context.Context, req domain.UpdateProfileRequest) (*domain.Profile, error) {
	const op = "storage.postgres.UpdateProfile"

	// Build dynamic query
	var setParts []string
	var args []interface{}
	argIndex := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Phone != nil {
		setParts = append(setParts, fmt.Sprintf("phone = $%d", argIndex))
		args = append(args, *req.Phone)
		argIndex++
	}
	if req.Address != nil {
		setParts = append(setParts, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, *req.Address)
		argIndex++
	}

	if len(setParts) == 0 {
		return s.GetProfile(ctx, req.ID)
	}

	query := fmt.Sprintf(`
		UPDATE profiles
		SET %s
		WHERE id = $%d
		RETURNING id, user_id, name, phone, address, email
	`, strings.Join(setParts, ", "), argIndex)

	args = append(args, req.ID)

	row := s.db.QueryRow(ctx, query, args...)

	var profile domain.Profile
	err := row.Scan(
		&profile.ID,
		&profile.UserID,
		&profile.Name,
		&profile.Phone,
		&profile.Address,
		&profile.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrProfileNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	s.log.Info("Profile updated", slog.String("profile_id", profile.ID))
	return &profile, nil
}

func (s *Storage) DeleteProfile(ctx context.Context, id string) error {
	const op = "storage.postgres.DeleteProfile"

	query := `DELETE FROM profiles WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if result.RowsAffected() == 0 {
		return storage.ErrProfileNotFound
	}

	s.log.Info("Profile deleted", slog.String("profile_id", id))
	return nil
}

func (s *Storage) ListProfiles(ctx context.Context, filter domain.ListProfilesFilter) ([]*domain.Profile, int64, error) {
	const op = "storage.postgres.ListProfiles"

	// Validate sort parameters (без created_at и updated_at)
	validSortFields := map[string]bool{
		"id":      true,
		"user_id": true,
		"name":    true,
		"email":   true,
	}

	sortBy := "id" // используем id как default вместо created_at
	if filter.SortBy != "" && validSortFields[filter.SortBy] {
		sortBy = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder == "ASC" {
		sortOrder = "ASC"
	}

	// Set default pagination
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	offset := (filter.Page - 1) * filter.Limit

	// Count total
	countQuery := `SELECT COUNT(*) FROM profiles`
	var total int64
	err := s.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: count profiles: %w", op, err)
	}

	// Get profiles
	query := fmt.Sprintf(`
		SELECT id, user_id, name, phone, address, email
		FROM profiles
		ORDER BY %s %s
		LIMIT $1 OFFSET $2
	`, sortBy, sortOrder)

	rows, err := s.db.Query(ctx, query, filter.Limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%s: query profiles: %w", op, err)
	}
	defer rows.Close()

	var profiles []*domain.Profile
	for rows.Next() {
		var profile domain.Profile
		err := rows.Scan(
			&profile.ID,
			&profile.UserID,
			&profile.Name,
			&profile.Phone,
			&profile.Address,
			&profile.Email,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%s: scan profile: %w", op, err)
		}
		profiles = append(profiles, &profile)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return profiles, total, nil
}
