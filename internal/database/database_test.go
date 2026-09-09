package database

import (
	"bytes"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func openDatabase(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "study.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
func TestTransactionsWALAndBackup(t *testing.T) {
	db := openDatabase(t)
	aborted := errors.New("abort")
	if err := db.Update(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO survey_submissions(status_mask,payload) VALUES(0,'{}')`); err != nil {
			return err
		}
		return aborted
	}); !errors.Is(err, aborted) {
		t.Fatal(err)
	}
	if err := db.View(func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			t.Fatal("failed transaction persisted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO survey_submissions(status_mask,payload) VALUES(0,'{}')`)
		return err
	}); err == nil {
		t.Fatal("read-only transaction allowed a write")
	}
	insert := func() error {
		return db.Update(func(tx *sql.Tx) error {
			_, err := tx.Exec(`INSERT INTO survey_submissions(status_mask,payload) VALUES(0,'{}')`)
			return err
		})
	}
	if err := insert(); err != nil {
		t.Fatal(err)
	}
	if err := db.View(func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			t.Fatal("unexpected first snapshot")
		}
		if err := insert(); err != nil {
			return err
		}
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&count); err != nil {
			return err
		}
		if count != 1 {
			t.Fatal("reader snapshot changed during write")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "backup.sqlite")
	if err := db.Backup(backup); err != nil {
		t.Fatal(err)
	}
	copied, err := Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer copied.Close()
	if err := copied.Check(); err != nil {
		t.Fatal(err)
	}
	if err := copied.View(func(tx *sql.Tx) error {
		var count int
		if err := tx.QueryRow("SELECT COUNT(*) FROM survey_submissions").Scan(&count); err != nil {
			return err
		}
		if count != 2 {
			t.Fatal("backup omitted WAL commits")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Backup(backup); err == nil {
		t.Fatal("backup overwrote existing file")
	}
	if err := db.Update(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO evidence_batches(reporter_id,batch_id,revision,digest,received_hour,report_json) VALUES(zeroblob(32),'batch','1','digest',0,'{}')`)
		return err
	}); err == nil {
		t.Fatal("foreign key violation accepted")
	}
}

func migrationFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	result := fstest.MapFS{}
	if err := fs.WalkDir(migrations, "migrations", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		body, err := fs.ReadFile(migrations, path)
		if err != nil {
			return err
		}
		result[path] = &fstest.MapFile{Data: body}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestMigrationHistoryAndAtomicFailure(t *testing.T) {
	db := openDatabase(t)
	if err := migrate(db.writer, migrations); err != nil {
		t.Fatal("repeat migrations", err)
	}
	files := migrationFiles(t)
	for _, file := range files {
		file.Data = []byte(strings.ReplaceAll(string(file.Data), "\n", "\r\n"))
	}
	if err := migrate(db.writer, files); err != nil {
		t.Fatal("line endings changed migration identity", err)
	}
	files = migrationFiles(t)
	files["migrations/0003_failure.sql"] = &fstest.MapFile{Data: []byte("CREATE TABLE partial_change(id INTEGER); INVALID SQL;")}
	if err := migrate(db.writer, files); err == nil {
		t.Fatal("broken migration succeeded")
	}
	var count int
	if err := db.writer.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name='partial_change'").Scan(&count); err != nil || count != 0 {
		t.Fatal("partial schema committed", err)
	}
	files = migrationFiles(t)
	files["migrations/0001_initial.sql"].Data = append(files["migrations/0001_initial.sql"].Data, []byte("\n-- changed")...)
	if err := migrate(db.writer, files); err == nil {
		t.Fatal("edited applied migration accepted")
	}
	if _, err := db.writer.Exec("INSERT INTO schema_migrations VALUES(99,'future','future','now')"); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db.writer, migrations); err == nil {
		t.Fatal("newer schema accepted by old application")
	}
}
func TestOpenDoesNotOverwriteNonSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	original := []byte("this is not a SQLite database")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	if db, err := Open(path); err == nil {
		db.Close()
		t.Fatal("non-SQLite database accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("source was modified")
	}
}
