package main

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// runMigrations applies embedded SQL files in filename order and records a
// checksum for each one. Re-running is safe; editing an applied file fails
// loudly instead of silently drifting the schema.
func runMigrations(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version varchar(64) PRIMARY KEY,
		checksum varchar(64) NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`).Error; err != nil {
		return err
	}
	names, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return err
	}
	files := []string{}
	for _, n := range names {
		if !n.IsDir() && strings.HasSuffix(n.Name(), ".sql") {
			files = append(files, "migrations/"+n.Name())
		}
	}
	sort.Strings(files)
	for _, file := range files {
		body, err := migrationFiles.ReadFile(file)
		if err != nil {
			return err
		}
		version := strings.TrimSuffix(path.Base(file), ".sql")
		sum := fmt.Sprintf("%x", sha256.Sum256(body))
		var stored string
		res := db.Raw("SELECT checksum FROM schema_migrations WHERE version = ?", version).Scan(&stored)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			if stored != sum {
				return fmt.Errorf("migration %s was modified after being applied (checksum mismatch)", version)
			}
			continue
		}
		tx := db.Begin()
		if tx.Error != nil {
			return tx.Error
		}
		if err := tx.Exec(string(body)).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}
		if err := tx.Exec("INSERT INTO schema_migrations (version, checksum) VALUES (?, ?)", version, sum).Error; err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit().Error; err != nil {
			return err
		}
	}
	return nil
}

// migrationStatus is exposed for operational checks.
func migrationStatus(db *gorm.DB) ([]map[string]any, error) {
	rows, err := db.Raw("SELECT version, checksum, applied_at FROM schema_migrations ORDER BY version").Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var version, checksum string
		var appliedAt any
		if err := rows.Scan(&version, &checksum, &appliedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"version": version, "checksum": checksum, "applied_at": appliedAt})
	}
	return out, rows.Err()
}
