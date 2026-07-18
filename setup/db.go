package setup

import (
	"fmt"
	"strings"

	. "pogomon/sqlmodels"

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

	gdb, err := GetGormSqliteDB(dbPath)
	if err != nil {
		return nil, err
	} else if gdb.Error != nil {
		return gdb, gdb.Error
	}

	err = gdb.AutoMigrate(
		&Pokemon{},
		&Move{},
		&PokemonMove{},
		&Evolution{},
		&UserSave{},
		&PartyPokemon{},
		&PartyPokemonMove{},
	)

	if err != nil {
		return gdb, err
	}

	pokemon := make([]Pokemon, 0, GEN1POKEMONCOUNT)
	var moves []Move
	var pokemonMoves []PokemonMove
	moveIdSet := make(map[uint]any)
	var evolutions []Evolution

	for _, pitem := range apiData {
		pokemon = append(pokemon, Pokemon{
			ID:             pitem.ID,
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
			// NOTE: Deduplicate PokemonMoves -> Moves slice
			if _, exists := moveIdSet[mitem.ID]; !exists {
				moveIdSet[mitem.ID] = struct{}{}
				moves = append(moves, Move{
					ID:            mitem.ID,
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
			pokemonMoves = append(pokemonMoves, PokemonMove{
				PokemonID:    pitem.ID,
				MoveID:       mitem.ID,
				LevelLearned: mitem.LevelLearned,
				LearnMethod:  mitem.LearnMethod,
			})
		}

		for _, evoRaw := range pitem.NextEvolutions {
			evolutions = append(evolutions, Evolution{
				PokemonID:       pitem.ID,
				EvolvesIntoID:   evoRaw.EvolvesIntoID,
				EvolvesIntoName: evoRaw.EvolvesIntoName,
				Trigger:         evoRaw.Trigger,
				MinLevel:        evoRaw.MinLevel,
				Item:            evoRaw.Item,
				IsPlayerChoice:  len(pitem.NextEvolutions) > 1,
			})
		}
	}

	tx := gdb.CreateInBatches(pokemon, len(pokemon))
	if tx.Error != nil {
		return gdb, tx.Error
	}
	tx = gdb.CreateInBatches(moves, len(moves))
	if tx.Error != nil {
		return gdb, tx.Error
	}
	tx = gdb.CreateInBatches(pokemonMoves, len(moves))
	if tx.Error != nil {
		return gdb, tx.Error
	}
	tx = gdb.CreateInBatches(evolutions, len(evolutions))
	if tx.Error != nil {
		return gdb, tx.Error
	}

	return gdb, nil
}
