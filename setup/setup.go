package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	"gorm.io/gorm"
)

const (
	BaseUrl          string = "https://pokeapi.co/api/v2"
	Gen1PokemonCount int    = 151
	SpriteUrlBase    string = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-i/red-blue/transparent"
)

type dict = map[string]any

func GetSqliteDb(db_path string) (*gorm.DB, error) {
	return internalGormDbSetup(db_path)
}

func CreateSqliteDb(data []fullPokeData, path string) error {
	return createSqliteDb(data, path)
}

// func FetchDataAndCreateSqliteDb(db_path string) error {
// 	return CreateSqliteDb(FetchPokemonData(), db_path)
// }

func FetchPokemonData() []fullPokeData {
	// WARN: buffered channel, don't change unless you know what you're doing (more than me 🙃).
	// WARN: concurrency gremlins will appear
	pokeDataChan := make(chan fullPokeData, Gen1PokemonCount)
	var wg sync.WaitGroup
	for i := range Gen1PokemonCount {
		pokeId := uint(i + 1)
		pokemonUrl := fmt.Sprintf("%s/pokemon/%d", BaseUrl, pokeId)

		// * Where the ✨Magic✨ happens
		wg.Add(1)
		go func(url string, pokeId uint) {
			defer wg.Done()

			// * Top level PokeAPI pokemon object request
			pokemonData, err := fetchPokeAPIData(url)
			if err != nil {
				// TODO: is this the best way to handle this error?
				fmt.Fprintln(os.Stderr, err)
				return
			}

			// * Get Pokemon Type data, no additional requests
			type1, type2 := getPokemonTypes(pokemonData)

			stats := getStats(pokemonData)
			if stats == nil {
				stats = &statsData{}
			}

			// * Get moves, will perform additional network requests for detailed move data...
			mvData, err := getMovesData(pokemonData)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			// * Get Sprites PNGs (2 network requests, front and back PNGs)
			frontSprite, backSprite, err := getSprites(pokeId)

			// * Species Data PokeAPI request, network request
			var growthRate *string = nil
			if speciesUrl, ok := pokemonData["species"].(dict)["url"].(string); ok {
				speciesData, spErr := fetchPokeAPIData(speciesUrl)
				if spErr != nil {
					fmt.Fprintln(os.Stderr, spErr)
					return
				}
				grstr := speciesData["growth_rate"].(dict)["name"].(string)
				growthRate = &grstr
				// TODO: Evolution Data
			}

			pokeDataChan <- fullPokeData{
				Id:             pokeId,
				Name:           pokemonData["name"].(string),
				Type1:          type1,
				Type2:          type2,
				BaseExperience: int(pokemonData["base_experience"].(float64)),
				Stats:          *stats,
				Moves:          mvData,
				NextEvolutions: []nextEvoData{}, // TODO: fix later
				GrowthRate:     growthRate,
				FrontSprite:    frontSprite,
				BackSprite:     backSprite,
			}
			// end go func
		}(pokemonUrl, pokeId)
		// end forloop
	}

	// * Wait for all goroutines and close the channel
	wg.Wait()
	// this may not get called if the buffered channel is changed, btw
	close(pokeDataChan)

	// * Get data out of channel
	fullAPIData := make([]fullPokeData, 0, len(pokeDataChan))
	for item := range pokeDataChan {
		fullAPIData = append(fullAPIData, item)
		fmt.Printf("Showing results I guess. %+v\n", item)
	}
	return fullAPIData
}

func fetchPokeAPIData(url string) (dict, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data dict
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	return data, nil
}

func getSprites(pokeId uint) (ftSprite []byte, bkSprite []byte, err error) {
	err = nil
	frontUrl := fmt.Sprintf("%s/%d.png", SpriteUrlBase, pokeId)
	backUrl := fmt.Sprintf("%s/back/%d.png", SpriteUrlBase, pokeId)

	ftResp, ftRspErr := http.Get(frontUrl)
	if ftRspErr != nil {
		err = ftRspErr
	}
	defer ftResp.Body.Close()

	ftSprite, ftErr := spriteHandler(ftResp)
	if ftErr != nil {
		err = ftErr
	}

	bkResp, bkRspErr := http.Get(backUrl)
	if bkRspErr != nil {
		err = bkRspErr
	}
	defer bkResp.Body.Close()

	bkSprite, bkErr := spriteHandler(bkResp)
	if bkErr != nil {
		err = bkErr
	}
	return
}

