package setup

import (
	"fmt"
	"strings"

	"go-pokebattle/dex"

	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open a connection to sqlite and initalize gorm
func GetGormSqliteDB(dbPath string) (*gorm.DB, error) {
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
func CreateAndSeedDB(apiData []PokeApiData, dbPath string) (*gorm.DB, error) {
	fmt.Println("Initializing db @filepath:", dbPath)

	db, err := GetGormSqliteDB(dbPath)
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
	var moves []dex.Move
	var pokemonMoves []dex.PokemonMove
	moveIdSet := mapset.NewSet[uint]()

	for _, pitem := range apiData {
		pokemon = append(pokemon, dex.Pokemon{
			ID:             pitem.Id,
			Name:           pitem.Name,
			Type1:          pitem.Type1,
			Type2:          pitem.Type2,
			HP:             pitem.HP,
			Attack:         pitem.Attack,
			Defense:        pitem.Defense,
			SpAttack:       pitem.SpAttack,
			SpDefense:      pitem.SpDefense,
			Speed:          pitem.Speed,
			BaseExperience: pitem.BaseExperience,
			GrowthRate:     pitem.GrowthRate,
			FrontSprite:    pitem.Sprites.Front,
			BackSprite:     pitem.Sprites.Back,
		})
		for _, mitem := range pitem.Moves {
			if !moveIdSet.Contains(mitem.Id) {
				moveIdSet.Add(mitem.Id)
				moves = append(moves, dex.Move{
					ID:            mitem.Id,
					Name:          mitem.Name,
					Power:         mitem.Power,
					Accuracy:      mitem.Accuracy,
					MaxPP:         mitem.MaxPP,
					Type:          mitem.Type,
					DamageClass:   mitem.DamageClass,
					Ailment:       mitem.Ailment,
					AilmentChance: mitem.AilmentChance,
					Category:      mitem.MoveCategory,
					Healing:       mitem.Healing,
					Drain:         mitem.Drain,
				})
			}
			pokemonMoves = append(pokemonMoves, dex.PokemonMove{
				PokemonID:    pitem.Id,
				MoveID:       mitem.Id,
				LevelLearned: mitem.LevelLearned,
				LearnMethod:  mitem.LearnMethod,
			})
		}
	}

	tx := db.CreateInBatches(pokemon, len(pokemon))
	if tx.Error != nil {
		return db, tx.Error
	}
	tx = db.CreateInBatches(moves, len(moves))
	if tx.Error != nil {
		return db, tx.Error
	}
	tx = db.CreateInBatches(pokemonMoves, len(moves))
	if tx.Error != nil {
		return db, tx.Error
	}
	// TODO evolutions

	return db, nil
}
