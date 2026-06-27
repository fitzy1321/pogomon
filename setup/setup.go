package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	. "go-pokebattle/result"

	mapset "github.com/deckarep/golang-set/v2"
)

const (
	BASEURL          string = "https://pokeapi.co/api/v2"
	GEN1POKEMONCOUNT int    = 151
	SPRITEURLBASE    string = "https://raw.githubusercontent.com/PokeAPI/sprites/master/sprites/pokemon/versions/generation-i/red-blue/transparent"
)

type (
	dict = map[string]any
)

func FetchPokemonData() []FullPokeData {
	fullApiData := make([]FullPokeData, 0, GEN1POKEMONCOUNT)
	chPokeData := make(chan Result[FullPokeData], GEN1POKEMONCOUNT)

	// use this to cap amount of goroutines running at once
	semafore := make(chan struct{}, 20)
	for i := range GEN1POKEMONCOUNT {
		pokeId := uint(i + 1)
		url := fmt.Sprintf("%s/pokemon/%d", BASEURL, pokeId)

		go func() {
			semafore <- struct{}{}        // "acquire lock"
			defer func() { <-semafore }() // "release lock once complete"
			chPokeData <- rawApiDataToStructs(url, pokeId)
		}()
	}

	for range GEN1POKEMONCOUNT {
		r := <-chPokeData
		if r.IsOk() {
			fullApiData = append(fullApiData, r.Value)
			fmt.Printf("Pokemon #%d, %s\n", r.Value.Id, r.Value.Name)
			// fmt.Printf("%+v\n", r)
		} else {
			fmt.Fprintf(os.Stderr, "Error from goroutines: %+v", r.GetError())
		}
	}
	close(chPokeData)

	return fullApiData
}

func rawApiDataToStructs(url string, pokeId uint) Result[FullPokeData] {
	pokemonData, err := fetchPokeAPIData(url)
	if err != nil {
		return Err[FullPokeData](err)
	}

	type1, type2, err := getPokemonTypes(pokemonData)
	if err != nil {
		return Err[FullPokeData](err)
	}

	stats := getStats(pokemonData)
	if stats == nil {
		stats = &StatsData{}
	}

	chMvData := make(chan Result[[]MoveData], 1)
	chSprites := make(chan Result[Sprites], 1)
	chGrowthRate := make(chan Result[*string], 1)

	go func() {
		chMvData <- getMovesData(pokemonData)
	}()

	go func() {
		chSprites <- getSprites(pokeId)
	}()

	go func() {
		if speciesUrl, ok := pokemonData["species"].(dict)["url"].(string); ok {
			speciesData, err := fetchPokeAPIData(speciesUrl)
			if err != nil {
				chGrowthRate <- Err[*string](err)
				return
			}
			grstr, ok := speciesData["growth_rate"].(dict)["name"].(string)
			if !ok {
				chGrowthRate <- ErrFromStr[*string](fmt.Sprintf("Error getting growth_rate from species data for #%d", pokeId))
				return
			}
			chGrowthRate <- Ok(&grstr)
			// TODO: Evolution Data
		}
	}()

	mvDataRes := <-chMvData
	if mvDataRes.IsErr() {
		return Err[FullPokeData](mvDataRes.GetError())
	}
	spriteRes := <-chSprites
	if spriteRes.IsErr() {
		return Err[FullPokeData](spriteRes.GetError())
	}
	grRes := <-chGrowthRate
	if grRes.IsErr() {
		return Err[FullPokeData](grRes.GetError())
	}

	return Ok(FullPokeData{
		Id:             pokeId,
		Name:           pokemonData["name"].(string),
		Type1:          type1,
		Type2:          type2,
		BaseExperience: int(pokemonData["base_experience"].(float64)),
		Stats:          *stats,
		Moves:          mvDataRes.Value,
		NextEvolutions: []NextEvoData{}, // TODO: fix later
		GrowthRate:     grRes.Value,
		Sprites:        spriteRes.Value,
	})
}

