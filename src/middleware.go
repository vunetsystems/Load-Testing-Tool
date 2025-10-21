package main

import (
	"log"
	"net/http"
	"path/filepath"
	"time"
	"vuDataSim/src/handlers"

	"github.com/rs/cors"
)

// Middleware for logging requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Middleware for CORS
func corsMiddleware(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Configure appropriately for production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	return c.Handler(next)
}

// Serve static files with proper MIME types
func serveStatic(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for root path
	if r.URL.Path == "/" {
		http.ServeFile(w, r, handlers.StaticDir+"/index.html")
		return
	}

	// Serve other static files with proper MIME types
	staticPath := handlers.StaticDir + r.URL.Path

	const contentTypeHeader = "Content-Type"

	// Set proper MIME types based on file extension
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".css":
		w.Header().Set(contentTypeHeader, "text/css")
	case ".js":
		w.Header().Set(contentTypeHeader, "application/javascript")
	case ".html":
		w.Header().Set(contentTypeHeader, "text/html")
	case ".json":
		w.Header().Set(contentTypeHeader, "application/json")
	case ".ico":
		w.Header().Set(contentTypeHeader, "image/x-icon")
	case ".png":
		w.Header().Set(contentTypeHeader, "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set(contentTypeHeader, "image/jpeg")
	case ".svg":
		w.Header().Set(contentTypeHeader, "image/svg+xml")
	}

	http.ServeFile(w, r, staticPath)
}
