package job

import (
	"io"
	"os"
	"path/filepath"

	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/pathutil"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

// ClearLogsJob clears old log files to prevent disk space issues.
type ClearLogsJob struct{}

// NewClearLogsJob creates a new log cleanup job instance.
func NewClearLogsJob() *ClearLogsJob {
	return new(ClearLogsJob)
}

// ensureFileExists creates the necessary directories and file if they don't exist
func ensureFileExists(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	file, err := pathutil.OpenFileUnder(dir, path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

// Here Run is an interface method of the Job interface
func (j *ClearLogsJob) Run() {
	logFiles := []string{xray.GetIPLimitLogPath(), xray.GetIPLimitBannedLogPath(), xray.GetAccessPersistentLogPath()}
	logFilesPrev := []string{xray.GetIPLimitBannedPrevLogPath(), xray.GetAccessPersistentPrevLogPath()}

	// Ensure all log files and their paths exist
	for _, path := range append(logFiles, logFilesPrev...) {
		if err := ensureFileExists(path); err != nil {
			logger.Warning("Failed to ensure log file exists:", path, "-", err)
		}
	}

	// Clear log files and copy to previous logs
	for i := range len(logFiles) {
		if i > 0 {
			// Copy to previous logs
			logFilePrev, err := pathutil.OpenFileUnder(filepath.Dir(logFilesPrev[i-1]), logFilesPrev[i-1], os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
			if err != nil {
				logger.Warning("Failed to open previous log file for writing:", logFilesPrev[i-1], "-", err)
				continue
			}

			logFile, err := pathutil.OpenFileUnder(filepath.Dir(logFiles[i]), logFiles[i], os.O_RDONLY, 0o600)
			if err != nil {
				logger.Warning("Failed to open current log file for reading:", logFiles[i], "-", err)
				if closeErr := logFilePrev.Close(); closeErr != nil {
					logger.Warning("Failed to close previous log file:", logFilesPrev[i-1], "-", closeErr)
				}
				continue
			}

			_, err = io.Copy(logFilePrev, logFile)
			if err != nil {
				logger.Warning("Failed to copy log file:", logFiles[i], "to", logFilesPrev[i-1], "-", err)
			}

			if closeErr := logFile.Close(); closeErr != nil {
				logger.Warning("Failed to close current log file:", logFiles[i], "-", closeErr)
			}
			if closeErr := logFilePrev.Close(); closeErr != nil {
				logger.Warning("Failed to close previous log file:", logFilesPrev[i-1], "-", closeErr)
			}
		}

		err := os.Truncate(logFiles[i], 0)
		if err != nil {
			logger.Warning("Failed to truncate log file:", logFiles[i], "-", err)
		}
	}
}
