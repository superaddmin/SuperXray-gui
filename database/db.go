// Package database provides database initialization, migration, and management utilities
// for the SuperXray panel using GORM with SQLite.
package database

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"slices"
	"sync/atomic"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/config"
	"github.com/superaddmin/SuperXray-gui/v2/database/model"
	"github.com/superaddmin/SuperXray-gui/v2/util/crypto"
	"github.com/superaddmin/SuperXray-gui/v2/util/xraycompat"
	"github.com/superaddmin/SuperXray-gui/v2/xray"
	"golang.org/x/crypto/bcrypt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db atomic.Pointer[gorm.DB]

const (
	defaultUsername           = "admin"
	defaultPassword           = "admin"
	baselineMigrationVersion  = "202606260001"
	baselineMigrationName     = "baseline-auto-migrate"
	baselineMigrationChecksum = "gorm-auto-migrate-current-models"
)

func initModels(gdb *gorm.DB) error {
	models := []any{
		&model.User{},
		&model.Inbound{},
		&model.OutboundTraffics{},
		&model.Setting{},
		&model.InboundClientIps{},
		&xray.ClientTraffic{},
		&model.HistoryOfSeeders{},
		&model.CustomGeoResource{},
		&model.SchemaMigration{},
		&model.MigrationEvent{},
	}
	for _, model := range models {
		if err := gdb.AutoMigrate(model); err != nil {
			log.Printf("Error auto migrating model: %v", err)
			return err
		}
	}
	return nil
}

func recordBaselineMigration(gdb *gorm.DB) error {
	now := time.Now().UnixMilli()

	return gdb.Transaction(func(tx *gorm.DB) error {
		migration := model.SchemaMigration{
			Version:    baselineMigrationVersion,
			Name:       baselineMigrationName,
			Checksum:   baselineMigrationChecksum,
			DurationMs: 0,
			Status:     "applied",
		}
		if err := tx.Where("version = ?", baselineMigrationVersion).
			Attrs(migration).
			FirstOrCreate(&migration).Error; err != nil {
			return err
		}

		event := model.MigrationEvent{
			Version:    baselineMigrationVersion,
			Direction:  "up",
			StartedAt:  now,
			FinishedAt: now,
			Status:     "applied",
		}
		return tx.Where("version = ? AND direction = ?", baselineMigrationVersion, "up").
			Attrs(event).
			FirstOrCreate(&event).Error
	})
}

// initUser creates a default admin user if the users table is empty.
func initUser(gdb *gorm.DB) error {
	empty, err := isTableEmpty(gdb, "users")
	if err != nil {
		log.Printf("Error checking if users table is empty: %v", err)
		return err
	}
	if empty {
		hashedPassword, err := crypto.HashPasswordAsBcrypt(defaultPassword)

		if err != nil {
			log.Printf("Error hashing default password: %v", err)
			return err
		}

		user := &model.User{
			Username: defaultUsername,
			Password: hashedPassword,
		}
		return gdb.Create(user).Error
	}
	return nil
}

// runSeeders migrates user passwords to bcrypt and records seeder execution to prevent re-running.
func runSeeders(gdb *gorm.DB, isUsersEmpty bool) error {
	return gdb.Transaction(func(tx *gorm.DB) error {
		empty, err := isTableEmpty(tx, "history_of_seeders")
		if err != nil {
			log.Printf("Error checking if users table is empty: %v", err)
			return err
		}

		if empty && isUsersEmpty {
			hashSeeder := &model.HistoryOfSeeders{
				SeederName: "UserPasswordHash",
			}
			return tx.Create(hashSeeder).Error
		} else {
			var seedersHistory []string
			if err := tx.Model(&model.HistoryOfSeeders{}).Pluck("seeder_name", &seedersHistory).Error; err != nil {
				return err
			}

			if !slices.Contains(seedersHistory, "UserPasswordHash") && !isUsersEmpty {
				var users []model.User
				if err := tx.Find(&users).Error; err != nil {
					return err
				}

				for _, user := range users {
					if _, err := bcrypt.Cost([]byte(user.Password)); err == nil {
						continue
					}
					hashedPassword, err := crypto.HashPasswordAsBcrypt(user.Password)
					if err != nil {
						log.Printf("Error hashing password for user '%s': %v", user.Username, err)
						return err
					}
					if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("password", hashedPassword).Error; err != nil {
						return err
					}
				}

				hashSeeder := &model.HistoryOfSeeders{
					SeederName: "UserPasswordHash",
				}
				return tx.Create(hashSeeder).Error
			}
		}

		return nil
	})
}

// isTableEmpty returns true if the named table contains zero rows.
func isTableEmpty(gdb *gorm.DB, tableName string) (bool, error) {
	var count int64
	err := gdb.Table(tableName).Count(&count).Error
	return count == 0, err
}

// InitDB sets up the database connection, migrates models, and runs seeders.
func InitDB(dbPath string) error {
	return InitDBWithMigration(dbPath, nil)
}

// InitDBWithMigration publishes a connection only after schema, compatibility,
// and caller-owned data migrations commit together.
func InitDBWithMigration(dbPath string, migrate func(*gorm.DB) error) error {
	gdb, err := PrepareDBWithMigration(dbPath, migrate)
	if err != nil {
		return err
	}
	previousDB := db.Swap(gdb)
	if previousDB != nil {
		_ = closeGormDB(previousDB)
	}
	return nil
}

