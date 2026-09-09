package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/LingyeNBird/CodexSubscribeStudy/internal/study"
	"github.com/LingyeNBird/CodexSubscribeStudy/web"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func run() error {
	backup := flag.String("backup", "", "write a consistent SQLite snapshot to a new file and exit")
	importPath := flag.String("import-bbolt", "", "import a read-only bbolt file into an empty SQLite database and exit")
	flag.Parse()
	if *backup != "" && *importPath != "" {
		return errors.New("use -backup or -import-bbolt, not both")
	}
	maximum, err := strconv.Atoi(env("STUDY_MAX_REPORTERS", "10000"))
	if err != nil || maximum < 1 || maximum > 100000 {
		return errors.New("invalid STUDY_MAX_REPORTERS")
	}
	dbPath := env("STUDY_DB", "data/study.sqlite")
	store, err := study.Open(dbPath, maximum)
	if err != nil {
		return fmt.Errorf("open SQLite database: %w", err)
	}
	defer store.Close()
	legacyPath := *importPath
	explicit := legacyPath != ""
	if !explicit {
		var configured bool
		legacyPath, configured = os.LookupEnv("STUDY_LEGACY_DB")
		if !configured {
			legacyPath = filepath.Join(filepath.Dir(dbPath), "study.db")
		}
		imported, err := store.HasLegacyImport()
		if err != nil {
			return err
		}
		if imported {
			legacyPath = ""
		}
	}
	if legacyPath != "" {
		_, statErr := os.Stat(legacyPath)
		if explicit || !errors.Is(statErr, os.ErrNotExist) {
			if statErr != nil {
				return fmt.Errorf("read legacy database: %w", statErr)
			}
			result, err := store.ImportBbolt(legacyPath)
			if err != nil {
				return fmt.Errorf("import legacy database: %w", err)
			}
			log.Printf("Legacy import completed: surveys=%d identities=%d batches=%d already_imported=%t; source retained", result.Surveys, result.Identities, result.Batches, result.AlreadyImported)
		}
	}
	if explicit {
		return nil
	}
	if *backup != "" {
		if err := store.Backup(*backup); err != nil {
			return fmt.Errorf("backup SQLite: %w", err)
		}
		log.Print("SQLite backup completed")
		return nil
	}
	handler := study.NewServer(store, web.Assets())
	server := &http.Server{Addr: env("STUDY_ADDR", ":8080"), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192, ErrorLog: log.New(io.Discard, "", 0)}
	// TLS and network peers see IPs. The application deliberately has no access
	// log; configure the deployment proxy/CDN consistently with the privacy notice.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Print("Codex Subscribe Study started; SQLite WAL storage; no access logging")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP service failed: %w", err)
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
