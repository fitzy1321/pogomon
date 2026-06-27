package setup

type (
	FullPokeData struct {
		Id             uint
		Name           string
		Type1          string
		Type2          *string // nullable
		BaseExperience int
		Stats          StatsData
		Moves          []MoveData
		NextEvolutions []NextEvoData
		GrowthRate     *string // nullable
		Sprites
	}

	StatsData struct {
		Attack         int
		Defense        int
		Hp             int
		SpecialAttack  int
		SpecialDefense int
		Speed          int
	}

	MoveData struct {
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
		StatChanges   []StatChange // TODO: maybe nullable?

	}

	StatChange struct {
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
		front, back []byte
	}

	_mvIR struct {
		name   string
		level  int
		url    string
		method string
	}
)
