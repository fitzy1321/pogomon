package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	. "pogomon/result"
)

type HttpGetter interface {
	Get(url string) (resp *http.Response, err error)
}

func topLevelPokemonData(client HttpGetter, pokemonId uint) Result[pp_t] {
	url := fmt.Sprintf("%s/pokemon/%d", BASEURL, pokemonId)

	pokemap, err := networkGetHandler(client, url)
	if err != nil {
		return Err[pp_t](err)
	}

	name := pokemap["name"].(string)

	type1, type2, err := parsePTypes(pokemap)
	if err != nil {
		return Err[pp_t](err)
	}

	mStats, err := parsePStats(pokemap)
	if err != nil {
		return Err[pp_t](err)
	} else if mStats == nil {
		mStats = &PokemonStats{}
	}

	moveCh := make(chan Result[[]MoveData])
	spriteCh := make(chan Result[*Sprites])
	evoCh := make(chan Result[[]NextEvoData])
	grCh := make(chan Result[*string])

	go func() {
		moveCh <- getMovesData(client, pokemap)
	}()

	go func() {
		spriteCh <- getSprites(client, pokemonId)
	}()

	go func() {
		if speciesUrl, ok := pokemap["species"].(dict)["url"].(string); ok {
			speciesData, err := networkGetHandler(client, speciesUrl)
			if err != nil {
				grCh <- Err[*string](err)
				evoCh <- Err[[]NextEvoData](err)
				return
			}

			go func() { evoCh <- Wrap(getEvoData(client, speciesData, name)) }()

			grstr, ok := speciesData["growth_rate"].(dict)["name"].(string)
			if !ok {
				grCh <- ErrFromStr[*string](fmt.Sprintf("Pokemon Id: #%d Couldn't load speciesData[\"growth_rate\"][\"name\"]\n", pokemonId))
				return
			}
			grCh <- Ok(&grstr)
		}
	}()

	moveRes := <-moveCh
	if moveRes.IsErr() {
		return Err[pp_t](moveRes.Error)
	}
	moves := moveRes.Value

	spriteRes := <-spriteCh
	if spriteRes.IsErr() {
		return Err[pp_t](spriteRes.Error)
	}
	sprites := spriteRes.Value

	var growthRate *string = nil
	grRes := <-grCh
	if grRes.IsErr() {
		fmt.Fprintf(os.Stderr, "Error fetching evolution data: %+v", grRes.Error)
	} else {
		growthRate = grRes.Value
	}

	var evoData = []NextEvoData{}
	evoRes := <-evoCh
	if evoRes.IsErr() {
		fmt.Fprintf(os.Stderr, "Error fetching evolution data: %+v", evoRes.Error)
	} else {
		evoData = evoRes.Value
	}

	baseExp := int(pokemap["base_experience"].(float64))
	pokemon := PokeApiData{
		ID:             pokemonId,
		Name:           name,
		Type1:          type1,
		Type2:          type2,
		BaseExperience: &baseExp,
		PokemonStats:   *mStats,
		Moves:          moves,
		NextEvolutions: evoData,
		GrowthRate:     growthRate,
		Sprites:        *sprites,
	}
	return Ok(&pokemon)
}

// this acts as a makeshift lru_cache(max_size=none)
var (
	requestCache = make(map[string]any)
	mu           sync.RWMutex
)

func networkGetHandler(client HttpGetter, url string) (dict, error) {
	mu.RLock()
	cResp, ok := requestCache[url].(dict)
	mu.RUnlock()
	if ok {
		return cResp, nil
	}

	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP Status Code: %d. HTTP Status MSG: %s", resp.StatusCode, resp.Status)
	}

	var data dict // don't make here, we want to check for nil later
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data != nil {
		mu.Lock()
		requestCache[url] = data
		mu.Unlock()
	}

	return data, nil
}

func getSprites(client HttpGetter, pokeId uint) Result[*Sprites] {
	if client == nil {
		client = http.DefaultClient
	}

	frontUrl := fmt.Sprintf("%s/%d.png", SPRITEURLBASE, pokeId)
	backUrl := fmt.Sprintf("%s/back/%d.png", SPRITEURLBASE, pokeId)

	sprHandler := func(resp *http.Response) ([]byte, error) {
		if resp.Header.Get("Content-Type") == "image/png" {
			return io.ReadAll(resp.Body)
		}
		return nil, fmt.Errorf("Wrong Content-Type from network response.%v", resp.Header.Get("Content-Type"))
	}

	ftResp, err := client.Get(frontUrl)
	if err != nil {
		return Err[*Sprites](err)
	}
	defer ftResp.Body.Close()

	ftSprite, err := sprHandler(ftResp)
	if err != nil {
		return Err[*Sprites](err)
	}

	bkResp, err := client.Get(backUrl)
	if err != nil {
		return Err[*Sprites](err)
	}
	defer bkResp.Body.Close()

	bkSprite, err := sprHandler(bkResp)
	if err != nil {
		return Err[*Sprites](err)
	}
	return Ok(&Sprites{ftSprite, bkSprite})
}

