package sqlite

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func NewSQLiteDatabase(dbPath string, dbLogMode bool) (*gorm.DB, error) {
	db, err := gorm.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=1000")
	//db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %s\n", err)
	}
	db.LogMode(dbLogMode)
	legacyMigration(db)
	return db, nil
}

func legacyMigration(db *gorm.DB) {
	// NOTE(Leon Handreke): We do this here because some databases were initialized and
	// used before we introduced gormigrate. This code can be removed once we are
	// sure that no users with v0.1.x databases remain.
	migrationsTableExists := 0
	db.Table("sqlite_master").
		Where("type = 'table'").
		Where("name = 'migrations'").
		Count(&migrationsTableExists)
	usersTableExists := 0
	db.Table("sqlite_master").
		Where("type = 'table'").
		Where("name = 'users'").
		Count(&usersTableExists)
	if migrationsTableExists == 0 && usersTableExists == 1 {
		db.Exec("CREATE TABLE migrations (id VARCHAR(255) PRIMARY KEY)")
		db.Exec("INSERT INTO migrations (id) VALUES ('SCHEMA_INIT')")
	}
}
