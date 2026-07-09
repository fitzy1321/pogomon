package setup

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	. "go-pokebattle/result"

	"gorm.io/gorm"
)

const (
	BASEURL             string = "https://pokeapi.co/api/v2"
	CACHEFILE           string = "POKEDATA_CACHE.gob"
	FOREIGNKEYSTR       string = "?_foreign_keys=on"
	GEN1POKEMONCOUNT    int    = 151
	SPRITEURLBASE       string = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-i/red-blue/transparent"
	TradeEvolutionLevel int    = 32
)

type (
	dict = map[string]any
)

// Call PokeAPI and etl into Sqlite tables
func FetchDataAndCreateDB(dbPath string, client *http.Client) (*gorm.DB, []error) {
	var data []PokeApiData
	if FileExists(CACHEFILE) {
		var err error
		data, err = LoadGobFile(CACHEFILE)
		if err != nil {
			return nil, []error{err}
		}
	} else {
		var errs []error
		if client == nil {
			client = http.DefaultClient
		}
		data, errs = FetchPokemonData(client)
		if errs != nil || len(errs) > 0 {
			return nil, errs
		}
		if len(data) == 0 {
			return nil, []error{errors.New("Failed to fetch data from PokeAPI")}
		}
		// if fOpts.SaveToCacheFile {
		err := SaveGobFile(data, CACHEFILE)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occurred saving cache gob file: %+v\n", err)
		}
		// }
	}

	res, err := CreateAndSeedDB(data, dbPath)
	if err != nil {
		return res, []error{err}
	}
	return res, nil
}

func FetchPokemonData(client *http.Client) ([]PokeApiData, []error) {
	fullApiData := make([]PokeApiData, 0, GEN1POKEMONCOUNT)

	dataCh := make(chan Result[*PokeApiData], GEN1POKEMONCOUNT)
	sema := make(chan struct{}, 20) // to cap # goroutines running

	wg := sync.WaitGroup{}
	for i := range GEN1POKEMONCOUNT {
		pokeId := uint(i + 1)

		wg.Go(func() {
			sema <- struct{}{}        // add
			defer func() { <-sema }() // done / release
			dataCh <- topLevelPokemonData(client, pokeId)
		})
	}
	wg.Wait()
	close(dataCh)
	var errs []error
	for r := range dataCh {
		if r.IsOk() {
			fullApiData = append(fullApiData, *r.Value)
			fmt.Printf("Pokemon #%d, %s, %+v, \n", r.Value.ID, r.Value.Name, r.Value.NextEvolutions)
			// fmt.Printf("%+v\n", r)
		} else {
			errs = append(errs, r.Error)
		}
	}

	return fullApiData, errs
}
