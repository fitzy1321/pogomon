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

type (
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
		EvolvesIntoID uint
		EvolvesInto   string
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

func topLevelPokemonData(client *http.Client, pokemonId uint) Result[PokeApiData] {
	url := fmt.Sprintf("%s/pokemon/%d", BASEURL, pokemonId)

	pokemap, err := networkHandler(client, url)
	if err != nil {
		return Err[PokeApiData](err)
	}

	name := pokemap["name"].(string)

	type1, type2, err := parsePTypes(pokemap)
	if err != nil {
		return Err[PokeApiData](err)
	}

	mStats, err := parsePStats(pokemap)
	if err != nil {
		return Err[PokeApiData](err)
	} else if mStats == nil {
		mStats = &PokemonStats{}
	}

	moveCh := make(chan Result[[]MoveData], 1)
	spriteCh := make(chan Result[Sprites], 1)
	evoCh := make(chan Result[[]NextEvoData], 1)
	grCh := make(chan Result[*string], 1)

	go func() {
		moveCh <- getMovesData(client, pokemap)
	}()

	go func() {
		spriteCh <- getSprites(client, pokemonId)
	}()

	go func() {
		if speciesUrl, ok := pokemap["species"].(dict)["url"].(string); ok {
			speciesData, err := networkHandler(client, speciesUrl)
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
		return Err[PokeApiData](moveRes.Error)
	}

	spriteRes := <-spriteCh
	if spriteRes.IsErr() {
		return Err[PokeApiData](spriteRes.Error)
	}

	grRes := <-grCh
	if grRes.IsErr() {
		return Err[PokeApiData](grRes.Error)
	}

	evoRes := <-evoCh
	if evoRes.IsErr() {
		return Err[PokeApiData](evoRes.Error)
	}

	baseExp := int(pokemap["base_experience"].(float64))
	return Ok(PokeApiData{
		ID:             pokemonId,
		Name:           name,
		Type1:          type1,
		Type2:          type2,
		BaseExperience: &baseExp,
		PokemonStats:   *mStats,
		Moves:          moveRes.Value,
		NextEvolutions: evoRes.Value,
		GrowthRate:     grRes.Value,
		Sprites:        spriteRes.Value,
	})
}

func networkHandler(client *http.Client, url string) (dict, error) {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)
	if err != nil {
		return dict{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP Status Code: %d. HTTP Status MSG: %s", resp.StatusCode, resp.Status)
	}

	var data dict
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return dict{}, err
	}
	return data, nil
}

func getMovesData(client *http.Client, pokeData dict) Result[[]MoveData] {
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
		mvData, err := networkHandler(client, move.url)
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

func getSprites(client *http.Client, pokeId uint) Result[Sprites] {
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
		return Err[Sprites](err)
	}
	defer ftResp.Body.Close()

	ftSprite, err := sprHandler(ftResp)
	if err != nil {
		return Err[Sprites](err)
	}

	bkResp, err := client.Get(backUrl)
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

func buildEvoEntry(client *http.Client, nextNode map[string]any) (*NextEvoData, error) {
	species := nextNode["species"].(dict)
	nextName := species["name"].(string)

	details := nextNode["evolution_details"].([]any)
	var detail dict
	if len(details) > 0 {
		detail, _ = details[0].(dict)
	}
	if detail == nil {
		detail = dict{}
	}

	var minLevel uint
	if lvl, ok := detail["min_level"].(float64); ok {
		minLevel = uint(lvl)
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

	nextPokeData, err := networkHandler(client, fmt.Sprintf("%s/pokemon/%s/", BASEURL, nextName))
	if err != nil {
		return nil, err
	}

	nextID, ok := nextPokeData["id"].(float64)
	if !ok || nextID == 0 || nextID > 151 {
		return nil, nil
	}

	return &NextEvoData{
		EvolvesIntoID: uint(nextID),
		EvolvesInto:   nextName,
		Trigger:       trigger,
		MinLevel:      minLevel,
		Item:          item,
	}, nil
}

func evoWalk(client *http.Client, node dict, target string) ([]NextEvoData, error) {
	species := node["species"].(dict)
	evolvesTo := node["evolves_to"].([]any)

	if speciesName, ok := species["name"]; ok && speciesName == target {
		entries := make([]NextEvoData, 0, len(evolvesTo))
		for _, childRaw := range evolvesTo {
			child := childRaw.(dict)
			entry, err := buildEvoEntry(client, child)
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
		result, err := evoWalk(client, child, target)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, nil
}

func getEvoData(client *http.Client, speciesData dict, targetPokemonName string) ([]NextEvoData, error) {
	evo_chain, ok := speciesData["evolution_chain"].(dict)
	if !ok {
		return nil, nil // TODO is this ok?
	}
	evoChainUrl, ok := evo_chain["url"].(string)
	if !ok {
		return nil, nil
	}
	chainData, err := networkHandler(client, evoChainUrl)
	if err != nil {
		return nil, err
	}
	if chainData == nil {
		return nil, nil
	}

	result, err := evoWalk(client, chainData["chain"].(dict), targetPokemonName)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return []NextEvoData{}, nil
	}

	return result, nil
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
