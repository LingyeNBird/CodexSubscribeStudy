package main

import (
	"context"
	"flag"
	"github.com/LingyeNBird/CodexSubscribeStudy/internal/study"
	"github.com/LingyeNBird/CodexSubscribeStudy/web"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func main() {
	backup := flag.String("backup", "", "copy a consistent database backup to a local path and exit (stop server first)")
	flag.Parse()
	maximum, err := strconv.Atoi(env("STUDY_MAX_REPORTERS", "10000"))
	if err != nil || maximum < 1 || maximum > 100000 {
		log.Fatal("invalid STUDY_MAX_REPORTERS")
	}
	store, err := study.Open(env("STUDY_DB", "data/study.db"), maximum)
	if err != nil {
		log.Fatal("cannot open study database; check permissions and single-process lock")
	}
	defer store.Close()
	if *backup != "" {
		if err = store.Backup(*backup); err != nil {
			log.Fatal("backup failed")
		}
		log.Print("backup completed")
		return
	}
	if err = store.Prune(); err != nil {
		log.Fatal("database retention maintenance failed")
	}
	handler := study.NewServer(store, web.Assets())
	server := &http.Server{Addr: env("STUDY_ADDR", ":8080"), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192, ErrorLog: log.New(io.Discard, "", 0)}
	// TLS and network peers see IPs. The application deliberately has no access
	// log; configure the deployment proxy/CDN consistently with the privacy notice.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if store.Prune() != nil {
					log.Print("retention maintenance failed")
				}
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Print("Codex Subscribe Study started; no access logging")
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP service failed")
	}
}
