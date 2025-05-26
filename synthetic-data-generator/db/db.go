package db

import (
	"database/sql"
	"fmt"
	"synthetic-data-generator/env"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	conStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		env.GetEnvString("DB_USER", "postgres"),
		env.GetEnvString("DB_PASSWORD", "postgres"),
		env.GetEnvString("DB_HOST", "localhost"),
		env.GetEnvString("DB_PORT", "5432"),
		env.GetEnvString("DB_NAME", "postgres"),
	)

	db, err := sql.Open("postgres", conStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
