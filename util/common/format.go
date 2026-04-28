package common

import (
	"fmt"
)

// FormatTraffic formats traffic bytes into human-readable units (B, KB, MB, GB, TB, PB).
func FormatTraffic(trafficBytes int64) string {
	if trafficBytes < 0 {
		trafficBytes = 0
	}
	return formatTrafficFloat(float64(trafficBytes))
}

// FormatTrafficUint formats unsigned traffic byte counters without narrowing.
func FormatTrafficUint(trafficBytes uint64) string {
	return formatTrafficFloat(float64(trafficBytes))
}

func formatTrafficFloat(trafficBytes float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	unitIndex := 0
	size := trafficBytes

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}
	return fmt.Sprintf("%.2f%s", size, units[unitIndex])
}
