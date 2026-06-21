package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"go-pokebattle/setup"
)

// ! gorm struct, do not change the member names
type Pokemon struct {
	Id              uint `gorm:"primaryKey"`
	Name            string
	Type_1          string
	Type_2          *string
	Base_hp         uint
	Base_attack     uint
	Base_defense    uint
	Base_sp_attack  uint
	Base_sp_defense uint
	Base_speed      uint
	Base_experience *uint
	Growth_rate     *string
	Front_sprite    []byte
	Back_sprite     []byte
}

func (Pokemon) TableName() string {
	return "dex_pokemon"
}

// ! gorm struct, do not change the member names
type Move struct {
	Id             uint `gorm:"primaryKey"`
	Name           string
	Power          *uint
	Accuracy       *uint
	Max_pp         uint
	Type           *string
	Damage_class   *string
	Ailment        *string
	Ailment_chance *uint
	Move_category  *string
	Healing        *uint
	Drain          *int
}

func (Move) TableName() string {
	return "dex_move"
}

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
	db_path := "pokedata.db"
	// exists, err := dbPathExists(db_path)
	// if !exists || err != nil {
	// 	// call api and setup sqlitedb
	// 	apiPokeData := setup.FetchFromPokeAPI()
	// 	setup.CreateSqliteDb(apiPokeData)
	// }

	// * Fetch Data From PokeAPI * //
	apiPokeData := setup.FetchFromPokeAPI()

	// * Create SQLite DB, seed with API Data * //
	setup.CreateSqliteDb(apiPokeData, db_path)

	// * Wait for terminal input * //
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	_, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	}

	// * Get Gorm/Sqlite DB * //
	db, err := setup.GetSqliteDb(db_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to pokemon database: %v\n", err)
		return
	}

	// * Get all Pokemon * //
	var pokedex []Pokemon
	result := db.Find(&pokedex)
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error getting pokemon data: %v\n", result.Error)
		return
	}

	// * print some Pokemon data, coming from sqlite * //
	for _, k := range pokedex {
		fmt.Printf("Pokemon Id: %d Name: %s Type: %s\n", k.Id, k.Name, k.Type_1)
	}

	// // * get al moves * //
	// var movedex []Move
	// result = db.Find(&movedex)
	// if result.Error != nil {
	// 	fmt.Fprintf(os.Stderr, "Error getting move data: %v\n", result.Error)
	// 	return
	// }

	// // * print all moves * //
	// for _, k := range movedex {
	// 	fmt.Printf("Move id: %d, Name: %s\n", k.Id, k.Name)
	// }
}
