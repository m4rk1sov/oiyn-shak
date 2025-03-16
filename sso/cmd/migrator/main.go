package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
}

//const (
//	envLocal = "local"
//	envDev   = "dev"
//	envProd  = "prod"
//)

func main() {
	var cfg config
	//var migrationsPath, migrationsTable, sslMode string

	//flag.IntVar(&cfg.port, "port", 4000, "API server port")
	//flag.StringVar(&cfg.env, "env", envLocal, "environment")
	flag.StringVar(&cfg.db.dsn, "dsn", os.Getenv("MIGRATE_STRING"), "database connection string")
	// path to migrations
	flag.StringVar(&cfg.migrationsPath, "migrations-path", "", "path to migrations")
	// table for keeping info about migrations
	flag.StringVar(&cfg.migrationsTable, "migrations-table", "migrations", "name of migrations table")
	// flag for sslmode
	flag.StringVar(&cfg.sslMode, "sslmode", "disable", "sslmode")
	flag.Parse()

	if cfg.migrationsPath == "" {
		panic("migrations-path is required")
	}

	m, err := migrate.New(
		"file://"+cfg.migrationsPath,
		fmt.Sprintf("postgres://%s?x-migrations-table=%s&sslmode=%s", cfg.db.dsn, cfg.migrationsTable, cfg.sslMode),
	)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")

			return
		}

		panic(err)
	}
}
