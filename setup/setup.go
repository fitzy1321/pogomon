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
	dict        = map[string]any
	PokeApiData struct {
		ID             uint
		Name           string
		Type1          string
		Type2          *string // nullable
		BaseExperience *int    // nullable
		Moves          []MoveData
		NextEvolutions []NextEvoData
		GrowthRate     *string // nullable
		Sprites        Sprites
		PokemonStats
	}

	MoveData struct {
		ID            uint
		Name          string
		LevelLearned  int
		LearnMethod   *string
		MaxPP         int
		Power         *int         // nullable
		Accuracy      *int         // nullable
		Type          *string      // TODO: should this be nullable?
		DamageClass   *string      // nullable
		Ailment       *string      // nullable
		AilmentChance *int         // nullable
		MoveCategory  *string      // nullable
		Healing       *int         // nullable
		Drain         *int         // nullable
		StatChanges   []statChange // TODO: maybe nullable? wait, is this used?

	}

	PokemonStats struct {
		Attack    int
		Defense   int
		HP        int
		SpAttack  int
		SpDefense int
		Speed     int
	}

	statChange struct {
		Stat   string
		Change any // TODO: check type
	}

	NextEvoData struct {
		EvolvesIntoID   uint
		EvolvesIntoName *string
		Trigger         *string
		MinLevel        *int
		Item            *string // nullable
	}

	Sprites struct {
		Front, Back []byte
	}

	_mvIR struct {
		name   string
		level  int
		url    string
		method string
	}
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

	fullApiData := make([]PokeApiData, 0, GEN1POKEMONCOUNT)
	var errs []error
	for r := range dataCh {
		if r.IsOk() {
			fullApiData = append(fullApiData, *r.Value)
			// fmt.Printf("Pokemon #%d, %s, %+v, \n", r.Value.ID, r.Value.Name, r.Value.NextEvolutions)
			// fmt.Printf("%+v\n", r)
		} else {
			errs = append(errs, r.Error)
		}
	}

	return fullApiData, errs
}
