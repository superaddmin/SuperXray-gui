package common

import "testing"

func TestFormatTrafficClampsNegativeValues(t *testing.T) {
	if got := FormatTraffic(-1); got != "0.00B" {
		t.Fatalf("FormatTraffic(-1) = %q, want 0.00B", got)
	}
}

func TestFormatTrafficUintHandlesLargeCounters(t *testing.T) {
	if got := FormatTrafficUint(^uint64(0)); got == "" {
		t.Fatal("FormatTrafficUint returned an empty string")
	}
}
