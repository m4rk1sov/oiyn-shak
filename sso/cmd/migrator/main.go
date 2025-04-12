package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"log"
	"os"
	//"sso/internal/config"
)

type config struct {
	//port int
	//env  string
	migrationsPath  string
	migrationsTable string
	sslMode         string
	db              struct {
		dsn string
	}
	version int
	cmd     string
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Error loading .env file")
	}
	var cfg config

	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("MIGRATE_STRING"), "database connection string")
	// path to migrations
	flag.StringVar(&cfg.migrationsPath, "migrations-path", os.Getenv("MIGRATE_PATH"), "path to migrations")
	// table for keeping info about migrations
	flag.StringVar(&cfg.migrationsTable, "migrations-table", "migrations", "name of migrations table")
	// flag for sslmode
	flag.StringVar(&cfg.sslMode, "sslmode", os.Getenv("SSL_MODE"), "sslmode")
	// type of migration action (up, down, force, steps (-+N))
	flag.StringVar(&cfg.cmd, "cmd", "", "migration command: up, down, force, steps (-N for down, +N for up)")
	// version of action (int)
	flag.IntVar(&cfg.version, "version", 0, "migration version for: (fix and force, steps up and down)")
	flag.Parse()

	if cfg.db.dsn == "" {
		panic("dsn is empty")
	}

	if cfg.migrationsPath == "" {
		panic("migrations-path is required")
	}

	if cfg.cmd == "" {
		panic("migrations command is required")
	}

	if cfg.version == 0 {
		log.Println("version is zero")
	}

	m, err := migrate.New(
		"file://"+cfg.migrationsPath,
		fmt.Sprintf("postgres://%s?x-migrations-table=%s&sslmode=%s", cfg.db.dsn, cfg.migrationsTable, cfg.sslMode),
	)
	if err != nil {
		panic(err)
	}

	switch cfg.cmd {
	case "up":
		if err = m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("no migrations to apply")
				return
			}
			panic(err)
		}
	case "down":
		if err = m.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("no migrations to rollback")
				return
			}
			panic(err)
		}
	case "force":
		if cfg.version >= 0 {
			err := m.Force(cfg.version)
			if err != nil {
				if errors.Is(err, migrate.ErrInvalidVersion) {
					fmt.Println("invalid version")
					return
				}
				panic(fmt.Sprintf("failed to force dirty fix to version %d: %v", cfg.version, err))
			}
			fmt.Printf("Forced dirty migration to version %d\n", cfg.version)
		}
	case "steps":
		if err = m.Steps(cfg.version); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("no migrations to apply")
				return
			}
			panic(err)
		}
	default:
		panic("unknown migration command")
	}

}
