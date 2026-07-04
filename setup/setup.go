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
	BASEURL          string = "https://pokeapi.co/api/v2"
	CACHEFILE        string = "POKEDATA_CACHE.gob"
	FOREIGNKEYSTR    string = "?_foreign_keys=on"
	GEN1POKEMONCOUNT int    = 151
	SPRITEURLBASE    string = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-i/red-blue/transparent"
)

type (
	PokeApiData struct {
		Id             uint
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
		Id            uint
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
		EvolvesIntoId uint
		Trigger       string
		MinLevel      uint
		Item          *string // nullable
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

type (
	dict         = map[string]any
	FetchOptions struct {
		LoadFromCacheFile bool
		SaveToCacheFile   bool
		Client            *http.Client
	}
)

// Call PokeAPI and etl into Sqlite tables
func FetchDataAndCreateDB(dbPath string, fOpts FetchOptions) (*gorm.DB, []error) {
	var data []PokeApiData
	if fOpts.LoadFromCacheFile {
		if FileExists(CACHEFILE) {
			var err error
			data, err = LoadGobFile(CACHEFILE)
			if err != nil {
				//TODO
			}
		} else {
			return nil, []error{errors.New("Load Cache File option was passed in, but cachefile doesnot exist!")}
		}
	} else {
		var errs []error
		data, errs = FetchPokemonData(fOpts.Client)
		if errs != nil || len(errs) > 0 {
			return nil, errs
		}
		if len(data) == 0 {
			return nil, []error{errors.New("Failed to fetch data from PokeAPI")}
		}
		if fOpts.SaveToCacheFile {
			err := SaveGobFile(data, CACHEFILE)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error occurred saving cache gob file: %+v\n", err)
			}
		}
	}

	res, err := CreateAndSeedDB(data, dbPath)
	if err != nil {
		return res, []error{err}
	}
	return res, nil
}

func FetchPokemonData(client *http.Client) ([]PokeApiData, []error) {
	if client == nil {
		client = http.DefaultClient
	}

	fullApiData := make([]PokeApiData, 0, GEN1POKEMONCOUNT)
	dataCh := make(chan Result[PokeApiData], GEN1POKEMONCOUNT)
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
			fullApiData = append(fullApiData, r.Value)
			fmt.Printf("Pokemon #%d, %s\n", r.Value.Id, r.Value.Name)
			// fmt.Printf("%+v\n", r)
		} else {
			errs = append(errs, r.Error)
		}
	}

	return fullApiData, errs
}

// TODO: Keep or delete this func?
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
