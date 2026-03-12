package main

type ReadStore interface {
	Get(slug string) (Experiment, error)
}

type memStore struct {
	experiments map[string]Experiment
}

func newMemStore(experiments []Experiment) *memStore {
	m := &memStore{experiments: make(map[string]Experiment, len(experiments))}
	for _, exp := range experiments {
		m.experiments[exp.Slug] = exp
	}
	return m
}

func (m *memStore) Get(slug string) (Experiment, error) {
	exp, ok := m.experiments[slug]
	if !ok {
		return Experiment{}, ErrExperimentNotFound
	}
	return exp, nil
}
