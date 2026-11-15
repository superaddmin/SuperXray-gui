package web

import (
	"testing"
	"time"
)

func TestCronAcceptsFiveAndSixFieldSchedules(t *testing.T) {
	location := time.FixedZone("test-zone", 8*60*60)
	scheduler := newCronScheduler(location)
	start := time.Date(2026, 11, 15, 10, 29, 0, 0, location)
	for _, spec := range []string{"30 10 * * *", "0 30 10 * * *"} {
		id, err := scheduler.AddFunc(spec, func() {})
		if err != nil {
			t.Fatalf("AddFunc(%q): %v", spec, err)
		}
		want := start.Add(time.Minute)
		if next := scheduler.Entry(id).Schedule.Next(start); !next.Equal(want) {
			t.Fatalf("next run for %q = %s, want %s", spec, next, want)
		}
	}
	if _, err := scheduler.AddFunc("bad schedule", func() {}); err == nil {
		t.Fatal("invalid schedule accepted")
	}
}
