package setup

import (
	"fmt"
	"go-pokebattle/dex"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Initialize gorm and sqlite, without a full rebuild step
func GetSqliteDb(db_path string) (*gorm.DB, error) {
	return internalGormDbSetup(db_path)
}

// Rebuild sqlite db and initialize gorm. May cause problems if file already exists
func CreateSqliteDb(data []FullPokeData, path string) (*gorm.DB, error) {
	return createSqliteDb(data, path)
}

func createSqliteDb(data []FullPokeData, dbPath string) (*gorm.DB, error) {
	db, err := internalGormDbSetup(dbPath)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&dex.Pokemon{}, &dex.Move{}, &dex.PokemonMove{}, &dex.Evolution{})
	if err != nil {
		return nil, err
	}

	// TODO: insert data from fullPokeData slices, scrapped from PokeAPI

	return db, nil
}

func internalGormDbSetup(dbPath string) (*gorm.DB, error) {
	const fkstr string = "?_foreign_keys=on"

	if !strings.Contains(dbPath, fkstr) {
		dbPath = fmt.Sprintf("%s%s", dbPath, fkstr)
	}

	fmt.Println(dbPath)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return db, err
	}

	if res := db.Exec("PRAGMA foreign_keys = ON", nil); res.Error != nil {
		return nil, res.Error
	}

	return db, nil
}
