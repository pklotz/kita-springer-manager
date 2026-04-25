package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	frontendassets "github.com/pak/kita-springer-manager/frontend"
	"github.com/pak/kita-springer-manager/internal/api"
	"github.com/pak/kita-springer-manager/internal/db"
	"github.com/pak/kita-springer-manager/internal/store"
	"github.com/pak/kita-springer-manager/internal/transit"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":9092"), "HTTP listen address (env ADDR)")
	dbPath := flag.String("db", envOr("DB_PATH", defaultDBPath()), "SQLite database file (env DB_PATH)")
	flag.Parse()

	absDB, _ := filepath.Abs(*dbPath)
	if err := os.MkdirAll(filepath.Dir(absDB), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}
	if _, err := os.Stat(absDB); os.IsNotExist(err) {
		log.Printf("Datenbank nicht gefunden — neue Datenbank wird angelegt: %s", absDB)
	} else {
		log.Printf("Datenbank: %s", absDB)
	}

	database, err := db.Open(absDB)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	settings, err := store.GetSettings(database)
	if err != nil {
		log.Fatalf("read settings: %v", err)
	}
	canton := settings.Canton
	if !store.IsValidCanton(canton) {
		canton = "BE"
	}
	thisYear := time.Now().Year()
	for y := thisYear; y <= thisYear+2; y++ {
		if err := store.SeedHolidays(database, canton, y); err != nil {
			log.Printf("warn: seed holidays %d (%s): %v", y, canton, err)
		}
	}

	if n, err := store.CleanupPastConnections(database, time.Now().Format("2006-01-02")); err != nil {
		log.Printf("warn: cleanup cached connections: %v", err)
	} else if n > 0 {
		log.Printf("cleaned %d cached connections for past assignments", n)
	}

	router := api.NewRouter(database, transit.NewClient(), frontendassets.DistFS())

	srv := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Server running on http://localhost%s", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultDBPath returns data/app.db relative to the binary's directory,
// so the server can be run from any working directory.
func defaultDBPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "data/app.db"
	}
	return filepath.Join(filepath.Dir(exe), "data", "app.db")
}
