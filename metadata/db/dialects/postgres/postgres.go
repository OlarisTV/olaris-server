package postgres

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func NewPostgresDatabase(connection string, dbLogMode bool) (*gorm.DB, error) {
	db, err := gorm.Open("postgres", connection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %s\n", err)
	}
	db.LogMode(dbLogMode)
	return db, nil
}
