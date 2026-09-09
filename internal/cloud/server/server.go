package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/veerbal1/paddock/internal/cloud/backend"
	"github.com/veerbal1/paddock/internal/shared/fence"
)

// FenceSetter is the one downlink into the device fleet. The server owns
// this interface, so it never imports the sim package: main wires the
// pieces in, and the dependency arrow keeps pointing at this package,
// not away from it.
type FenceSetter interface {
	SetFence(fence.Rect)
	Fence() fence.Rect
}

// FenceStore is the diary: where an authored fence is written down before
// anyone is told about it. FencePublisher is the notice board it gets
// pinned to. Both are declared here, next to the handler that needs them,
// for the same reason as FenceSetter — and with a concrete *store.Store or
// *mqttx.Client in these fields, PUT could only ever be tested against a
// real Postgres and a real broker.
type FenceStore interface {
	SaveFence(ctx context.Context, farmID, paddockID string, polygon []byte) (int, error)
}

type FencePublisher interface {
	PublishFence(farmID, paddockID string, version int, polygon []byte) error
}

// Server is the owner-facing face of the backend: everything a human or a
// browser asks for goes through here.
type Server struct {
	backend *backend.Backend
	fences  FenceSetter

	st        FenceStore
	mq        FencePublisher
	farmID    string
	paddockID string
}

func New(b *backend.Backend, fs FenceSetter) *Server {
	return &Server{backend: b, fences: fs}
}

// WithDownlink plugs diary + board behind PUT, for one farm's paddock.
// Without it PUT only touches memory (unit tests); with it PUT persists
// and publishes.
func (s *Server) WithDownlink(st FenceStore, mq FencePublisher, farmID, paddockID string) *Server {
	s.st = st
	s.mq = mq
	s.farmID = farmID
	s.paddockID = paddockID
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
			if s.st != nil && s.mq != nil {
				// Downlink before the local copy: if the fence cannot be
				// recorded and announced, the owner must not be told it
				// was applied.
				if err := s.publishDownlink(r.Context(), f); err != nil {
					http.Error(w, "downlink failed", http.StatusInternalServerError)
					return
				}
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
	v, err := s.st.SaveFence(dctx, s.farmID, s.paddockID, raw)
	if err != nil {
		log.Printf("downlink: diary write failed: %v", err)
		return err
	}
	if err := s.mq.PublishFence(s.farmID, s.paddockID, v, raw); err != nil {
		log.Printf("downlink: board v%d failed: %v", v, err)
		return err
	}
	return nil
}