// PrepareDB migrates a private candidate connection without changing GetDB.
// The caller owns the returned connection and must close its SQL pool.
func PrepareDB(dbPath string) (*gorm.DB, error) {
	return PrepareDBWithMigration(dbPath, nil)
}

// PrepareDBWithMigration runs migrate inside the initialization transaction.
// The hook must use the supplied transaction and must not publish it globally.
func PrepareDBWithMigration(dbPath string, migrate func(*gorm.DB) error) (*gorm.DB, error) {
	dir := path.Dir(dbPath)
	err := os.MkdirAll(dir, 0o750)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- path is the configured database or a private candidate, never an upload filename.
	file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	protectErr := file.Chmod(0o600)
	closeErr := file.Close()
	if err := errors.Join(protectErr, closeErr); err != nil {
		return nil, err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}
	gdb, err := gorm.Open(sqlite.Open(dbPath), c)
	if err != nil {
		return nil, err
	}
	failed := true
	defer func() {
		if !failed {
			return
		}
		_ = closeGormDB(gdb)
	}()

	err = gdb.Transaction(func(tx *gorm.DB) error {
		if err := initModels(tx); err != nil {
			return err
		}
		if err := recordBaselineMigration(tx); err != nil {
			return err
		}
		isUsersEmpty, err := isTableEmpty(tx, "users")
		if err != nil {
			return err
		}
		if err := initUser(tx); err != nil {
			return err
		}
		if err := runSeeders(tx, isUsersEmpty); err != nil {
			return err
		}
		if err := normalizeStoredXrayConfig(tx); err != nil {
			return err
		}
		if migrate != nil {
			return migrate(tx)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	failed = false
	return gdb, nil
}

func normalizeStoredXrayConfig(tx *gorm.DB) error {
	var inbounds []model.Inbound
	if err := tx.Find(&inbounds).Error; err != nil {
		return err
	}
	for _, inbound := range inbounds {
		settings, stream, changed, err := xraycompat.NormalizeInbound(string(inbound.Protocol), inbound.Settings, inbound.StreamSettings)
		if err != nil {
			return fmt.Errorf("normalize inbound %d (%s): %w", inbound.Id, inbound.Tag, err)
		}
		if changed {
			if err := tx.Model(&model.Inbound{}).Where("id = ?", inbound.Id).Updates(map[string]any{"settings": settings, "stream_settings": stream}).Error; err != nil {
				return err
			}
		}
	}
	var settings []model.Setting
	if err := tx.Find(&settings).Error; err != nil {
		return err
	}
	for _, setting := range settings {
		if setting.Key != "xrayTemplateConfig" {
			continue
		}
		updated, changed, err := xraycompat.NormalizeConfig(setting.Value)
		if err != nil {
			return fmt.Errorf("normalize setting %q: %w", setting.Key, err)
		}
		if changed {
			if err := tx.Model(&model.Setting{}).Where("id = ?", setting.Id).Update("value", updated).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func closeGormDB(gdb *gorm.DB) error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CloseDB closes the database connection if it exists.
func CloseDB() error {
	// Keep a non-nil, closed pool during replacement. Concurrent callers receive
	// sql.ErrConnDone instead of a nil-pointer panic; InitDB publishes only a
	// completely migrated replacement using an atomic swap.
	return closeGormDB(db.Load())
}

// GetDB returns the global GORM database instance.
func GetDB() *gorm.DB {
	return db.Load()
}

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsSQLiteDB checks if the given file is a valid SQLite database by reading its signature.
func IsSQLiteDB(file io.ReaderAt) (bool, error) {
	signature := []byte("SQLite format 3\x00")
	buf := make([]byte, len(signature))
	_, err := file.ReadAt(buf, 0)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, signature), nil
}

// Checkpoint performs a WAL checkpoint on the SQLite database to ensure data consistency.
func Checkpoint() error {
	db := GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	var result struct{ Busy, Log, Checkpointed int }
	if err := db.Raw("PRAGMA wal_checkpoint(TRUNCATE)").Scan(&result).Error; err != nil {
		return err
	}
	if result.Busy != 0 {
		return errors.New("database checkpoint is busy; retry after active database operations finish")
	}
	return nil
}

// BackupSQLite creates an online, transactionally consistent SQLite snapshot.
// VACUUM INTO includes committed WAL pages and avoids copying a changing file.
func BackupSQLite(destination string) error {
	db := GetDB()
	if db == nil {
		return errors.New("database is not initialized")
	}
	return db.Exec("VACUUM INTO ?", destination).Error
}

// ValidateSQLiteDB opens the provided sqlite DB path with a throw-away connection
// and runs a PRAGMA integrity_check to ensure the file is structurally sound.
// It does not mutate global state or run migrations.
func ValidateSQLiteDB(dbPath string) error {
	if _, err := os.Stat(dbPath); err != nil { // file must exist
		return err
	}
	gdb, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	var res string
	if err := gdb.Raw("PRAGMA integrity_check;").Scan(&res).Error; err != nil {
		return err
	}
	if res != "ok" {
		return errors.New("sqlite integrity check failed: " + res)
	}
	return nil
}
