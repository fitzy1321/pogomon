package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"go-pokebattle/pokedata"
	"go-pokebattle/setup"
)

func dbPathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	// happy path, file exists no errors
	if err == nil {
		return true, nil
	}
	// file doesnot exist error, return false instead
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	// some other error, e.g. permission denied
	return false, err
}

func main() {
	// * Catch all panics!
	defer func() {
		r := recover()
		if r == nil {
			return
		}
	}()

	db_path := "pokedata.db"
	// exists, err := dbPathExists(db_path)
	// if !exists || err != nil {
	// 	// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
	// 	setup.FetchDataAndCreateSqliteDb(db_path)
	// }

	// * Fetch Data From PokeAPI, Create SQLite DB, seeded with API Data
	// setup.FetchDataAndCreateSqliteDb(db_path)
	data := setup.FetchPokemonData()
	fmt.Println("Length of pokemon data from api:", len(data))
	setup.CreateSqliteDb(data, db_path)

	// * Wait for terminal input
	fmt.Print("> ")
	fmt.Scanln()

	// * Get Gorm/Sqlite DB
	db, err := setup.GetSqliteDb(db_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to pokemon database: %v\n", err)
		return
	}

	// * Get all Pokemon from db
	var pokedex []pokedata.Pokemon
	result := db.Find(&pokedex)
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error getting pokemon data: %v\n", result.Error)
		return
	}

	// * Print Pokemons
	for _, k := range pokedex {
		fmt.Printf("Pokemon Id: %d Name: %s Type: %s\n", k.Id, k.Name, k.Type_1)
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