func fetchPokeAPIData(url string) (dict, error) {
	resp, err := http.Get(url)
	if err != nil {
		return dict{}, err
	}
	defer resp.Body.Close()

	var data dict
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return dict{}, err
	}
	return data, nil
}

func getSprites(pokeId uint) Result[Sprites] {
	frontUrl := fmt.Sprintf("%s/%d.png", SPRITEURLBASE, pokeId)
	backUrl := fmt.Sprintf("%s/back/%d.png", SPRITEURLBASE, pokeId)

	sprHandler := func(resp *http.Response) ([]byte, error) {
		if resp.Header.Get("Content-Type") == "image/png" {
			return io.ReadAll(resp.Body)
		}
		return nil, fmt.Errorf("Wrong Content-Type from network response.%v", resp.Header.Get("Content-Type"))
	}

	ftResp, err := http.Get(frontUrl)
	if err != nil {
		return Err[Sprites](err)
	}
	defer ftResp.Body.Close()

	ftSprite, err := sprHandler(ftResp)
	if err != nil {
		return Err[Sprites](err)
	}

	bkResp, err := http.Get(backUrl)
	if err != nil {
		return Err[Sprites](err)
	}
	defer bkResp.Body.Close()

	bkSprite, err := sprHandler(bkResp)
	if err != nil {
		return Err[Sprites](err)
	}
	return Ok(Sprites{ftSprite, bkSprite})
}

func getMovesData(pokeData dict) Result[[]MoveData] {
	var rbMoves []_mvIR
	names := mapset.NewSet[string]()

	pokeMoves, ok := pokeData["moves"].([]any)
	if !ok {
		return Err[[]MoveData](errors.New("No move data ..."))
	}

	for _, pmMv := range pokeMoves {
		md := pmMv.(dict)
		vgdTop, ok := md["version_group_details"].([]any)
		if !ok {
			return Err[[]MoveData](errors.New("Failed to parse 'version_group_details' in go map ..."))
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

	var detailed []MoveData
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
		statChanges := []StatChange{}

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
		if tAilment, ok := meta["ailment"].(dict)["name"].(string); ok {
			ailment = &tAilment
		}

		var ailmentChance *int = nil
		if tAilChnc, ok := meta["ailment_chance"].(int); ok {
			ailmentChance = &tAilChnc
		}

		var moveCategory *string = nil
		if tMovCat, ok := meta["category"].(dict)["name"].(string); ok {
			moveCategory = &tMovCat
		}

		var healing *int = nil
		if tHealing, ok := meta["healing"].(int); ok {
			healing = &tHealing
		}

		var drain *int = nil
		if tDrain, ok := meta["drain"].(int); ok {
			drain = &tDrain
		}

		detailed = append(detailed, MoveData{
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
			MoveCategory:  moveCategory,
			Healing:       healing,
			Drain:         drain,
			StatChanges:   statChanges,
		})
	}
	return Ok(detailed)
}

func getPokemonTypes(data dict) (string, *string, error) {
	var type1 string
	var type2 *string
	for _, t := range data["types"].([]any) {
		tm := t.(dict)
		fSlot, ok := tm["slot"].(float64)
		if !ok || fSlot != 0 {
			return "", nil, errors.New("Couldn't load data['type']['slot'] ")
		}
		var name string
		if name, ok = tm["type"].(dict)["name"].(string); !ok || name == "" {
			return "", nil, errors.New("Couldn't load data['type']['name'] ")
		}

		slot := int(fSlot)
		switch slot {
		case 1:
			type1 = name
		case 2:
			type2 = &name
		default:
			return "", nil, fmt.Errorf("Unknown type slot number found:%d\n", slot)
		}

	}
	return type1, type2, nil
}

func getStats(data dict) *StatsData {
	stats := make(map[string]int)
	for _, v := range data["stats"].([]any) {
		tm := v.(dict)
		name := tm["stat"].(dict)["name"].(string)
		baseStat := int(tm["base_stat"].(float64))
		stats[name] = baseStat
	}

	return &StatsData{
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
