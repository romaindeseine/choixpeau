package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type configFile struct {
	Experiments []Experiment `yaml:"experiments"`
}

func loadExperiments(path string) ([]Experiment, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg configFile
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if err := validateExperiments(cfg.Experiments); err != nil {
		return nil, err
	}

	for i := range cfg.Experiments {
		if cfg.Experiments[i].Seed == "" {
			cfg.Experiments[i].Seed = cfg.Experiments[i].Slug
		}
	}

	return cfg.Experiments, nil
}

func validateSlug(exp Experiment) error {
	if exp.Slug == "" {
		return fmt.Errorf("experiment with empty slug")
	}
	return nil
}

func validateStatus(exp Experiment) error {
	switch exp.Status {
	case StatusDraft, StatusRunning, StatusPaused, StatusStopped:
		return nil
	default:
		return fmt.Errorf("experiment %q has invalid status %q", exp.Slug, exp.Status)
	}
}

func validateVariantName(exp Experiment) error {
	for _, v := range exp.Variants {
		if v.Name == "" {
			return fmt.Errorf("experiment %q has variant with empty name", exp.Slug)
		}
	}
	return nil
}

func validateVariantWeight(exp Experiment) error {
	for _, v := range exp.Variants {
		if v.Weight <= 0 {
			return fmt.Errorf("experiment %q variant %q has non-positive weight", exp.Slug, v.Name)
		}
	}
	return nil
}

func validateUniqueVariants(exp Experiment) error {
	names := make(map[string]bool, len(exp.Variants))
	for _, v := range exp.Variants {
		if names[v.Name] {
			return fmt.Errorf("experiment %q has duplicate variant %q", exp.Slug, v.Name)
		}
		names[v.Name] = true
	}
	return nil
}

func validateVariantsNotEmpty(exp Experiment) error {
	if len(exp.Variants) == 0 {
		return fmt.Errorf("experiment %q has no variants", exp.Slug)
	}
	return nil
}

func validateOverrides(exp Experiment) error {
	variantNames := make(map[string]bool, len(exp.Variants))
	for _, v := range exp.Variants {
		variantNames[v.Name] = true
	}
	for userID, variantName := range exp.Overrides {
		if !variantNames[variantName] {
			return fmt.Errorf("experiment %q override for %q references unknown variant %q", exp.Slug, userID, variantName)
		}
	}
	return nil
}

func validateVariants(exp Experiment) error {
	if err := validateVariantsNotEmpty(exp); err != nil {
		return err
	}
	if err := validateVariantName(exp); err != nil {
		return err
	}
	if err := validateVariantWeight(exp); err != nil {
		return err
	}
	if err := validateUniqueVariants(exp); err != nil {
		return err
	}
	return nil
}

func validateUniqueSlugs(exps []Experiment) error {
	slugs := make(map[string]bool, len(exps))
	for _, exp := range exps {
		if slugs[exp.Slug] {
			return fmt.Errorf("duplicate experiment slug %q", exp.Slug)
		}
		slugs[exp.Slug] = true
	}
	return nil
}

func validateExperiments(exps []Experiment) error {
	if err := validateUniqueSlugs(exps); err != nil {
		return err
	}
	for _, exp := range exps {
		if err := validateSlug(exp); err != nil {
			return err
		}
		if err := validateStatus(exp); err != nil {
			return err
		}
		if err := validateVariants(exp); err != nil {
			return err
		}
		if err := validateOverrides(exp); err != nil {
			return err
		}
	}
	return nil
}
