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
const POKEMON_COUNT int = 151

func SqliteDb(db_path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(db_path), &gorm.Config{})
}

func DataSeeding() {
	var wg sync.WaitGroup
	var pokeId int
	var pokemon_url string
	for i := range POKEMON_COUNT {
		pokeId = i + 1
		pokemon_url = fmt.Sprintf("%s/pokemon/%d", BASE_URL, pokeId)
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := http.Get(url)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			defer resp.Body.Close()

			var data map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			fmt.Println(data["name"])

		}(pokemon_url)
	}
	wg.Wait()
}
