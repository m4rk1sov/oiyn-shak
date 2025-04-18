package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"time"

	//_ "database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Storage struct {
	//db *sql.DB
	db *pgxpool.Pool
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
	dbConfig.MinIdleConns = defaultMinIdleConns
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

func New(dsn string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := pgxpool.NewWithConfig(context.Background(), Config(dsn))
	//db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	err = db.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() {
	s.db.Close()
}

func (s *Storage) SaveUser(ctx context.Context, email string, passwordHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	query := `INSERT INTO users(email, password_hash) VALUES ($1, $2) RETURNING id`

	var id pgtype.Int8
	err := s.db.QueryRow(ctx, query, email, passwordHash).Scan(&id)

	if err != nil {
		var postgresErr *pgconn.PgError
		if errors.As(err, &postgresErr) && postgresErr.Code == "23505" {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return id.Int64, nil
}

func (s *Storage) User(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.User"

	query := `SELECT id, email, password_hash FROM users WHERE email = $1`

	var user models.User
	err := s.db.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) App(ctx context.Context, id int) (models.App, error) {
	const op = "storage.postgres.App"

	query := `SELECT id, name, secret FROM apps WHERE id = $1`

	var app models.App
	err := s.db.QueryRow(ctx, query, id).Scan(&app.ID, &app.Email, &app.Secret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}
