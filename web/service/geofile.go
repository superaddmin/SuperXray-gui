package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/logger"
	"github.com/superaddmin/SuperXray-gui/v2/util/pathutil"
	"github.com/xtls/xray-core/common/geodata"
	"google.golang.org/protobuf/proto"
)

var (
	defaultGeofileUpdateMu = sync.Mutex{}
	validGeofilePattern    = regexp.MustCompile(`^[a-zA-Z0-9._-]+\.dat$`)
)

var defaultGeofileEntries = []geofileEntry{
	{URL: "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat", FileName: "geoip.dat"},
	{URL: "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat", FileName: "geosite.dat"},
	{URL: "https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geoip.dat", FileName: "geoip_IR.dat"},
	{URL: "https://github.com/chocolate4u/Iran-v2ray-rules/releases/latest/download/geosite.dat", FileName: "geosite_IR.dat"},
	{URL: "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geoip.dat", FileName: "geoip_RU.dat"},
	{URL: "https://github.com/runetfreedom/russia-v2ray-rules-dat/releases/latest/download/geosite.dat", FileName: "geosite_RU.dat"},
}

type geofileEntry struct {
	URL      string
	FileName string
}

type geofileUpdater struct {
	client      *http.Client
	binDir      string
	entries     []geofileEntry
	validate    func() error
	restart     func() error
	replaceFile func(string, string) error
}

type stagedGeofile struct {
	entry    geofileEntry
	destPath string
	tempPath string
	modTime  time.Time
}

type geofileCommitRecord struct {
	file        *stagedGeofile
	backupPath  string
	hadOriginal bool
}

