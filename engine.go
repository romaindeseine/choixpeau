package main

import (
	"fmt"

	"github.com/twmb/murmur3"
)

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

	return assignExperiment(exp, userID), nil
}

func (e *engine) BulkAssign(userID string, experimentSlugs []string) ([]Assignment, error) {
	status := StatusRunning
	filter := ExperimentFilter{Status: &status}
	if len(experimentSlugs) > 0 {
		filter.Slugs = experimentSlugs
	}

	experiments, err := e.store.List(filter)
	if err != nil {
		return nil, fmt.Errorf("listing experiments: %w", err)
	}

	assignments := make([]Assignment, 0, len(experiments))
	for _, exp := range experiments {
		assignments = append(assignments, assignExperiment(exp, userID))
	}
	return assignments, nil
}

func assignExperiment(exp Experiment, userID string) Assignment {
	if v, ok := exp.Overrides[userID]; ok {
		return Assignment{Experiment: exp.Slug, Variant: v, UserID: userID}
	}

	h := murmur3.Sum32([]byte(exp.Seed + userID))

	var totalWeight int
	for _, v := range exp.Variants {
		totalWeight += v.Weight
	}

	bucket := h % uint32(totalWeight)

	var cumulative uint32
	for _, v := range exp.Variants {
		cumulative += uint32(v.Weight)
		if bucket < cumulative {
			return Assignment{Experiment: exp.Slug, Variant: v.Name, UserID: userID}
		}
	}

	return Assignment{Experiment: exp.Slug, Variant: exp.Variants[len(exp.Variants)-1].Name, UserID: userID}
}
