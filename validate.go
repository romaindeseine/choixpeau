package pearcut

import "fmt"

func (e Experiment) Validate() error {
	if err := e.validateSlug(); err != nil {
		return err
	}
	if err := e.validateStatus(); err != nil {
		return err
	}
	if err := e.validateVariants(); err != nil {
		return err
	}
	if err := e.validateOverrides(); err != nil {
		return err
	}
	if err := e.validateTags(); err != nil {
		return err
	}
	if err := e.validateTargetingRules(); err != nil {
		return err
	}
	if err := e.validateLayer(); err != nil {
		return err
	}
	return nil
}

func (e Experiment) validateSlug() error {
	if e.Slug == "" {
		return fmt.Errorf("experiment with empty slug")
	}
	return nil
}

func (e Experiment) validateStatus() error {
	switch e.Status {
	case StatusDraft, StatusRunning, StatusPaused, StatusStopped:
		return nil
	default:
		return fmt.Errorf("experiment %q has invalid status %q", e.Slug, e.Status)
	}
}

func (e Experiment) validateVariants() error {
	if len(e.Variants) == 0 {
		return fmt.Errorf("experiment %q has no variants", e.Slug)
	}
	names := make(map[string]bool, len(e.Variants))
	for _, v := range e.Variants {
		if v.Name == "" {
			return fmt.Errorf("experiment %q has variant with empty name", e.Slug)
		}
		if v.Weight <= 0 {
			return fmt.Errorf("experiment %q variant %q has non-positive weight", e.Slug, v.Name)
		}
		if names[v.Name] {
			return fmt.Errorf("experiment %q has duplicate variant %q", e.Slug, v.Name)
		}
		names[v.Name] = true
	}
	return nil
}

func (e Experiment) validateTags() error {
	for _, tag := range e.Tags {
		if tag == "" {
			return fmt.Errorf("experiment %q has empty tag", e.Slug)
		}
	}
	return nil
}

func (e Experiment) validateTargetingRules() error {
	for i, rule := range e.TargetingRules {
		if rule.Attribute == "" {
			return fmt.Errorf("experiment %q targeting rule %d has empty attribute", e.Slug, i)
		}
		switch rule.Operator {
		case OperatorIn, OperatorNotIn:
			// valid
		default:
			return fmt.Errorf("experiment %q targeting rule %d has invalid operator %q", e.Slug, i, rule.Operator)
		}
		if len(rule.Values) == 0 {
			return fmt.Errorf("experiment %q targeting rule %d has empty values", e.Slug, i)
		}
	}
	return nil
}

func (e Experiment) validateLayer() error {
	if e.Layer == (Layer{}) {
		return nil
	}
	if e.Layer.Name == "" {
		return fmt.Errorf("experiment %q has layer with empty name", e.Slug)
	}
	if e.Layer.TrafficPercentage < 0 || e.Layer.TrafficPercentage > 100 {
		return fmt.Errorf("experiment %q has invalid layer traffic_percentage %d (must be 0-100)", e.Slug, e.Layer.TrafficPercentage)
	}
	return nil
}

func (e Experiment) validateOverrides() error {
	variantNames := make(map[string]bool, len(e.Variants))
	for _, v := range e.Variants {
		variantNames[v.Name] = true
	}
	for userID, variantName := range e.Overrides {
		if !variantNames[variantName] {
			return fmt.Errorf("experiment %q override for %q references unknown variant %q", e.Slug, userID, variantName)
		}
	}
	return nil
}
