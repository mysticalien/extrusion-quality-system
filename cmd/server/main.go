package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprint(w, "Homepage!")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, `{"status":"ok"}`)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 3 * time.Second,
	}

	// TODO: add graceful shutdown
	// если ошибка не http.ErrServerClosed → это проблема
	logger.Info("starting server", "addr", server.Addr)
	err := server.ListenAndServe()
	if err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
