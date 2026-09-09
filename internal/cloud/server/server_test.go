package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/veerbal1/paddock/internal/cloud/backend"
	"github.com/veerbal1/paddock/internal/shared/fence"
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

// fakeDownlink is both diary and board, recording the order it was used in.
// The order is the point: a fence must be on disk before the fleet is told
// about it, or a crash leaves collars enforcing a fence nobody remembers.
type fakeDownlink struct {
	calls   []string
	version int
	saveErr error
	pubErr  error
}

func (f *fakeDownlink) SaveFence(_ context.Context, farmID, paddockID string, _ []byte) (int, error) {
	f.calls = append(f.calls, "save "+farmID+"/"+paddockID)
	if f.saveErr != nil {
		return 0, f.saveErr
	}
	f.version++
	return f.version, nil
}

func (f *fakeDownlink) PublishFence(farmID, paddockID string, version int, _ []byte) error {
	f.calls = append(f.calls, fmt.Sprintf("publish %s/%s v%d", farmID, paddockID, version))
	return f.pubErr
}

func putFence(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/fence", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	return rec
}

func TestPutFenceSavesBeforePublishing(t *testing.T) {
	fakes := &fakeFences{}
	dl := &fakeDownlink{}
	srv := New(backend.New(), fakes).WithDownlink(dl, dl, "f9", "north")

	rec := putFence(t, srv, `{"MinX":0,"MinY":0,"MaxX":10,"MaxY":10}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}

	want := []string{"save f9/north", "publish f9/north v1"}
	if !reflect.DeepEqual(dl.calls, want) {
		t.Fatalf("downlink calls = %v, want %v", dl.calls, want)
	}
	if fakes.calls != 1 {
		t.Errorf("SetFence called %d times, want 1", fakes.calls)
	}
}

func TestPutFenceDoesNotPublishWhenDiskFails(t *testing.T) {
	fakes := &fakeFences{}
	dl := &fakeDownlink{saveErr: errors.New("disk is on fire")}
	srv := New(backend.New(), fakes).WithDownlink(dl, dl, "f9", "north")

	rec := putFence(t, srv, `{"MinX":0,"MinY":0,"MaxX":10,"MaxY":10}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("PUT status = %d, want 500", rec.Code)
	}
	if len(dl.calls) != 1 {
		t.Errorf("calls = %v, want the save attempt only — nothing may be announced", dl.calls)
	}
	if fakes.calls != 0 {
		t.Errorf("SetFence called %d times, want 0: the owner was told it failed", fakes.calls)
	}
}

func TestPutFenceReportsBoardFailure(t *testing.T) {
	fakes := &fakeFences{}
	dl := &fakeDownlink{pubErr: errors.New("broker unreachable")}
	srv := New(backend.New(), fakes).WithDownlink(dl, dl, "f9", "north")

	rec := putFence(t, srv, `{"MinX":0,"MinY":0,"MaxX":10,"MaxY":10}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("PUT status = %d, want 500 — the fleet never heard it", rec.Code)
	}
	// The version is on disk regardless: boot re-pins it from there.
	if len(dl.calls) != 2 {
		t.Errorf("calls = %v, want save then a failed publish", dl.calls)
	}
}
