package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/fence"
	"github.com/veerbal1/paddock/internal/mqttx"
	"github.com/veerbal1/paddock/internal/store"
)

// FenceSetter is the one downlink into the device fleet. The server owns
// this interface, so it never imports the sim package: main wires the
// pieces in, and the dependency arrow keeps pointing at this package,
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
	st      *store.Store
	mq      *mqttx.Client
}

func New(b *backend.Backend, fs FenceSetter) *Server {
	return &Server{backend: b, fences: fs}
}

// WithDownlink plugs diary + board behind PUT. Without it PUT only
// touches memory (unit tests); with it PUT persists and publishes.
func (s *Server) WithDownlink(st *store.Store, mq *mqttx.Client) *Server {
	s.st = st
	s.mq = mq
	return s
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
			if s.st != nil && s.mq != nil {
				if err := s.publishDownlink(r.Context(), f); err != nil {
					http.Error(w, "downlink failed", http.StatusInternalServerError)
					return
				}
			}
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

// publishDownlink writes the drawing to the diary, then pins it on the
// board. Diary first: if the board fails, the next backend boot
// re-pins from the diary.
func (s *Server) publishDownlink(ctx context.Context, f fence.Rect) error {
	raw, err := json.Marshal(f)
	if err != nil {
		return err
	}
	dctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	v, err := s.st.SaveFence(dctx, "1", "p1", raw)
	if err != nil {
		return err
	}
	if err := s.mq.PublishFence("1", "p1", v, raw); err != nil {
		log.Printf("downlink: board v%d failed: %v", v, err)
		return err
	}
	return nil
}
