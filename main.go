package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"local-music/internal/api"
	"local-music/internal/catalog"
	"local-music/internal/library"
	"local-music/internal/storage"
)

//go:embed all:web
var webFiles embed.FS

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dbPath := env("MUSIC_DB", filepath.Join("data", "music.db"))
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return err
	}
	db, err := storage.Open(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	musicDir := env("MUSIC_DIR", "music")
	if err := os.MkdirAll(musicDir, 0o755); err != nil {
		return err
	}
	scanner, err := library.NewScanner(musicDir)
	if err != nil {
		return err
	}
	catalogService := catalog.New(db, scanner)
	if count, err := catalogService.Scan(context.Background()); err != nil {
		log.Printf("no se pudo completar el escaneo inicial: %v", err)
	} else {
		log.Printf("biblioteca escaneada: %d archivos de audio", count)
	}

	addr := env("MUSIC_ADDR", "127.0.0.1:8080")
	server := &http.Server{Addr: addr, Handler: api.New(db, webFiles, catalogService, scanner), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 2 * time.Minute, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("reproductor musical disponible en http://%s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
