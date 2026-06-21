package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"

	// "runtime"

	"go-pokebattle/setup"
)

// func osLevelStuff() error {
// 	home_path, ok := os.LookupEnv("HOME")
// 	if !ok {
// 		return fmt.Errorf("No Home ENV, something is wrong ...\n")

// 	}
// 	fmt.Println("Home path:", home_path)

// 	xdg_data := os.Getenv("XDG_DATA_HOME")
// 	fmt.Println("idk if this is real? :", xdg_data)

// 	xdg_config := os.Getenv("XDG_CONFIG_HOME")
// 	fmt.Println("XDG_CONFIG_HOME:", xdg_config)

// 	osname := runtime.GOOS
// 	switch osname {
// 	case "windows":
// 		fmt.Println("Windows specific stuff")
// 	case "darwin":
// 		fmt.Println("MacOS stuff")
// 	case "linux":
// 		fmt.Println("linux stuff")
// 	default:
// 		fmt.Println("I have no idea what you're on ...")
// 	}

// 	return nil
// }

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
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err // some other error, e.g. permission denied
}

func main() {
	setup.DataSeeding()
	db_path := "pokedata.db"
	exists, err := dbPathExists(db_path)
	if !exists || err != nil {
		// call api and setup sqlitedb
		setup.DataSeeding()
	}

	buf := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	_, err = buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	}

	// open gorm and sqlite db
	db, err := setup.SqliteDb(db_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to pokemon database: %v\n", err)
		return
	}

	// get all pokemon
	var pokedex []Pokemon
	result := db.Find(&pokedex)
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error getting pokemon data: %v\n", result.Error)
		return
	}

	// print pokemon
	for _, k := range pokedex {
		fmt.Printf("Pokemon Id: %d Name: %s Type: %s\n", k.Id, k.Name, k.Type_1)
	}

	// // get al moves
	// var movedex []Move
	// result = db.Find(&movedex)
	// if result.Error != nil {
	// 	fmt.Fprintf(os.Stderr, "Error getting move data: %v\n", result.Error)
	// 	return
	// }

	// // print all moves
	// for _, k := range movedex {
	// 	fmt.Printf("Move id: %d, Name: %s\n", k.Id, k.Name)
	// }
}
