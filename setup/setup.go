package setup

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"go-pokebattle/dex"
	. "go-pokebattle/result"

	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/driver/sqlite"
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
		NextEvolutions []nextEvoData
		GrowthRate     *string // nullable
		Sprites        sprites
		stats
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

	stats struct {
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

	nextEvoData struct {
		EvolvesIntoId uint
		Trigger       string
		MinLevel      uint
		Item          *string // nullable
	}

	sprites struct {
		front, back []byte
	}

	_mvIR struct {
		name   string
		level  int
		url    string
		method string
	}
)

type (
	dict   = map[string]any
	fnOpts struct{ LoadFromCacheFile, SaveToCacheFile bool }
)

// Call PokeAPI and etl into Sqlite tables
func FetchDataAndCreateDB(dbPath string, options fnOpts) (*gorm.DB, []error) {
	var data []PokeApiData
	if options.LoadFromCacheFile {
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
		data, errs = FetchPokemonData()
		if errs != nil || len(errs) > 0 {
			return nil, errs
		}
		if len(data) == 0 {
			return nil, []error{errors.New("Failed to fetch data from PokeAPI")}
		}
		if options.SaveToCacheFile {
			err := SaveGobFile(data, CACHEFILE)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error occurred saving cache gob file: %+v", err)
			}
		}
	}

	res, err := CreateAndSeedDB(data, dbPath)
	if err != nil {
		return res, []error{err}
	}
	return res, nil
}

func FetchPokemonData() ([]PokeApiData, []error) {
	fullApiData := make([]PokeApiData, 0, GEN1POKEMONCOUNT)
	dataCh := make(chan Result[PokeApiData], GEN1POKEMONCOUNT)
	sema := make(chan struct{}, 20) // to cap # goroutines running

	wg := sync.WaitGroup{}
	for i := range GEN1POKEMONCOUNT {
		pokeId := uint(i + 1)

		wg.Go(func() {
			sema <- struct{}{}        // add
			defer func() { <-sema }() // done / release
			dataCh <- rawApiDataToStructs(pokeId)
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

// Open a connection to sqlite and initalize gorm
func GetGormSqliteDB(dbPath string) (*gorm.DB, error) {
	if !strings.Contains(dbPath, FOREIGNKEYSTR) {
		dbPath = fmt.Sprintf("%s%s", dbPath, FOREIGNKEYSTR)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return db, err
	}

	if res := db.Exec("PRAGMA foreign_keys = ON", nil); res.Error != nil {
		return nil, res.Error
	}

	return db, nil
}

// This will take data from the api and try to create and insert sqlite tables and data
func CreateAndSeedDB(apiData []PokeApiData, dbPath string) (*gorm.DB, error) {
	fmt.Println("Initializing db @filepath:", dbPath)

	db, err := GetGormSqliteDB(dbPath)
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&dex.Pokemon{},
		&dex.Move{},
		&dex.PokemonMove{},
		&dex.Evolution{},
	)
	if err != nil {
		return db, err
	}

	pokemon := make([]dex.Pokemon, 0, GEN1POKEMONCOUNT)
	var moves []dex.Move
	var pokemonMoves []dex.PokemonMove
	moveIdSet := mapset.NewSet[uint]()

	for _, pitem := range apiData {
		pokemon = append(pokemon, dex.Pokemon{
			ID:             pitem.Id,
			Name:           pitem.Name,
			Type1:          pitem.Type1,
			Type2:          pitem.Type2,
			HP:             pitem.HP,
			Attack:         pitem.Attack,
			Defense:        pitem.Defense,
			SpAttack:       pitem.SpAttack,
			SpDefense:      pitem.SpDefense,
			Speed:          pitem.Speed,
			BaseExperience: pitem.BaseExperience,
			GrowthRate:     pitem.GrowthRate,
			FrontSprite:    pitem.Sprites.front,
			BackSprite:     pitem.Sprites.back,
		})
		for _, mitem := range pitem.Moves {
			if !moveIdSet.Contains(mitem.Id) {
				moveIdSet.Add(mitem.Id)
				moves = append(moves, dex.Move{
					ID:            mitem.Id,
					Name:          mitem.Name,
					Power:         mitem.Power,
					Accuracy:      mitem.Accuracy,
					MaxPP:         mitem.MaxPP,
					Type:          mitem.Type,
					DamageClass:   mitem.DamageClass,
					Ailment:       mitem.Ailment,
					AilmentChance: mitem.AilmentChance,
					Category:      mitem.MoveCategory,
					Healing:       mitem.Healing,
					Drain:         mitem.Drain,
				})
			}
			pokemonMoves = append(pokemonMoves, dex.PokemonMove{
				PokemonID:    pitem.Id,
				MoveID:       mitem.Id,
				LevelLearned: mitem.LevelLearned,
				LearnMethod:  mitem.LearnMethod,
			})
		}
	}

	tx := db.CreateInBatches(pokemon, len(pokemon))
	if tx.Error != nil {
		return db, tx.Error
	}
	tx = db.CreateInBatches(moves, len(moves))
	if tx.Error != nil {
		return db, tx.Error
	}
	tx = db.CreateInBatches(pokemonMoves, len(moves))
	if tx.Error != nil {
		return db, tx.Error
	}
	// TODO evolutions

	return db, nil
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