// IsValidGeofileName validates the controller path parameter before it reaches
// the stricter service allowlist.
func (s *ServerService) IsValidGeofileName(fileName string) bool {
	if fileName == "" || strings.Contains(fileName, "..") || strings.ContainsAny(fileName, `/\`) {
		return false
	}
	return !filepath.IsAbs(fileName) && validGeofilePattern.MatchString(fileName)
}

func (s *ServerService) UpdateGeofile(fileName string) error {
	defaultGeofileUpdateMu.Lock()
	defer defaultGeofileUpdateMu.Unlock()

	updater := geofileUpdater{
		client:      serviceHTTPClient,
		binDir:      config.GetBinFolderPath(),
		entries:     defaultGeofileEntries,
		validate:    s.validateXrayConfigForGeofileUpdate,
		restart:     s.RestartXrayService,
		replaceFile: os.Rename,
	}
	return updater.update(fileName)
}

func (s *ServerService) validateXrayConfigForGeofileUpdate() error {
	xrayConfig, err := s.xrayService.GetXrayConfig()
	if err != nil {
		return fmt.Errorf("build Xray configuration: %w", err)
	}
	data, err := json.MarshalIndent(xrayConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal Xray configuration: %w", err)
	}

	binDir := config.GetBinFolderPath()
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		return fmt.Errorf("create Xray bin directory: %w", err)
	}
	tempFile, err := os.CreateTemp(binDir, ".geofile-config-*.json")
	if err != nil {
		return fmt.Errorf("create temporary Xray configuration: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	_, writeErr := tempFile.Write(data)
	if writeErr == nil {
		writeErr = tempFile.Sync()
	}
	closeErr := tempFile.Close()
	if writeErr != nil {
		return fmt.Errorf("write temporary Xray configuration: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close temporary Xray configuration: %w", closeErr)
	}

	if _, err := runXrayCommand("run", "-test", "-config", tempPath); err != nil {
		return fmt.Errorf("Xray configuration validation failed: %w", err)
	}
	return nil
}

func (u geofileUpdater) selectEntries(fileName string) ([]geofileEntry, error) {
	if fileName == "" {
		return u.entries, nil
	}
	for _, entry := range u.entries {
		if entry.FileName == fileName {
			return []geofileEntry{entry}, nil
		}
	}
	return nil, fmt.Errorf("invalid geofile name %q: not in allowlist", fileName)
}

func (u geofileUpdater) stage(entry geofileEntry) (*stagedGeofile, error) {
	destPath := filepath.Join(u.binDir, entry.FileName)
	localValid := validateGeofile(destPath, entry.FileName) == nil
	req, err := http.NewRequest(http.MethodGet, entry.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if localValid {
		if info, statErr := os.Stat(destPath); statErr == nil && !info.ModTime().IsZero() {
			req.Header.Set("If-Modified-Since", info.ModTime().UTC().Format(http.TimeFormat))
		}
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		if !localValid {
			return nil, fmt.Errorf("server returned not modified without a valid local geofile")
		}
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	if err := validateContentLength(resp, maxDownloadFileBytes); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(u.binDir, 0o750); err != nil {
		return nil, fmt.Errorf("create bin directory: %w", err)
	}
	tempFile, err := os.CreateTemp(u.binDir, "."+entry.FileName+"-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create temporary geofile: %w", err)
	}
	tempPath := tempFile.Name()

	written, copyErr := io.Copy(tempFile, io.LimitReader(resp.Body, maxDownloadFileBytes+1))
	if copyErr == nil && written > maxDownloadFileBytes {
		copyErr = fmt.Errorf("response body exceeds %d bytes", maxDownloadFileBytes)
	}
	if copyErr == nil {
		copyErr = tempFile.Sync()
	}
	closeErr := tempFile.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("save temporary geofile: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("close temporary geofile: %w", closeErr)
	}
	if err := validateGeofile(tempPath, entry.FileName); err != nil {
		_ = os.Remove(tempPath)
		return nil, err
	}

	var modTime time.Time
	if raw := resp.Header.Get("Last-Modified"); raw != "" {
		parsed, parseErr := time.Parse(http.TimeFormat, raw)
		if parseErr != nil {
			logger.Warningf("Failed to parse Last-Modified header for %s: %v", entry.FileName, parseErr)
		} else {
			modTime = parsed
			if err := os.Chtimes(tempPath, modTime, modTime); err != nil {
				_ = os.Remove(tempPath)
				return nil, fmt.Errorf("set temporary geofile modification time: %w", err)
			}
		}
	}

	return &stagedGeofile{entry: entry, destPath: destPath, tempPath: tempPath, modTime: modTime}, nil
}

func cleanupStagedGeofiles(staged []*stagedGeofile) {
	for _, file := range staged {
		_ = os.Remove(file.tempPath)
	}
}

func createGeofileBackup(binDir string, file *stagedGeofile) (string, bool, error) {
	info, err := os.Lstat(file.destPath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !info.Mode().IsRegular() {
		return "", false, fmt.Errorf("existing path is not a regular file")
	}

	source, err := pathutil.OpenUnder(binDir, file.destPath)
	if err != nil {
		return "", false, err
	}
	defer source.Close()

	backup, err := os.CreateTemp(binDir, "."+file.entry.FileName+"-*.backup")
	if err != nil {
		return "", false, err
	}
	backupPath := backup.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(backupPath)
		}
	}()

	written, copyErr := io.Copy(backup, io.LimitReader(source, maxDownloadFileBytes+1))
	if copyErr == nil && written > maxDownloadFileBytes {
		copyErr = fmt.Errorf("existing geofile exceeds %d bytes", maxDownloadFileBytes)
	}
	if copyErr == nil {
		copyErr = backup.Sync()
	}
	closeErr := backup.Close()
	if copyErr != nil {
		return "", false, copyErr
	}
	if closeErr != nil {
		return "", false, closeErr
	}
	if err := os.Chmod(backupPath, info.Mode().Perm()); err != nil {
		return "", false, err
	}
	if err := os.Chtimes(backupPath, info.ModTime(), info.ModTime()); err != nil {
		return "", false, err
	}
	cleanup = false
	return backupPath, true, nil
}

func (u geofileUpdater) rollback(records []geofileCommitRecord) error {
	var rollbackErrors []string
	for i := len(records) - 1; i >= 0; i-- {
		record := &records[i]
		if record.hadOriginal {
			if err := u.replaceFile(record.backupPath, record.file.destPath); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Sprintf(
					"restore %s failed: %v; recovery backup preserved at %s",
					record.file.entry.FileName, err, record.backupPath,
				))
				continue
			}
			record.backupPath = ""
			continue
		}
		if err := os.Remove(record.file.destPath); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("remove new %s: %v", record.file.entry.FileName, err))
		}
	}
	if len(rollbackErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(rollbackErrors, "; "))
	}
	return nil
}

func (u geofileUpdater) commit(staged []*stagedGeofile) ([]geofileCommitRecord, error) {
	if u.replaceFile == nil {
		u.replaceFile = os.Rename
	}
	records := make([]geofileCommitRecord, 0, len(staged))
	for _, file := range staged {
		backupPath, hadOriginal, err := createGeofileBackup(u.binDir, file)
		if err != nil {
			rollbackErr := u.rollback(records)
			if rollbackErr != nil {
				return nil, fmt.Errorf("backup %s: %w; rollback failed: %v", file.entry.FileName, err, rollbackErr)
			}
			return nil, fmt.Errorf("backup %s: %w", file.entry.FileName, err)
		}
		records = append(records, geofileCommitRecord{file: file, backupPath: backupPath, hadOriginal: hadOriginal})
		if err := u.replaceFile(file.tempPath, file.destPath); err != nil {
			rollbackErr := u.rollback(records)
			if rollbackErr != nil {
				return nil, fmt.Errorf("replace %s: %w; rollback failed: %v", file.entry.FileName, err, rollbackErr)
			}
			return nil, fmt.Errorf("replace %s: %w", file.entry.FileName, err)
		}
	}
	return records, nil
}

func cleanupGeofileBackups(records []geofileCommitRecord) {
	for _, record := range records {
		if record.backupPath == "" {
			continue
		}
		if err := os.Remove(record.backupPath); err != nil && !os.IsNotExist(err) {
			logger.Warningf("Failed to remove geofile backup %s: %v", record.backupPath, err)
		}
	}
}

func (u geofileUpdater) update(fileName string) error {
	entries, err := u.selectEntries(fileName)
	if err != nil {
		return err
	}
	if u.client == nil {
		return fmt.Errorf("geofile HTTP client is not configured")
	}
	if u.replaceFile == nil {
		u.replaceFile = os.Rename
	}

	staged := make([]*stagedGeofile, 0, len(entries))
	var stageErrors []string
	for _, entry := range entries {
		file, stageErr := u.stage(entry)
		if stageErr != nil {
			stageErrors = append(stageErrors, fmt.Sprintf("%s: %v", entry.FileName, stageErr))
			continue
		}
		if file != nil {
			staged = append(staged, file)
		}
	}
	defer cleanupStagedGeofiles(staged)

	if len(stageErrors) > 0 {
		return fmt.Errorf("geofile update failed: %s", strings.Join(stageErrors, "; "))
	}
	if len(staged) == 0 {
		return nil
	}
	records, err := u.commit(staged)
	if err != nil {
		return fmt.Errorf("commit geofile update: %w", err)
	}
	if u.validate != nil {
		validationErr := u.validate()
		if validationErr != nil {
			rollbackErr := u.rollback(records)
			if rollbackErr != nil {
				return fmt.Errorf("validate Xray configuration after geofile update: %v; rollback failed: %w", validationErr, rollbackErr)
			}
			return fmt.Errorf("validate Xray configuration after geofile update: %w; previous geofiles were restored", validationErr)
		}
	}
	if u.restart == nil {
		cleanupGeofileBackups(records)
		return nil
	}
	restartErr := u.restart()
	if restartErr == nil {
		cleanupGeofileBackups(records)
		return nil
	}

	rollbackErr := u.rollback(records)
	if rollbackErr != nil {
		return fmt.Errorf("Xray restart failed after geofile update: %v; rollback failed: %w", restartErr, rollbackErr)
	}
	restoreRestartErr := u.restart()
	if restoreRestartErr != nil {
		return fmt.Errorf(
			"Xray restart failed after geofile update: %v; geofiles were rolled back but restoring Xray failed: %w",
			restartErr, restoreRestartErr,
		)
	}
	return fmt.Errorf("Xray restart failed after geofile update and the previous geofiles were restored: %w", restartErr)
}

func validateGeofile(path, fileName string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("read geofile for validation: %w", err)
	}
	defer file.Close()
	data, err := readBodyLimited(file, maxDownloadFileBytes)
	if err != nil {
		return fmt.Errorf("read geofile for validation: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("geofile is empty")
	}

	if strings.HasPrefix(fileName, "geoip") {
		var list geodata.GeoIPList
		if err := proto.Unmarshal(data, &list); err != nil {
			return fmt.Errorf("invalid geoip data: %w", err)
		}
		for _, entry := range list.Entry {
			if entry != nil && strings.TrimSpace(entry.Code) != "" {
				return nil
			}
		}
		return fmt.Errorf("geoip data has no usable entries")
	}

	var list geodata.GeoSiteList
	if err := proto.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("invalid geosite data: %w", err)
	}
	for _, entry := range list.Entry {
		if entry != nil && strings.TrimSpace(entry.Code) != "" {
			return nil
		}
	}
	return fmt.Errorf("geosite data has no usable entries")
}
