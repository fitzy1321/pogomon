package setup

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const BASE_URL string = "https://pokeapi.co/api/v2"
const SPRITE_URL_BASE string = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-i/red-blue/transparent"

const POKEMON_COUNT int = 151

func GetSqliteDb(db_path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(db_path), &gorm.Config{})
}

func CreateSqliteDb(data []fullPokeData, path string) error {
	return nil
}

func FetchFromPokeAPI() []fullPokeData {
	pokeDataChan := make(chan fullPokeData, POKEMON_COUNT)
	var wg sync.WaitGroup
	for i := range POKEMON_COUNT {
		pokeId := uint(i + 1)
		pokemon_url := fmt.Sprintf("%s/pokemon/%d", BASE_URL, pokeId)

		wg.Add(1)
		go func(url string, pokeId uint) {
			defer wg.Done()

			pokemondata, err := fetchPokeAPI(url)
			if err != nil {
				// TODO: is this the best way to handle this error?
				fmt.Fprintln(os.Stderr, err)
				return
			}

			type_1, type_2 := getTypes(pokemondata)

			pokeDataChan <- fullPokeData{
				Id:              pokeId,
				Name:            pokemondata["name"].(string),
				Type_1:          type_1,
				Type_2:          type_2,
				Base_experience: uint(pokemondata["base_experience"].(float64)),
				Stats:           getStats(pokemondata),
			}

		}(pokemon_url, pokeId)
	}
	wg.Wait()
	close(pokeDataChan)

	fullAPIData := make([]fullPokeData, POKEMON_COUNT)
	for item := range pokeDataChan {
		fullAPIData = append(fullAPIData, item)
		fmt.Printf("Showing results I guess. %v\n", item)
	}
	return fullAPIData
}

type movesData struct {
}

type nextEvoData struct {
	Evolves_into_id uint
	Trigger         string
	Min_level       uint
	Item            *string // nullable
}

type fullPokeData struct {
	Id              uint
	Name            string
	Type_1          string
	Type_2          *string // nullable
	Base_experience uint
	Stats           map[string]int
	Moves           []movesData
	Next_evolutions []nextEvoData
	Growth_Rate     string
	Front_sprite    []byte
	Back_sprite     []byte
}

func getStats(data map[string]any) map[string]int {
	stats := make(map[string]int)
	for _, v := range data["stats"].([]any) {
		tm := v.(map[string]any)
		name := tm["stat"].(map[string]any)["name"].(string)
		baseStat := int(tm["base_stat"].(float64))

		// need to fix a couple strings
		switch name {
		case "special-attack":
			name = "special_attack"
		case "special-defense":
			name = "special_defense"
		}

		stats[name] = baseStat
	}
	return stats
}

func fetchPokeAPI(url string) (map[string]any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pokemondata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&pokemondata); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	return pokemondata, nil
}

//	func getSprites(pokeId int) ([]byte, []byte) {
//		front_resp, front_err := http.Get(fmt.Sprintf("%s/%d", SPRITE_URL_BASE, pokeId))
//		back_resp, back_err := http.Get(fmt.Sprintf("%s/back/%d", SPRITE_URL_BASE, pokeId))
//	}

func getTypes(data map[string]any) (string, *string) {
	var type_1 string
	var type_2 *string
	for _, t := range data["types"].([]any) {
		tm := t.(map[string]any)
		slot := int(tm["slot"].(float64))
		tmpVar := tm["type"].(map[string]any)["name"].(string)
		var name *string = nil
		if tmpVar != "" {
			name = &tmpVar
		}

		switch slot {
		case 1:
			// type_1 should always be there, so normal string works
			type_1 = *name
		case 2:
			// type_2 can be null, so this should be a pointer
			type_2 = name
		default:
			// pass
		}
	}
	return type_1, type_2
}

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
