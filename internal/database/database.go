package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func getDatabaseURL() string {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, name,
	)
}

func GetDatabase() (*sql.DB, error) {
	url := getDatabaseURL()
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	for attempt := 1; attempt <= 10; attempt++ {
		err = db.Ping()
		if err == nil {
			return db, nil
		}
		log.Printf("waiting for database (attempt %d/10): %v", attempt, err)
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
}