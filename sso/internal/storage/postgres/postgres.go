package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/jackc/pgx"
)

type Storage struct {
	db *sql.DB
}

func New(dsn string) (*Storage, error) {
	const op = "storage.postgres.New"

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveUser(ctx context.Context, email string, passwordHash []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	stmt, err := s.db.Prepare(`INSERT INTO users(email, password_hash) VALUES(?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	res, err := stmt.ExecContext(ctx, email, passwordHash)
	if err != nil {
		var postgresErr pgx.PgError

		if errors.As(err, &postgresErr) && postgresErr.Error() == "duplicate key value violates unique constraint \"users_email_key\"" {
			
		}
	}
}
