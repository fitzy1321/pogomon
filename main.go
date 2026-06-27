package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"go-pokebattle/dex"
	"go-pokebattle/setup"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

func dbPathExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

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
	if exists, err := dbPathExists(dbPath); err != nil {
		printErrExit(fmt.Errorf("Error occured checking for sqlite file: %v\n", err))
	} else if !exists {
		// data := setup.FetchPokemonData()
		// fmt.Println("Length of pokemon data from api:", len(data))
		// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
		db, err = setup.FetchDataAndCreateDB(dbPath)
		if err != nil {
			printErrExit(fmt.Errorf("Something failed creating pokemon db: %+v\n", err))
		}
		// * Wait for terminal input
		fmt.Print("> ")
		fmt.Scanln()
	} else {
		db, err = setup.GetSqliteDb(dbPath)
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
