package pearcut

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/murmur3"
)

type engine struct {
	store     ReadStore
	publisher EventPublisher
}

func NewEngine(store ReadStore, publisher EventPublisher) Engine {
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	return &engine{store: store, publisher: publisher}
}

func (e *engine) Assign(ctx context.Context, experimentSlug string, userID string) (Assignment, error) {
	exp, err := e.store.Get(experimentSlug)
	if err != nil {
		return Assignment{}, err
	}

	if exp.Status != StatusRunning {
		return Assignment{}, ErrExperimentNotRunning
	}

	a := Assignment{Experiment: exp.Slug, Variant: assignVariant(exp, userID)}
	e.publisher.Publish(ctx, AssignmentEvent{
		UserID:     userID,
		Experiment: a.Experiment,
		Variant:    a.Variant,
		Timestamp:  time.Now(),
	})
	return a, nil
}

func (e *engine) BulkAssign(ctx context.Context, userID string, experimentSlugs []string) ([]Assignment, error) {
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
		a := Assignment{Experiment: exp.Slug, Variant: assignVariant(exp, userID)}
		e.publisher.Publish(ctx, AssignmentEvent{
			UserID:     userID,
			Experiment: a.Experiment,
			Variant:    a.Variant,
			Timestamp:  time.Now(),
		})
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func assignVariant(exp Experiment, userID string) string {
	if v, ok := exp.Overrides[userID]; ok {
		return v
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
			return v.Name
		}
	}

	return exp.Variants[len(exp.Variants)-1].Name
}
