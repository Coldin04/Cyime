package database

import (
	"log"
	"os"
	"path/filepath"

	"g.co1d.in/Coldin04/CyimeWrite/server/internal/config"
	"g.co1d.in/Coldin04/CyimeWrite/server/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect initializes the database connection and runs auto-migrations.
func Connect() {
	var err error

	// Use a logger to see generated SQL
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * 1000,  // Slow SQL threshold
			LogLevel:                  logger.Info, // Log level
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Disable color
		},
	)

	// For simplicity, we'll place the SQLite file in the user's home directory.
	// A better approach for production would be a configurable path.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	dbPath := filepath.Join(homeDir, ".cyimewrite")
	if err := os.MkdirAll(dbPath, os.ModePerm); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}
	// SQLite DSN with safe defaults:
	//   _journal_mode=WAL       — readers don't block writers and vice versa.
	//   _busy_timeout=5000      — wait up to 5s on locked db before SQLITE_BUSY.
	//   _foreign_keys=1         — enforce ON DELETE CASCADE declared in models.
	//   _synchronous=NORMAL     — durability/perf trade-off appropriate for WAL.
	//   _txlock=immediate       — acquire RESERVED lock on BEGIN to avoid
	//                             SQLITE_BUSY on transaction promotion.
	dsn := filepath.Join(dbPath, "cyimewrite.db") +
		"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1&_synchronous=NORMAL&_txlock=immediate"

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// SQLite serializes writers; opening multiple write connections only causes
	// SQLITE_BUSY contention. Pin the pool to a single connection so GORM does
	// not silently fan out under load.
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to access underlying *sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)

	// Verify foreign keys are actually enabled. mattn/go-sqlite3 will silently
	// ignore the DSN flag if compiled without the FK feature, and the rest of
	// the schema relies on ON DELETE CASCADE for cleanup, so refuse to boot if
	// they are off.
	var fkEnabled int
	if err := DB.Raw("PRAGMA foreign_keys").Scan(&fkEnabled).Error; err != nil {
		log.Fatalf("Failed to read foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		log.Fatalf("SQLite foreign keys are not enabled (got %d); refusing to start", fkEnabled)
	}

	// Verify WAL is active so we don't silently fall back to rollback journal.
	var journalMode string
	if err := DB.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		log.Fatalf("Failed to read journal_mode pragma: %v", err)
	}
	if journalMode != "wal" && journalMode != "WAL" {
		log.Printf("Warning: SQLite journal_mode=%q (expected wal); concurrent reads may block writers", journalMode)
	}

	log.Println("Database connection established.")

	// Optional development reset (disabled by default).
	// Set RESET_WORKSPACE_TABLES_ON_BOOT=true to drop workspace/content/media tables.
	if config.IsTrue(os.Getenv("RESET_WORKSPACE_TABLES_ON_BOOT")) {
		resetTables := []string{
			"blob_gc_jobs",
			"blob_objects",
			"asset_gc_jobs",
			"assets",
			"document_asset_refs",
			"document_bodies",
			"documents",
			"folders",
			// Legacy table name from previous schema.
			"document_contents",
		}
		for _, table := range resetTables {
			if DB.Migrator().HasTable(table) {
				if err := DB.Migrator().DropTable(table); err != nil {
					log.Fatalf("Failed to drop table %s: %v", table, err)
				}
			}
		}
	}

	// Auto-migrate the schema
	err = DB.AutoMigrate(
		&models.User{},
		&models.UserImageBedConfig{},
		&models.AuthProvider{},
		&models.UserIdentityProvider{},
		&models.UserSession{},
		&models.UserRefreshToken{},
		&models.Folder{},
		&models.Document{},
		&models.DocumentBody{},
		&models.DocumentPermission{},
		&models.DocumentImageTargetPreference{},
		&models.DocumentInvite{},
		&models.Notification{},
		&models.BlobObject{},
		&models.Asset{},
		&models.DocumentAssetRef{},
		&models.AssetGCJob{},
		&models.BlobGCJob{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	log.Println("Database migrated.")
}
