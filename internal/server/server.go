package server

import (
	"encoding/json"
	"net/http"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/fence"
)

// FenceSetter is the one downlink into the device fleet. The server owns
// this interface, so it never imports the sim package: main wires a
// *sim.Sim in, and the dependency arrow keeps pointing at this package,
// not away from it.
type FenceSetter interface {
	SetFence(fence.Rect)
	Fence() fence.Rect
}

// Server is the owner-facing face of the backend: everything a human or a
// browser asks for goes through here.
type Server struct {
	backend *backend.Backend
	fences  FenceSetter
}

func New(b *backend.Backend, fs FenceSetter) *Server {
	return &Server{backend: b, fences: fs}
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

	mux.HandleFunc("/fence", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(s.fences.Fence())
		case http.MethodPut:
			var f fence.Rect
			if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
				http.Error(w, "bad fence JSON", http.StatusBadRequest)
				return
			}
			if f.MinX >= f.MaxX || f.MinY >= f.MaxY {
				http.Error(w, "need Min < Max on both axes", http.StatusBadRequest)
				return
			}
			s.fences.SetFence(f)
			json.NewEncoder(w).Encode(f)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.backend.Alerts())
	})

	return mux
}
