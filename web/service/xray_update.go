package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/superaddmin/SuperXray-gui/v2/xray"
)

func coreAssetDirectory(cfg *xray.Config) (string, error) {
	var environment map[string]string
	if len(cfg.Env) != 0 {
		if err := json.Unmarshal(cfg.Env, &environment); err != nil {
			return "", errors.New("xray env must contain string values")
		}
	}
	// Match Xray's dotted-name precedence after applying config.env overrides.
	for _, key := range []string{"xray.location.asset", "XRAY_LOCATION_ASSET"} {
		if value, ok := environment[key]; ok {
			return filepath.Abs(value)
		}
		if value, ok := os.LookupEnv(key); ok {
			return filepath.Abs(value)
		}
	}
	return filepath.Abs(filepath.Dir(xray.GetBinaryPath()))
}

var coreVersionPattern = regexp.MustCompile(`^v([0-9]{2})\.([0-9]{1,2})\.([0-9]{1,2})$`)

func supportedCoreVersion(version string) bool {
	parts := coreVersionPattern.FindStringSubmatch(version)
	if parts == nil {
		return false
	}
	year, _ := strconv.Atoi(parts[1])
	month, _ := strconv.Atoi(parts[2])
	day, _ := strconv.Atoi(parts[3])
	return month >= 1 && month <= 12 && day >= 1 && day <= 31 &&
		(year > 26 || (year == 26 && (month > 10 || (month == 10 && day >= 10))))
}

// stageSupplementalGeo preserves local CustomGeo and regional databases while
// validating the new official geoip/geosite assets from the release archive.
func stageSupplementalGeo(sourceDir, stageDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".dat") || name == "geoip.dat" || name == "geosite.dat" || entry.IsDir() {
			continue
		}
		if err := copyStagedGeo(filepath.Join(sourceDir, name), filepath.Join(stageDir, name)); err != nil {
			return fmt.Errorf("stage supplemental geo file %s: %w", name, err)
		}
	}
	return nil
}

func copyStagedGeo(source, target string) error {
	// #nosec G304 -- source is enumerated from the administrator's existing Xray asset directory.
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDownloadFileBytes {
		return errors.New("geo asset must be a regular file within the download size limit")
	}
	// #nosec G304 -- target is a basename inside the updater's private staging directory.
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	copyErr := copyLimited(out, in, maxDownloadFileBytes)
	return errors.Join(copyErr, out.Sync(), out.Close())
}

type coreFileReplacement struct {
	target, staged, backup string
	hadOriginal, installed bool
}

type coreUpdateLifecycle struct {
	running func() bool
	stop    func() error
	start   func() error
}

// installCoreFiles keeps this operation's private backups until startup succeeds.
// retain is true only when recovery failed: callers must keep the staging tree.
func installCoreFiles(files []coreFileReplacement, lifecycle coreUpdateLifecycle) (retain bool, err error) {
	wasRunning := lifecycle.running()
	if wasRunning {
		if err := lifecycle.stop(); err != nil {
			return false, fmt.Errorf("stop Xray before update: %w", err)
		}
	}
	rollback := func(cause error) (bool, error) {
		if lifecycle.running() {
			if err := lifecycle.stop(); err != nil {
				return true, errors.Join(cause, fmt.Errorf("stop candidate core before restoring backup: %w", err))
			}
		}
		var restoreErr error
		for i := len(files) - 1; i >= 0; i-- {
			item := &files[i]
			if item.installed {
				if err := os.Remove(item.target); err != nil && !os.IsNotExist(err) {
					restoreErr = errors.Join(restoreErr, fmt.Errorf("remove candidate %s: %w", item.target, err))
					continue
				}
			}
			if item.hadOriginal {
				if err := os.Rename(item.backup, item.target); err != nil {
					restoreErr = errors.Join(restoreErr, fmt.Errorf("restore original from %s: %w", item.backup, err))
				}
			}
		}
		if restoreErr != nil {
			return true, errors.Join(cause, restoreErr)
		}
		if wasRunning {
			if err := lifecycle.start(); err != nil {
				return false, errors.Join(cause, fmt.Errorf("restart restored Xray: %w", err))
			}
		}
		return false, cause
	}
	for i := range files {
		item := &files[i]
		if _, err := os.Stat(item.backup); !os.IsNotExist(err) {
			return rollback(fmt.Errorf("refusing to overwrite core backup %s", item.backup))
		}
		if _, err := os.Stat(item.target); err == nil {
			if err := os.Rename(item.target, item.backup); err != nil {
				return rollback(fmt.Errorf("back up core file %s: %w", item.target, err))
			}
			item.hadOriginal = true
		} else if !os.IsNotExist(err) {
			return rollback(err)
		}
		if err := os.Rename(item.staged, item.target); err != nil {
			return rollback(fmt.Errorf("install core file %s: %w", item.target, err))
		}
		item.installed = true
	}
	if wasRunning {
		if err := lifecycle.start(); err != nil {
			return rollback(fmt.Errorf("start updated Xray: %w", err))
		}
	}
	return false, nil
}
