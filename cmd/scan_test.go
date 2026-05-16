package cmd

import (
	"testing"
	"time"
)

func TestParseSince_Hours(t *testing.T) {
	before := time.Now()
	got := parseSince("2h")
	want := before.Add(-2 * time.Hour)
	delta := got.Sub(want)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("parseSince(2h) off by %v", delta)
	}
}

func TestParseSince_Days(t *testing.T) {
	before := time.Now()
	got := parseSince("7d")
	want := before.Add(-7 * 24 * time.Hour)
	delta := got.Sub(want)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("parseSince(7d) off by %v", delta)
	}
}

func TestParseSince_Empty(t *testing.T) {
	got := parseSince("")
	if !got.IsZero() {
		t.Errorf("parseSince('') should return zero time, got %v", got)
	}
}

func TestParseSince_Invalid(t *testing.T) {
	// Invalid input falls back to 24h.
	before := time.Now()
	got := parseSince("notaduration")
	want := before.Add(-24 * time.Hour)
	delta := got.Sub(want)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("parseSince(invalid) should fall back to 24h, off by %v", delta)
	}
}

func TestParseSince_30Days(t *testing.T) {
	before := time.Now()
	got := parseSince("30d")
	want := before.Add(-30 * 24 * time.Hour)
	delta := got.Sub(want)
	if delta < -time.Second || delta > time.Second {
		t.Errorf("parseSince(30d) off by %v", delta)
	}
}