func getMovesData(client HttpGetter, pokeData dict) Result[[]MoveData] {
	movesIR := make(map[string]_mvIR)

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
				if _, has := movesIR[moveName]; !has {
					movesIR[moveName] = _mvIR{
						name:   moveName,
						level:  int(vgd["level_learned_at"].(float64)),
						url:    md["move"].(dict)["url"].(string),
						method: vgd["move_learn_method"].(dict)["name"].(string),
					}
				}
			}
		} // end for
	} // end for

	var detailed []MoveData
	for _, move := range movesIR {
		mvData, err := networkGetHandler(client, move.url)
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
			ID:            moveId,
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

func getEvoData(client HttpGetter, speciesData dict, targetPokemonName string) ([]NextEvoData, error) {
	// WARN no `return nil, nil` here.
	// WARN We need to return some value from this func, either an empty slice or error.
	evoChain, ok := speciesData["evolution_chain"].(dict)
	if !ok {
		return nil, errors.New("'evolution_chain' missing from species data")
	}
	evoChainUrl, ok := evoChain["url"].(string)
	if !ok {
		return nil, errors.New("'url' missing from 'evolution_chain' data")
	}
	chainData, err := networkGetHandler(client, evoChainUrl)
	if err != nil {
		return nil, err
	}
	if chainData == nil {
		return nil, fmt.Errorf("No additional chain data for species %s found", targetPokemonName)
	}

	result, err := _evoWalk(client, chainData["chain"].(dict), targetPokemonName)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func _evoWalk(client HttpGetter, node dict, target string) ([]NextEvoData, error) {
	species, ok := node["species"].(dict)
	if !ok {
		return nil, fmt.Errorf("'species' field missing for %s's chain data", target)
	}
	evolvesTo := node["evolves_to"].([]any)

	if speciesName, ok := species["name"]; ok && speciesName == target {
		entries := make([]NextEvoData, 0, len(evolvesTo))
		for _, childRaw := range evolvesTo {
			child := childRaw.(dict)
			entry, err := _buildEvoEntry(client, child)
			if err != nil {
				return nil, err
			}
			if entry != nil {
				entries = append(entries, *entry)
			}
		}
		return entries, nil
	}
	for _, childRaw := range evolvesTo {
		child, _ := childRaw.(dict)
		result, err := _evoWalk(client, child, target)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return []NextEvoData{}, nil
}

func _buildEvoEntry(client HttpGetter, nextNode map[string]any) (*NextEvoData, error) {
	species := nextNode["species"].(dict)
	nextName := species["name"].(string)

	details := nextNode["evolution_details"].([]any)
	var detail dict
	if len(details) > 0 {
		detail, _ = details[0].(dict)
	}
	if detail == nil {
		detail = make(map[string]any)
	}

	var minLevel int
	if lvl, ok := detail["min_level"].(float64); ok {
		minLevel = int(lvl)
	}

	trigger := detail["trigger"].(dict)["name"].(string)

	var item *string = nil
	tmpItems, ok := detail["item"].(dict)
	if ok {
		tmpItem := tmpItems["name"].(string)
		item = &tmpItem
	}

	if trigger == "trade" {
		trigger = "level-up"
		minLevel = TradeEvolutionLevel
		item = nil
	}

	nextPokeData, err := networkGetHandler(client, fmt.Sprintf("%s/pokemon/%s/", BASEURL, nextName))
	if err != nil {
		return nil, err
	}

	nextID, ok := nextPokeData["id"].(float64)
	if !ok || nextID == 0 || nextID > 151 {
		return nil, nil
	}

	return &NextEvoData{
		EvolvesIntoID:   uint(nextID),
		EvolvesIntoName: &nextName,
		Trigger:         &trigger,
		MinLevel:        &minLevel,
		Item:            item,
	}, nil
}

func parsePTypes(data dict) (string, *string, error) {
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
		slot := uint(fSlot)
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

func parsePStats(data dict) (*PokemonStats, error) {
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
