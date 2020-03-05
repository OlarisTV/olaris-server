package mysql

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func NewMySQLDatabase(connection string, dbLogMode bool) (*gorm.DB, error) {
	db, err := gorm.Open("mysql", connection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %s\n", err)
	}
	db.LogMode(dbLogMode)
	return db, nil
}
