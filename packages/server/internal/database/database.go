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
	dsn := filepath.Join(dbPath, "cyimewrite.db")

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
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
