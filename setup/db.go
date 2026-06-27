package setup

import (
	"fmt"
	"go-pokebattle/dex"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	FOREIGNKEYSTR string = "?_foreign_keys=on"
)

// Initialize gorm and sqlite, without a full rebuild step
func GetSqliteDb(dbPath string) (*gorm.DB, error) {
	if !strings.Contains(dbPath, FOREIGNKEYSTR) {
		dbPath = fmt.Sprintf("%s%s", dbPath, FOREIGNKEYSTR)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return db, err
	}

	if res := db.Exec("PRAGMA foreign_keys = ON", nil); res.Error != nil {
		return nil, res.Error
	}

	return db, nil
}

// This will take data from the api and try to create and insert sqlite tables and data
func CreateSqliteDb(apiData []PokeApiData, dbPath string) (*gorm.DB, error) {
	fmt.Println("Initializing db @filepath:", dbPath)

	db, err := GetSqliteDb(dbPath)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&dex.Pokemon{},
		&dex.Move{},
		&dex.PokemonMove{},
		&dex.Evolution{},
	)
	if err != nil {
		return db, err
	}

	pokemon := make([]dex.Pokemon, 0, GEN1POKEMONCOUNT)
	for _, item := range apiData {
		pokemon = append(pokemon, dex.Pokemon{
			ID:             item.Id,
			Name:           item.Name,
			Type1:          item.Type1,
			Type2:          item.Type2,
			HP:             item.Hp,
			Attack:         item.Attack,
			Defense:        item.Defense,
			SpAttack:       item.SpecialAttack,
			SpDefense:      item.SpecialDefense,
			Speed:          item.Speed,
			BaseExperience: item.BaseExperience,
			GrowthRate:     item.GrowthRate,
			FrontSprite:    item.Sprites.front,
			BackSprite:     item.Sprites.back,
		})
	}

	tx := db.CreateInBatches(pokemon, len(pokemon))
	if tx.Error != nil {
		return db, tx.Error
	}
	// moves := []MoveData{}
	// for pokemon := range apiData {
	// 	Pokemon
	// }

	return db, nil
}
