package setup

type (
	fullPokeData struct {
		Id             uint
		Name           string
		Type1          string
		Type2          *string // nullable
		BaseExperience int
		Stats          statsData
		Moves          []moveData
		NextEvolutions []nextEvoData
		GrowthRate     *string // nullable
		FrontSprite    []byte
		BackSprite     []byte
	}

	statsData struct {
		Attack         int
		Defense        int
		Hp             int
		SpecialAttack  int
		SpecialDefense int
		Speed          int
	}

	moveData struct {
		Name          string
		LevelLearned  uint
		LearnMethod   string
		MaxPp         int
		Power         *int         // nullable
		Accuracy      *int         // nullable
		Type          *string      // TODO: should this be nullable?
		DamageClass   *string      // nullable
		Ailment       *string      // nullable
		AilmentChance *int         // nullable
		MoveCategory  *string      // nullable
		Healing       *int         // nullable
		Drain         *int         // nullable
		StatChanges   []statChange // TODO: maybe nullable?

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

	_mvIR struct {
		name   string
		level  int
		url    string
		method string
	}
)
