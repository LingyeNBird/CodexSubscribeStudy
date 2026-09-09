package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB owns a serialized writer and a separate WAL reader pool.
// Every connection enforces foreign keys and waits for short-lived write locks.
type DB struct {
	writer *sql.DB
	reader *sql.DB
}

func Open(path string) (*DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var header [16]byte
	n, readErr := file.ReadAt(header[:], 0)
	closeErr := file.Close()
	if readErr != nil && readErr != io.EOF {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if n != 0 && (n != len(header) || string(header[:]) != "SQLite format 3\x00") {
		return nil, errors.New("database is not SQLite; choose a new STUDY_DB path and import the old file with -import-bbolt")
	}
	openPool := func(writer bool) (*sql.DB, error) {
		uriPath := filepath.ToSlash(absolute)
		if !strings.HasPrefix(uriPath, "/") {
			uriPath = "/" + uriPath
		}
		uri := url.URL{Scheme: "file", Path: uriPath}
		query := url.Values{}
		query.Add("_pragma", "foreign_keys(1)")
		query.Add("_pragma", "busy_timeout(5000)")
		query.Add("_pragma", "synchronous(FULL)")
		if writer {
			query.Set("_txlock", "immediate")
		} else {
			query.Add("_pragma", "query_only(1)")
		}
		uri.RawQuery = query.Encode()
		pool, err := sql.Open("sqlite", uri.String())
		if err != nil {
			return nil, err
		}
		if writer {
			pool.SetMaxOpenConns(1)
			pool.SetMaxIdleConns(1)
		} else {
			pool.SetMaxOpenConns(4)
			pool.SetMaxIdleConns(4)
		}
		if err := pool.Ping(); err != nil {
			pool.Close()
			return nil, err
		}
		return pool, nil
	}
	writer, err := openPool(true)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*DB, error) { writer.Close(); return nil, err }
	var journal string
	if err := writer.QueryRow("PRAGMA journal_mode=WAL").Scan(&journal); err != nil {
		return fail(err)
	}
	if journal != "wal" {
		return fail(fmt.Errorf("SQLite WAL unavailable: %s", journal))
	}
	if err := migrate(writer, migrations); err != nil {
		return fail(err)
	}
	reader, err := openPool(false)
	if err != nil {
		return fail(err)
	}
	return &DB{writer: writer, reader: reader}, nil
}

func (db *DB) Close() error { return errors.Join(db.reader.Close(), db.writer.Close()) }

func transaction(pool *sql.DB, readOnly bool, fn func(*sql.Tx) error) error {
	tx, err := pool.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) View(fn func(*sql.Tx) error) error   { return transaction(db.reader, true, fn) }
func (db *DB) Update(fn func(*sql.Tx) error) error { return transaction(db.writer, false, fn) }

// Backup makes a self-contained snapshot, including committed WAL contents.
// Existing destinations are never overwritten.
func (db *DB) Backup(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(absolute, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(absolute)
		return err
	}
	if _, err := db.writer.Exec("VACUUM INTO ?", absolute); err != nil {
		os.Remove(absolute)
		return err
	}
	file, err = os.OpenFile(absolute, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

func (db *DB) Check() error {
	return db.View(func(tx *sql.Tx) error {
		rows, err := tx.Query("PRAGMA integrity_check")
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var result string
			if err := rows.Scan(&result); err != nil {
				return err
			}
			if result != "ok" {
				return fmt.Errorf("SQLite integrity check failed: %s", result)
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		rows, err = tx.Query("PRAGMA foreign_key_check")
		if err != nil {
			return err
		}
		defer rows.Close()
		if rows.Next() {
			return errors.New("SQLite foreign key check failed")
		}
		return rows.Err()
	})
}
