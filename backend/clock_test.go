package backend

import (
	"errors"
	"testing"
	"time"
)

func stubClock(t *testing.T, skew time.Duration, err error) (calls *int, now *time.Time) {
	t.Helper()
	oldSkew, oldNow := clockSkew, clockNow
	t.Cleanup(func() { clockSkew, clockNow = oldSkew, oldNow })
	n := 0
	at := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	clockSkew = func(string) (time.Duration, error) { n++; return skew, err }
	clockNow = func() time.Time { return at }
	return &n, &at
}

func TestClockNoticeWithinLimit(t *testing.T) {
	stubClock(t, 45*time.Second, nil)
	m := &Backend{}
	if n := m.GetClockNotice(); n.Show {
		t.Errorf("notice for a 45s skew: %+v", n)
	}
}

func TestClockNoticeAhead(t *testing.T) {
	stubClock(t, 3*time.Minute+20*time.Second, nil)
	m := &Backend{}
	if n := m.GetClockNotice(); n != (ClockNotice{Show: true, Minutes: 3, Ahead: true}) {
		t.Errorf("notice = %+v", n)
	}
}

func TestClockNoticeBehindHours(t *testing.T) {
	stubClock(t, -2*time.Hour-30*time.Minute, nil)
	m := &Backend{}
	if n := m.GetClockNotice(); n != (ClockNotice{Show: true, Minutes: 150, Ahead: false}) {
		t.Errorf("notice = %+v", n)
	}
}

func TestClockNoticeJustOverLimitSaysOneMinute(t *testing.T) {
	stubClock(t, -61*time.Second, nil)
	m := &Backend{}
	if n := m.GetClockNotice(); n != (ClockNotice{Show: true, Minutes: 1, Ahead: false}) {
		t.Errorf("notice = %+v", n)
	}
}

func TestClockNoticeCachedThenRechecked(t *testing.T) {
	calls, now := stubClock(t, 5*time.Minute, nil)
	m := &Backend{}
	m.GetClockNotice()
	*now = now.Add(9 * time.Minute)
	m.GetClockNotice()
	if *calls != 1 {
		t.Fatalf("checked %d times within 10 minutes, want 1", *calls)
	}
	*now = now.Add(2 * time.Minute)
	m.GetClockNotice()
	if *calls != 2 {
		t.Errorf("checked %d times after 11 minutes, want 2", *calls)
	}
}

func TestClockNoticeUnreachableShowsNothing(t *testing.T) {
	stubClock(t, 0, errors.New("offline"))
	m := &Backend{}
	if n := m.GetClockNotice(); n.Show {
		t.Errorf("notice without a reference time: %+v", n)
	}
}
