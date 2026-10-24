package util

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func dateServer(offset time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", time.Now().Add(offset).UTC().Format(http.TimeFormat))
	}))
}

func TestClockSkewLocalAhead(t *testing.T) {
	srv := dateServer(-5 * time.Minute)
	defer srv.Close()
	skew, err := ClockSkew(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if skew < 4*time.Minute || skew > 6*time.Minute {
		t.Errorf("skew = %v, want about +5m", skew)
	}
}

func TestClockSkewLocalBehind(t *testing.T) {
	srv := dateServer(3 * time.Hour)
	defer srv.Close()
	skew, err := ClockSkew(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if skew > -3*time.Hour+time.Minute || skew < -3*time.Hour-time.Minute {
		t.Errorf("skew = %v, want about -3h", skew)
	}
}

func TestClockSkewInSync(t *testing.T) {
	srv := dateServer(0)
	defer srv.Close()
	skew, err := ClockSkew(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	// The Date header has one-second resolution.
	if skew < -2*time.Second || skew > 2*time.Second {
		t.Errorf("skew = %v, want about 0", skew)
	}
}

func TestClockSkewNoDateHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header()["Date"] = nil // keep net/http from adding one
	}))
	defer srv.Close()
	if _, err := ClockSkew(srv.URL); err == nil {
		t.Error("expected an error without a Date header")
	}
}
