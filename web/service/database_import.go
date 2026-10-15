package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/superaddmin/SuperXray-gui/v2/database"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"gorm.io/gorm"
)

// runtimeMaintenanceMu serializes core restarts, upgrades and DB replacement.
var runtimeMaintenanceMu sync.Mutex

// InitDB applies every panel migration before making the connection available.
// Startup, CLI migration, and import use this same transaction boundary.
func InitDB(dbPath string) error {
	return database.InitDBWithMigration(dbPath, migratePanelDatabase)
}

func migratePanelDatabase(tx *gorm.DB) error {
	inbounds := NewInboundService(database.NewRepositories(tx).Inbounds)
	if err := inbounds.migrationRequirementsOn(tx); err != nil {
		return fmt.Errorf("migrate historical inbound data: %w", err)
	}
	if err := migrationRemoveOrphanedClientTrafficsOn(tx); err != nil {
		return fmt.Errorf("migrate orphaned client traffic: %w", err)
	}
	return nil
}

func validateImportCandidate(candidate string) error {
	gdb, err := database.PrepareDBWithMigration(candidate, migratePanelDatabase)
	if err != nil {
		return err
	}
	pool, err := gdb.DB()
	if err != nil {
		return err
	}
	defer pool.Close()
	repositories := database.NewRepositories(gdb)
	previewInboundService := NewInboundService(repositories.Inbounds)
	preview := &XrayService{
		inboundService: *previewInboundService,
		settingService: *NewSettingService(repositories.Settings),
	}
	configuration, err := preview.getXrayConfig(false)
	if err != nil {
		return err
	}
	return xray.ValidateConfig(configuration)
}

type databaseImportLifecycle struct {
	running    bool
	stop       func() error
	start      func() error
	checkpoint func() error
	close      func() error
	open       func(string) error
}

// replacePreparedDatabase only runs after candidate migration and core validation.
// The original file is kept until the new database and runtime both succeed.
func replacePreparedDatabase(candidate, active string, lifecycle databaseImportLifecycle) error {
	backup, err := os.CreateTemp(filepath.Dir(active), ".superxray-rollback-*.db")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	if err := lifecycle.checkpoint(); err != nil {
		return fmt.Errorf("checkpoint before database import: %w", err)
	}
	if lifecycle.running {
		if err := lifecycle.stop(); err != nil {
			return errors.Join(fmt.Errorf("stop Xray before database import: %w", err), lifecycle.start())
		}
	}
	reopenOriginal := func(cause error) error {
		if err := lifecycle.open(active); err != nil {
			return errors.Join(cause, fmt.Errorf("reopen original database: %w", err))
		}
		if lifecycle.running {
			return errors.Join(cause, lifecycle.start())
		}
		return cause
	}
	if err := lifecycle.close(); err != nil {
		return reopenOriginal(fmt.Errorf("close original database: %w", err))
	}
	// A live WAL/SHM after our last connection closed means another process
	// still owns this database. Refuse replacement instead of orphaning its WAL.
	for _, suffix := range []string{"-wal", "-shm", "-journal"} {
		if _, err := os.Stat(active + suffix); err == nil {
			return reopenOriginal(fmt.Errorf("database sidecar %s is still in use; stop other database writers and retry", suffix))
		} else if !os.IsNotExist(err) {
			return reopenOriginal(err)
		}
	}
	if err := os.Rename(active, backupPath); err != nil {
		return reopenOriginal(fmt.Errorf("back up original database: %w", err))
	}

	restore := func(cause error, installed bool) error {
		if err := lifecycle.close(); err != nil {
			return errors.Join(cause, fmt.Errorf("close failed database; original retained at %s: %w", backupPath, err))
		}
		if installed {
			// Vacate the destination first: Windows cannot replace an existing DB
			// with os.Rename, unlike Unix. Preserve it until rollback succeeds.
			if err := os.Rename(active, candidate); err != nil {
				return errors.Join(cause, fmt.Errorf("remove failed database; original retained at %s: %w", backupPath, err))
			}
		}
		if err := os.Rename(backupPath, active); err != nil {
			return errors.Join(cause, fmt.Errorf("restore original database retained at %s: %w", backupPath, err))
		}
		return reopenOriginal(cause)
	}
	if err := os.Rename(candidate, active); err != nil {
		return restore(fmt.Errorf("install imported database: %w", err), false)
	}
	if err := lifecycle.open(active); err != nil {
		return restore(fmt.Errorf("open imported database: %w", err), true)
	}
	if lifecycle.running {
		if err := lifecycle.start(); err != nil {
			cause := fmt.Errorf("start imported Xray configuration: %w", err)
			if stopErr := lifecycle.stop(); stopErr != nil {
				cause = errors.Join(cause, fmt.Errorf("stop failed imported runtime: %w", stopErr))
			}
			return restore(cause, true)
		}
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("database imported; remove obsolete rollback file %s: %w", backupPath, err)
	}
	return nil
}
