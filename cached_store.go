package pearcut

import (
	"fmt"
	"sort"
	"sync"
)

type CachedStore struct {
	inner Store
	mu    sync.RWMutex
	cache map[string]Experiment
}

func NewCachedStore(inner Store) (*CachedStore, error) {
	experiments, err := inner.List(ExperimentFilter{})
	if err != nil {
		return nil, fmt.Errorf("warming cache: %w", err)
	}

	cache := make(map[string]Experiment, len(experiments))
	for _, exp := range experiments {
		cache[exp.Slug] = exp
	}

	return &CachedStore{inner: inner, cache: cache}, nil
}

func (c *CachedStore) Get(slug string) (Experiment, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	exp, ok := c.cache[slug]
	if !ok {
		return Experiment{}, ErrExperimentNotFound
	}
	return exp, nil
}

func (c *CachedStore) List(filter ExperimentFilter) ([]Experiment, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	slugSet := make(map[string]struct{}, len(filter.Slugs))
	for _, s := range filter.Slugs {
		slugSet[s] = struct{}{}
	}

	var results []Experiment
	for _, exp := range c.cache {
		if filter.Status != nil && exp.Status != *filter.Status {
			continue
		}
		if len(slugSet) > 0 {
			if _, ok := slugSet[exp.Slug]; !ok {
				continue
			}
		}
		results = append(results, exp)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Slug < results[j].Slug
	})

	return results, nil
}

func (c *CachedStore) Create(exp Experiment) error {
	if err := c.inner.Create(exp); err != nil {
		return err
	}

	fresh, err := c.inner.Get(exp.Slug)
	if err != nil {
		return fmt.Errorf("refreshing cache after create: %w", err)
	}

	c.mu.Lock()
	c.cache[fresh.Slug] = fresh
	c.mu.Unlock()

	return nil
}

func (c *CachedStore) Update(exp Experiment) error {
	if err := c.inner.Update(exp); err != nil {
		return err
	}

	fresh, err := c.inner.Get(exp.Slug)
	if err != nil {
		return fmt.Errorf("refreshing cache after update: %w", err)
	}

	c.mu.Lock()
	c.cache[fresh.Slug] = fresh
	c.mu.Unlock()

	return nil
}

func (c *CachedStore) Delete(slug string) error {
	if err := c.inner.Delete(slug); err != nil {
		return err
	}

	c.mu.Lock()
	delete(c.cache, slug)
	c.mu.Unlock()

	return nil
}
