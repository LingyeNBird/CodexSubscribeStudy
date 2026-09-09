package database

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

func migrate(db *sql.DB, files fs.FS) error {
	entries, err := fs.ReadDir(files, "migrations")
	if err != nil {
		return err
	}
	return transaction(db, false, func(tx *sql.Tx) error {
		if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
   version INTEGER PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TEXT NOT NULL
  ) STRICT`); err != nil {
			return err
		}
		type applied struct{ name, checksum string }
		history := map[int]applied{}
		rows, err := tx.Query("SELECT version,name,checksum FROM schema_migrations ORDER BY version")
		if err != nil {
			return err
		}
		for rows.Next() {
			var version int
			var item applied
			if err := rows.Scan(&version, &item.name, &item.checksum); err != nil {
				rows.Close()
				return err
			}
			history[version] = item
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		version := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				return fmt.Errorf("invalid migration file %q", entry.Name())
			}
			prefix, _, ok := strings.Cut(entry.Name(), "_")
			number, err := strconv.Atoi(prefix)
			version++
			if !ok || len(prefix) != 4 || err != nil || number != version {
				return fmt.Errorf("migration numbering must be contiguous: %s", entry.Name())
			}
			body, err := fs.ReadFile(files, "migrations/"+entry.Name())
			if err != nil {
				return err
			}
			body = []byte(strings.ReplaceAll(string(body), "\r\n", "\n"))
			digest := fmt.Sprintf("%x", sha256.Sum256(body))
			if old, exists := history[number]; exists {
				if old.name != entry.Name() || old.checksum != digest {
					return fmt.Errorf("applied migration %d has changed", number)
				}
				delete(history, number)
				continue
			}
			// A gap in the recorded history is not a partially applied migration.
			for recorded := range history {
				if recorded > number {
					return fmt.Errorf("migration history is missing version %d", number)
				}
			}
			if _, err := tx.Exec(string(body)); err != nil {
				return fmt.Errorf("migration %s: %w", entry.Name(), err)
			}
			if _, err := tx.Exec("INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,?,?,?)", number, entry.Name(), digest, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		if len(history) != 0 {
			return fmt.Errorf("database contains migrations newer than this application")
		}
		return nil
	})
}
