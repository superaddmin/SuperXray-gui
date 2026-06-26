package logger

import (
	"fmt"
	"sync"
	"testing"
)

func TestLogBufferIsConcurrentSafe(t *testing.T) {
	logBufferMu.Lock()
	logBuffer = nil
	logBufferMu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addToBuffer("INFO", fmt.Sprintf("entry-%d", i))
		}(i)
	}
	wg.Wait()

	if got := len(GetLogs(100, "INFO")); got == 0 {
		t.Fatal("GetLogs() returned no entries after concurrent writes")
	}
}
