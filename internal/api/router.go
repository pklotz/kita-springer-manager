package api

import (
	"database/sql"
	"io"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/pak/kita-springer-manager/internal/api/handlers"
	"github.com/pak/kita-springer-manager/internal/transit"
)

func NewRouter(db *sql.DB, tc *transit.Client, assets fs.FS) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:9092"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	h := handlers.New(db, tc)

	r.Route("/api", func(r chi.Router) {
		r.Route("/providers", func(r chi.Router) {
			r.Get("/", h.ListProviders)
			r.Post("/", h.CreateProvider)
			r.Put("/{id}", h.UpdateProvider)
			r.Delete("/{id}", h.DeleteProvider)
			r.Post("/{id}/seed-kitas", h.SeedKitas)
			r.Post("/{id}/import-excel", h.ImportExcel)
		})
		r.Route("/kitas", func(r chi.Router) {
			r.Get("/", h.ListKitas)
			r.Post("/", h.CreateKita)
			r.Post("/import", h.ImportKitasExcel)
			r.Get("/{id}", h.GetKita)
			r.Put("/{id}", h.UpdateKita)
			r.Delete("/{id}", h.DeleteKita)
		})
		r.Route("/assignments", func(r chi.Router) {
			r.Get("/", h.ListAssignments)
			r.Post("/", h.CreateAssignment)
			r.Post("/bulk-delete", h.BulkDeleteAssignments)
			r.Get("/{id}", h.GetAssignment)
			r.Put("/{id}", h.UpdateAssignment)
			r.Delete("/{id}", h.DeleteAssignment)
		})
		r.Route("/recurring", func(r chi.Router) {
			r.Get("/", h.ListRecurring)
			r.Post("/", h.CreateRecurring)
			r.Delete("/{id}", h.DeleteRecurring)
		})
		r.Route("/closures", func(r chi.Router) {
			r.Get("/", h.ListClosures)
			r.Post("/", h.CreateClosure)
			r.Delete("/{id}", h.DeleteClosure)
		})
		r.Get("/settings", h.GetSettings)
		r.Put("/settings", h.UpdateSettings)
		r.Route("/transit", func(r chi.Router) {
			r.Get("/connections", h.GetConnections)
			r.Get("/stops", h.SearchStops)
		})
		r.Get("/calendar.ics", h.GetCalendar)
	})

	fileServer := http.FileServer(http.FS(assets))
	r.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if path != "" && path[0] == '/' {
			path = path[1:]
		}
		if path == "" || !fileExists(assets, path) {
			serveIndex(w, req, assets)
			return
		}
		fileServer.ServeHTTP(w, req)
	}))

	return r
}

func fileExists(assets fs.FS, name string) bool {
	f, err := assets.Open(name)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

func serveIndex(w http.ResponseWriter, _ *http.Request, assets fs.FS) {
	f, err := assets.Open("index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}
