package server

import (
	"encoding/json"
	"net/http"

	"github.com/veerbal1/paddock/internal/backend"
)

// Server is the owner-facing face of the backend: everything a human or a
// browser asks for goes through here.
type Server struct {
	backend *backend.Backend
}

func New(b *backend.Backend) *Server {
	return &Server{backend: b}
}

// Routes returns the HTTP handler for the whole API.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/cows", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.backend.Latest())
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode("ok")
	})

	return mux
}
