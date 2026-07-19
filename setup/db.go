package setup

import (
	"errors"
	"fmt"
	"strings"

	"pogomon/consts"
	"pogomon/store"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const FOREIGNKEYSTR string = "?_foreign_keys=on"

// Open a connection to sqlite and initalize gorm
func GetGormSqliteDB(dbPath string) (*gorm.DB, error) {
	if !strings.Contains(dbPath, FOREIGNKEYSTR) {
		dbPath = fmt.Sprintf("%s%s", dbPath, FOREIGNKEYSTR)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return db, err
	}

	var enabled int
	if db.Raw("PRAGMA foreign_keys").Scan(&enabled); enabled != 1 {
		return nil, errors.New("Sqlite Foreign keys not enabled ...")
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
		&store.Pokemon{},
		&store.Move{},
		&store.PokemonMove{},
		&store.Evolution{},
		&store.UserSave{},
		&store.PartyPokemon{},
		&store.PartyPokemonMove{},
	)

	if err != nil {
		return gdb, err
	}

	pokemon := make([]store.Pokemon, 0, consts.GEN1POKEMONCOUNT)
	var moves []store.Move
	var pokemonMoves []store.PokemonMove
	moveIdSet := make(map[uint]any)
	var evolutions []store.Evolution

	for _, pitem := range apiData {
		pokemon = append(pokemon, store.Pokemon{
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
				moves = append(moves, store.Move{
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
			pokemonMoves = append(pokemonMoves, store.PokemonMove{
				PokemonID:    pitem.ID,
				MoveID:       mitem.ID,
				LevelLearned: mitem.LevelLearned,
				LearnMethod:  mitem.LearnMethod,
			})
		}

		for _, evoRaw := range pitem.NextEvolutions {
			evolutions = append(evolutions, store.Evolution{
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
