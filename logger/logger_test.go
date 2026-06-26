package logger

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/op/go-logging"
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

func TestLoggerInitAndWarningAreConcurrentSafe(t *testing.T) {
	t.Setenv("XUI_LOG_FOLDER", filepath.Join(t.TempDir(), "logs"))
	InitLogger(logging.ERROR)
	t.Cleanup(CloseLogger)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if worker%2 == 0 {
					InitLogger(logging.ERROR)
					continue
				}
				Warningf("race probe %d-%d", worker, j)
			}
		}(i)
	}
	wg.Wait()
}
