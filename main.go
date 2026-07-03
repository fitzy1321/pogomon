package main

import (
	"fmt"
	"os"

	"go-pokebattle/dex"
	"go-pokebattle/setup"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

func printErrExit(err error) {
	fmt.Fprintf(os.Stderr, "%+v", err)
	os.Exit(1)
}

func main() {
	// // * Catch all panics!
	// defer func() {
	// 	r := recover()
	// 	if r == nil {
	// 		return
	// 	}
	// }()

	// * Get and or Create Gorm/Sqlite DB
	dbPath := "pokedata.db"
	var db *gorm.DB = nil
	var err error = nil

	if !setup.FileExists(dbPath) {
		// data := setup.FetchPokemonData()
		// fmt.Println("Length of pokemon data from api:", len(data))
		// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
		data, errs := setup.FetchPokemonData()
		if errs != nil || len(errs) != 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "%+v", fmt.Errorf("Something failed creating pokemon db: %+v\n", e))
			}
			os.Exit(1)
		}
		if err := setup.SaveGobFile(data, setup.CACHEFILE); err != nil {
			printErrExit(err)
		}
		db, err = setup.CreateAndSeedDB(data, dbPath)
		if err != nil {
			printErrExit(err)
		}
		// db, errs = setup.FetchDataAndCreateDB(dbPath)
		// if errs != nil || len(errs) != 0 {
		// 	for _, e := range errs {
		// 		fmt.Fprintf(os.Stderr, "%+v", fmt.Errorf("Something failed creating pokemon db: %+v\n", e))
		// 	}
		// 	os.Exit(1)
		// }
		// * Wait for terminal input
		fmt.Print("> ")
		fmt.Scanln()
	} else {
		db, err = setup.GetGormSqliteDB(dbPath)
		if err != nil {
			printErrExit(fmt.Errorf("Something failed connecting to pokemon db: %v\n", err))
		}
	}

	// * Get all Pokemon from db
	var pokedex []dex.Pokemon
	result := db.Find(&pokedex)
	if result.Error != nil {
		printErrExit(fmt.Errorf("Error getting pokemon data: %v\n", result.Error))
	}

	// * Print Pokemons
	for _, k := range pokedex {
		var type2 string = "<nil>"
		if k.Type2 != nil {
			type2 = *k.Type2
		}
		titleName := cases.Title(language.Und, cases.NoLower).String(k.Name)
		fmt.Printf("Pokemon #%d, %s.  types: %s %s\n", k.ID, titleName, k.Type1, type2)
	}

	// // * Get all moves from db
	// var movedex []Move
	// result = db.Find(&movedex)
	// if result.Error != nil {
	// 	fmt.Fprintf(os.Stderr, "Error getting move data: %v\n", result.Error)
	// 	return
	// }

	// // * Print moves
	// for _, k := range movedex {
	// 	fmt.Printf("Move id: %d, Name: %s\n", k.Id, k.Name)
	// }
}
