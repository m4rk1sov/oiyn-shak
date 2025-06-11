package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgconn"
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

func (s *Storage) SaveUser(ctx context.Context, name, phone, address, email string, passwordHash []byte) (int64, string, string, bool, error) {
	//TODO implement me
	panic("implement me")
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

func (s *Storage) SaveUserWithPermission(ctx context.Context, name string, phone string, address string, email string, passwordHash []byte, permissionID int64) (int64, string, string, bool, error) {
	const op = "storage.postgres.SaveUserWithPermission"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, "", "", false, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			return
		}
	}(tx, ctx)

	userQuery := `
		INSERT INTO users(email, password_hash, name, phone, address)
		VALUES ($1, $2, $3, $4, $5)
        RETURNING id, name, email, activated`

	var id int64
	var resName, resEmail string
	var activated bool

	err = tx.QueryRow(ctx, userQuery, email, passwordHash, name, phone, address).Scan(&id, &resName, &resEmail, &activated)

	if err != nil {
		var postgresErr *pgconn.PgError
		if errors.As(err, &postgresErr) && postgresErr.Code == "23505" {
			return 0, "", "", false, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}
		return 0, "", "", false, fmt.Errorf("%s: %w", op, err)
	}

	permQuery := `
		INSERT INTO users_permissions(user_id, permission_id)
		VALUES ($1, $2) ON CONFLICT DO NOTHING`

	_, err = tx.Exec(ctx, permQuery, id, permissionID)
	if err != nil {
		return 0, "", "", false, fmt.Errorf("%s: failed to add permission: %w", op, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, "", "", false, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return id, resName, resEmail, activated, nil
}

func (s *Storage) UserByEmail(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.UserByEmail"

	query := `SELECT id, email, password_hash, name, phone, address, activated FROM users WHERE email = $1`

	var user models.User
	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.Name, &user.Phone, &user.Address, &user.Activated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) UserByID(ctx context.Context, userID int64) (models.User, error) {
	const op = "storage.postgres.UserByID"

	query := `SELECT id, email, password_hash, name, phone, address, activated FROM users WHERE id = $1`

	var user models.User
	err := s.db.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.PasswordHash,
		&user.Name, &user.Phone, &user.Address, &user.Activated,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) App(ctx context.Context, id int32) (models.App, error) {
	const op = "storage.postgres.App"

	query := `SELECT id, name, secret FROM apps WHERE id = $1`

	var app models.App
	err := s.db.QueryRow(ctx, query, id).Scan(&app.ID, &app.Name, &app.Secret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.App{}, fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}

		return models.App{}, fmt.Errorf("%s: %w", op, err)
	}

	return app, nil
}

func (s *Storage) GetAppSecret(ctx context.Context, appID int32) (string, error) {
	const op = "storage.postgres.GetAppSecret"

	query := `SELECT secret FROM apps WHERE id = $1`

	var secret string
	err := s.db.QueryRow(ctx, query, appID).Scan(&secret)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrAppNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return secret, nil
}

