package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	. "go-pokebattle/result"

	mapset "github.com/deckarep/golang-set/v2"
)

func rawApiDataToStructs(pokeId uint) Result[PokeApiData] {
	url := fmt.Sprintf("%s/pokemon/%d", BASEURL, pokeId)

	pokemap, err := fetchPokeAPIData(url)
	if err != nil {
		return Err[PokeApiData](err)
	}

	type1, type2, err := getPokemonTypes(pokemap)
	if err != nil {
		return Err[PokeApiData](err)
	}

	mStats, err := getStats(pokemap)
	if err != nil {
		return Err[PokeApiData](err)
	} else if mStats == nil {
		mStats = &PokemonStats{}
	}

	moveCh := make(chan Result[[]MoveData], 1)
	spriteCh := make(chan Result[Sprites], 1)
	grCh := make(chan Result[*string], 1)

	go func() {
		moveCh <- getMovesData(pokemap)
	}()

	go func() {
		spriteCh <- getSprites(pokeId)
	}()

	go func() {
		if speciesUrl, ok := pokemap["species"].(dict)["url"].(string); ok {
			speciesData, err := fetchPokeAPIData(speciesUrl)
			if err != nil {
				grCh <- Err[*string](err)
				return
			}
			grstr, ok := speciesData["growth_rate"].(dict)["name"].(string)
			if !ok {
				grCh <- ErrFromStr[*string](fmt.Sprintf("Pokemon Id: #%d Couldn't load speciesData[\"growth_rate\"][\"name\"]\n", pokeId))
				return
			}
			grCh <- Ok(&grstr)
			// TODO: Evolution Data
		}
	}()

	moveRes := <-moveCh
	if err := moveRes.Error; err != nil {
		return Err[PokeApiData](err)
	}
	move := moveRes.Value

	grRes := <-grCh
	if err := grRes.Error; err != nil {
		return Err[PokeApiData](err)
	}
	growthRate := grRes.Value

	spriteRes := <-spriteCh
	if err := spriteRes.Error; err != nil {
		return Err[PokeApiData](err)
	}
	sprites := spriteRes.Value

	baseExp := int(pokemap["base_experience"].(float64))
	return Ok(PokeApiData{
		Id:             pokeId,
		Name:           pokemap["name"].(string),
		Type1:          type1,
		Type2:          type2,
		BaseExperience: &baseExp,
		PokemonStats:   *mStats,
		Moves:          move,
		NextEvolutions: []NextEvoData{}, // TODO: fix later
		GrowthRate:     growthRate,
		Sprites:        sprites,
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
		return Err[[]MoveData](errors.New("Couldn't load data[\"move\"]"))
	}

	for _, pmMv := range pokeMoves {
		md := pmMv.(dict)
		vgdTop, ok := md["version_group_details"].([]any)
		if !ok {
			return Err[[]MoveData](errors.New("Couldn't load data[\"moves\"][\"version_group_details\"]"))
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

		moveId := uint(mvData["id"].(float64))

		meta, ok := mvData["meta"].(dict)
		if !ok {
			// TODO: error handle idk man ...
		}

		// TODO: implement []statChange data
		statChanges := []statChange{}

		var power *int = nil
		if tp, ok := mvData["power"].(float64); ok {
			ttp := int(tp)
			power = &ttp
		}

		var acc *int = nil
		if tacc, ok := mvData["accuracy"].(float64); ok {
			ttacc := int(tacc)
			acc = &ttacc
		}

		var mpp int = 0
		if tmpp, ok := mvData["pp"].(float64); ok {
			mpp = int(tmpp)
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
		if tAilChnc, ok := meta["ailment_chance"].(float64); ok {
			atv := int(tAilChnc)
			ailmentChance = &atv
		}

		var moveCategory *string = nil
		if tMovCat, ok := meta["category"].(dict)["name"].(string); ok {
			moveCategory = &tMovCat
		}

		var healing *int = nil
		if tHealing, ok := meta["healing"].(float64); ok {
			tth := int(tHealing)
			healing = &tth
		}

		var drain *int = nil
		if tDrain, ok := meta["drain"].(float64); ok {
			ttd := int(tDrain)
			drain = &ttd
		}

		detailed = append(detailed, MoveData{
			Id:            moveId,
			Name:          move.name,
			LevelLearned:  move.level,
			LearnMethod:   &move.method,
			MaxPP:         mpp,
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
		if !ok {
			return "", nil, errors.New("Couldn't load data[\"types\"][\"slot\"] ")
		}
		var name string
		if name, ok = tm["type"].(dict)["name"].(string); !ok || name == "" {
			return "", nil, errors.New("Couldn't load data[\"types\"][\"type\"][\"name\"] ")
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

func getStats(data dict) (*PokemonStats, error) {
	mStats := make(map[string]int)
	for _, v := range data["stats"].([]any) {
		tm := v.(dict)
		name, ok := tm["stat"].(dict)["name"].(string)
		if !ok || name == "" {
			return nil, errors.New("Couldn't load data[\"stats\"][\"stat\"][\"name\"]")
		}
		t, ok := tm["base_stat"].(float64)
		if !ok {
			return nil, errors.New("Coudn't load data[\"stats\"][\"base_stat\"]")
		}
		baseStat := int(t)
		mStats[name] = baseStat
	}

	return &PokemonStats{
		Attack:    mStats["attack"],
		Defense:   mStats["defense"],
		HP:        mStats["hp"],
		SpAttack:  mStats["special-attack"],
		SpDefense: mStats["special-defense"],
		Speed:     mStats["speed"],
	}, nil
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
