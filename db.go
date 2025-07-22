package main

import (
	"database/sql"
	"fmt"
	"os"
)

func getDBConnection() (*sql.DB, error) {
	password := os.Getenv("PGPASSWORD")
	if password == "" {
		return nil, fmt.Errorf("PGPASSWORD environment variable not set")
	}

	connStr := fmt.Sprintf("host=localhost port=5432 user=postgres password=%s dbname=dbos sslmode=disable", password)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("database connection failed: %w", err)
	}

	return db, nil
}

func InitializeSchema() error {
	db, err := getDBConnection()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	_, err = db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}