func (s *Storage) SaveRefresh(ctx context.Context, token string, userID int64, appID int32, expiresAt time.Time) error {
	const op = "storage.postgres.SaveRefresh"

	query := `
	INSERT INTO refresh_tokens (token, user_id, app_id, expires_at) 
	VALUES ($1, $2, $3, $4) 
	ON CONFLICT (token) DO NOTHING`

	_, err := s.db.Exec(ctx, query, token, userID, appID, expiresAt)
	if err != nil {
		var postgresErr *pgconn.PgError
		if errors.As(err, &postgresErr) && postgresErr.Code == "23505" {
			return fmt.Errorf("%s: %w", op, storage.ErrTokenExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return err
}

func (s *Storage) DeleteRefresh(ctx context.Context, token string) error {
	const op = "storage.postgres.DeleteRefresh"

	query := `DELETE FROM refresh_tokens WHERE token = $1`

	cmd, err := s.db.Exec(ctx, query, token)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if cmd.RowsAffected() == 0 {
		return storage.ErrRefreshTokenNotFound
	}

	return nil
}

func (s *Storage) ExistsRefresh(ctx context.Context, token string) (bool, error) {
	const op = "storage.postgres.ExistsRefresh"

	query := `SELECT 1 FROM refresh_tokens WHERE token = $1 AND expires_at > now() LIMIT 1`

	row := s.db.QueryRow(ctx, query, token)

	var dummy int
	err := row.Scan(&dummy)
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}
	return true, nil
}

func (s *Storage) GetUserPermissions(ctx context.Context, id int64) ([]string, error) {
	const op = "storage.postgres.GetUserPermissions"

	query := `
	SELECT permissions.code FROM permissions 
    INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id 
    INNER JOIN users ON users_permissions.user_id = users.id 
    WHERE users.id = $1`

	rows, err := s.db.Query(ctx, query, id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string

	for rows.Next() {
		var permission string

		err := rows.Scan(&permission)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("%s: %w", op, storage.ErrPermissionNotFound)
			}

			return nil, fmt.Errorf("%s: %w", op, err)
		}

		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissions, nil
}

func (s *Storage) UserPermissions(ctx context.Context, userID int64) ([]models.Permission, error) {
	const op = "storage.postgres.UserPermissions"

	query := `
	SELECT permissions.id, permissions.code FROM permissions
    INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
    WHERE users_permissions.user_id = $1`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var permissions []models.Permission

	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(&permission.ID, &permission.Code)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return permissions, nil
}

func (s *Storage) HasUserPermission(ctx context.Context, id int64, permission string) (bool, error) {
	const op = "storage.postgres.HasUserPermission"

	query := `
	SELECT EXISTS ( 
	SELECT 1 FROM users_permissions 
    JOIN permissions ON users_permissions.permission_id = permissions.id
	WHERE users_permissions.user_id = $1 AND permissions.code = $2 
	)`

	var allowed bool

	err := s.db.QueryRow(ctx, query, id, permission).Scan(&allowed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("%s: %w", op, storage.ErrPermissionNotFound)
		}

		return false, fmt.Errorf("%s: %w", op, err)
	}

	return allowed, nil
}

func (s *Storage) AddUserPermission(ctx context.Context, userID int64, permissionID int64) error {
	const op = "storage.postgres.AddUserPermission"

	query := `
		INSERT INTO users_permissions(user_id, permission_id) 
		VALUES ($1, $2) ON CONFLICT DO NOTHING `

	_, err := s.db.Exec(ctx, query, userID, permissionID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return err
}

func (s *Storage) GetUserPermissionsAsModels(ctx context.Context, userID int64) ([]models.Permission, error) {
	return s.UserPermissions(ctx, userID)
}

func (s *Storage) SaveVerificationToken(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	const op = "storage.postgres.SaveVerificationToken"

	query := `
		INSERT INTO email_verification_tokens (token, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at,
			created_at = now()`

	_, err := s.db.Exec(ctx, query, token, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) VerifyEmail(ctx context.Context, token string) (int64, error) {
	const op = "storage.postgres.VerifyEmail"

	// Начинаем транзакцию
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			return
		}
	}(tx, ctx)

	// 1. Получаем пользователя по токену и проверяем срок действия
	var userID int64
	checkQuery := `
		SELECT user_id FROM email_verification_tokens
		WHERE token = $1 AND expires_at > now()`

	err = tx.QueryRow(ctx, checkQuery, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrTokenNotFound)
		}
		return 0, fmt.Errorf("%s: failed to check token: %w", op, err)
	}

	updateQuery := `UPDATE users SET activated = true WHERE id = $1`
	_, err = tx.Exec(ctx, updateQuery, userID)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to activate user: %w", op, err)
	}

	deleteQuery := `DELETE FROM email_verification_tokens WHERE token = $1`
	_, err = tx.Exec(ctx, deleteQuery, token)
	if err != nil {
		return 0, fmt.Errorf("%s: failed to delete token: %w", op, err)
	}

	// 4. Коммитим транзакцию
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return userID, nil
}

func (s *Storage) GetVerificationTokenByUserID(ctx context.Context, userID int64) (string, time.Time, error) {
	const op = "storage.postgres.GetVerificationTokenByUserID"

	query := `SELECT token, expires_at FROM email_verification_tokens WHERE user_id = $1`

	var token string
	var expiresAt time.Time
	err := s.db.QueryRow(ctx, query, userID).Scan(&token, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", time.Time{}, fmt.Errorf("%s: %w", op, storage.ErrTokenNotFound)
		}
		return "", time.Time{}, fmt.Errorf("%s: %w", op, err)
	}

	return token, expiresAt, nil
}
