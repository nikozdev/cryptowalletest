package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
)

func initMigrationTable(db *sql.DB) error {
	query := `CREATE TABLE IF NOT EXISTS t_migration (
		v_id SERIAL PRIMARY KEY,
		v_name TEXT NOT NULL UNIQUE
	)`
	_, err := db.Exec(query)
	return err
}

func checkMigrationApplied(db *sql.DB, name string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM t_migration WHERE v_name = $1)`
	err := db.QueryRow(query, name).Scan(&exists)
	return exists, err
}

func RunMigrations(db *sql.DB, dir string) error {
	err := initMigrationTable(db)
	if err != nil {
		return fmt.Errorf("failed to init migration table: %w", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to glob migrations: %w", err)
	}
	sort.Strings(files)
	for _, file := range files {
		name := filepath.Base(file)
		applied, err := checkMigrationApplied(db, name)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", name, err)
		}
		if applied {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", name, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", name, err)
		}
		_, err = tx.Exec(string(content))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", name, err)
		}
		_, err = tx.Exec(
			`INSERT INTO t_migration (v_name) VALUES ($1)`,
			name,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", name, err)
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", name, err)
		}
		log.Printf("applied migration: %s", name)
	}
	return nil
}