package main

type engine struct {
	store ReadStore
}

func NewEngine(store ReadStore) Engine {
	return &engine{store: store}
}

func (e *engine) Assign(experimentSlug string, userID string) (Assignment, error) {
	exp, err := e.store.Get(experimentSlug)
	if err != nil {
		return Assignment{}, err
	}

	if exp.Status != StatusRunning {
		return Assignment{}, ErrExperimentNotRunning
	}

	if v, ok := exp.Overrides[userID]; ok {
		return Assignment{Experiment: experimentSlug, Variant: v, UserID: userID}, nil
	}

	h := murmur3_32([]byte(exp.Seed+userID), 0)

	var totalWeight int
	for _, v := range exp.Variants {
		totalWeight += v.Weight
	}

	bucket := h % uint32(totalWeight)

	var cumulative uint32
	for _, v := range exp.Variants {
		cumulative += uint32(v.Weight)
		if bucket < cumulative {
			return Assignment{Experiment: experimentSlug, Variant: v.Name, UserID: userID}, nil
		}
	}

	return Assignment{Experiment: experimentSlug, Variant: exp.Variants[len(exp.Variants)-1].Name, UserID: userID}, nil
}
