package pearcut

import (
	"fmt"
	"sync"
)

type CachedStore struct {
	inner Store
	mu    sync.RWMutex
	cache map[string]Experiment
}

func NewCachedStore(inner Store) (*CachedStore, error) {
	result, err := inner.List(ExperimentFilter{}, ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("warming cache: %w", err)
	}

	cache := make(map[string]Experiment, len(result.Experiments))
	for _, exp := range result.Experiments {
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

func (c *CachedStore) List(filter ExperimentFilter, opts ListOptions) (ExperimentListResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	all := make([]Experiment, 0, len(c.cache))
	for _, exp := range c.cache {
		all = append(all, exp)
	}

	filtered := filterExperiments(all, filter)
	sortExperiments(filtered, opts)
	page, total := paginateExperiments(filtered, opts)

	return ExperimentListResult{Experiments: page, Total: total}, nil
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
