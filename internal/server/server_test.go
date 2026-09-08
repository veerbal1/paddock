package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/veerbal1/paddock/internal/backend"
	"github.com/veerbal1/paddock/internal/fence"
)

// fakeFences records what the handler pushed down.
type fakeFences struct {
	current fence.Rect
	calls   int
}

func (f *fakeFences) SetFence(r fence.Rect) { f.current = r; f.calls++ }
func (f *fakeFences) Fence() fence.Rect     { return f.current }

func TestPutFencePushesDown(t *testing.T) {
	fakes := &fakeFences{current: fence.Rect{MinX: 0, MaxX: 100, MinY: 0, MaxY: 100}}
	srv := New(backend.New(), fakes)

	req := httptest.NewRequest(http.MethodPut, "/fence",
		strings.NewReader(`{"MinX":40,"MinY":40,"MaxX":60,"MaxY":60}`))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if fakes.calls != 1 {
		t.Fatalf("SetFence called %d times, want 1", fakes.calls)
	}
	want := fence.Rect{MinX: 40, MaxX: 60, MinY: 40, MaxY: 60}
	if fakes.current != want {
		t.Fatalf("fence = %+v, want %+v", fakes.current, want)
	}
}

func TestPutFenceRejectsInsideOut(t *testing.T) {
	fakes := &fakeFences{}
	srv := New(backend.New(), fakes)

	req := httptest.NewRequest(http.MethodPut, "/fence",
		strings.NewReader(`{"MinX":60,"MinY":40,"MaxX":40,"MaxY":60}`))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT status = %d, want 400", rec.Code)
	}
	if fakes.calls != 0 {
		t.Fatalf("SetFence called %d times, want 0", fakes.calls)
	}
}