func spriteHandler(resp *http.Response) ([]byte, error) {
	if resp.Header.Get("Content-Type") == "image/png" {
		return io.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("Wrong Content-Type from network response.%v", resp.Header.Get("Content-Type"))
}

func getMovesData(pokeData dict) ([]moveData, error) {
	var rbMoves []_mvIR
	names := mapset.NewSet[string]()

	pokeMoves, ok := pokeData["moves"].([]any)
	if !ok {
		return nil, fmt.Errorf("No move data ...")
	}

	for _, pmMv := range pokeMoves {
		md := pmMv.(dict)
		vgdTop, ok := md["version_group_details"].([]any)
		if !ok {
			// TODO: error handle, idk man ...
		}
		for _, vgdIR := range vgdTop {
			vgd := vgdIR.(dict)
			if vgd["version_group"].(dict)["name"].(string) == "red-blue" {
				moveName := md["move"].(dict)["name"].(string)
				if !names.Contains(moveName) {
					names.Add(moveName)
					rbMoves = append(rbMoves, _mvIR{
						name:   moveName,
						level:  int(vgd["level_learned_at"].(float64)),
						url:    md["move"].(dict)["url"].(string),
						method: vgd["move_learn_method"].(dict)["name"].(string),
					})
				}
			}
		} // end for
	} // end for

	var detailed []moveData
	for _, move := range rbMoves {
		mvData, err := fetchPokeAPIData(move.url)
		if err != nil {
			// TODO: error handle, idk man ..
		}
		meta, ok := mvData["meta"].(dict)
		if !ok {
			// TODO: error handle idk man ...
		}

		// TODO: implement []statChange data
		// statChanges := []statChange{}

		var power *int = nil
		if tp, ok := mvData["power"].(int); ok {
			power = &tp
		}

		var acc *int = nil
		if tacc, ok := mvData["accuracy"].(int); ok {
			acc = &tacc
		}

		var mpp int = 0
		if tmpp, ok := mvData["pp"].(int); ok {
			mpp = tmpp
		}

		var mtype *string = nil
		if tmtype, ok := mvData["type"].(dict)["name"].(string); ok {
			mtype = &tmtype
		}

		var dc *string = nil
		if tdc, ok := mvData["damage_class"].(dict)["name"].(string); ok {
			dc = &tdc
		}
		var ailment *string = nil
		if tailment, ok := meta["ailment"].(dict)["name"].(string); ok {
			ailment = &tailment
		}

		var ailmentChance *int = nil
		if tailChnc, ok := meta["ailment_chance"].(int); ok {
			ailmentChance = &tailChnc
		}

		detailed = append(detailed, moveData{
			Name:          move.name,
			LevelLearned:  uint(move.level),
			LearnMethod:   move.method,
			MaxPp:         mpp,
			Power:         power,
			Accuracy:      acc,
			Type:          mtype,
			DamageClass:   dc,
			Ailment:       ailment,
			AilmentChance: ailmentChance,
			// Move_category:  meta["category"].(dict)["name"].(string),
			// Healing:        meta["healing"].(int),
			// Drain:          meta["drain"].(int),
			StatChanges: []statChange{}, // TODO: fix later
		})
	}
	return detailed, nil
}

func getPokemonTypes(data dict) (string, *string) {
	var type1 string
	var type2 *string
	for _, t := range data["types"].([]any) {
		tm := t.(dict)
		slot := int(tm["slot"].(float64))
		tmpVar := tm["type"].(dict)["name"].(string)
		var name *string = nil
		if tmpVar != "" {
			name = &tmpVar
		}

		switch slot {
		case 1:
			// type_1 should always be there, so normal string works
			type1 = *name
		case 2:
			// type_2 can be null, so this should be a pointer
			type2 = name
		default:
			// TODO: ? pass ?
		}
	}
	return type1, type2
}

func getStats(data dict) *statsData {
	stats := make(map[string]int)
	for _, v := range data["stats"].([]any) {
		tm := v.(dict)
		name := tm["stat"].(dict)["name"].(string)
		baseStat := int(tm["base_stat"].(float64))
		stats[name] = baseStat
	}
	return &statsData{
		Attack:         stats["attack"],
		Defense:        stats["defense"],
		Hp:             stats["hp"],
		SpecialAttack:  stats["special-attack"],
		SpecialDefense: stats["special-defense"],
		Speed:          stats["speed"],
	}
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
