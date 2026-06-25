package setup

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func createSqliteDb(data []fullPokeData, path string) error {
	_, err := GetSqliteDb(path)
	if err != nil {
		return err
	}

	// TODO: Create Table Schema
	// TODO: Make sure foreign Keys are on for sqlite
	// TODO: ETL go struct to sql inserts
	// TODO: commit, cleanup, exit

	return nil
}

func internalGormDbSetup(db_path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(db_path), &gorm.Config{})
}
