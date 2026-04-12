package pearcut

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/twmb/murmur3"
)

type engine struct {
	reader    AssignReader
	publisher EventPublisher
}

func NewEngine(reader AssignReader, publisher EventPublisher) Engine {
	if publisher == nil {
		publisher = NoopPublisher{}
	}
	return &engine{reader: reader, publisher: publisher}
}

func (e *engine) Assign(ctx context.Context, userID string, experimentSlug string, attributes map[string]string) (Assignment, error) {
	exp, err := e.reader.Get(experimentSlug)
	if err != nil {
		return Assignment{}, err
	}

	if exp.Status != StatusRunning {
		return Assignment{}, ErrExperimentNotRunning
	}

	// Overrides bypass layer check and targeting
	var variant string
	if v, ok := exp.Overrides[userID]; ok {
		variant = v
	} else {
		if !isIncludedByLayer(exp, userID) {
			return Assignment{}, ErrUserExcludedByLayer
		}

		if !matchesTargeting(exp.TargetingRules, attributes) {
			slog.Warn("user does not match targeting rules",
				"experiment", exp.Slug,
				"user_id", userID,
				"targeting_rules", exp.TargetingRules,
				"provided_attributes", attributes,
			)
			return Assignment{}, ErrUserNotTargeted
		}

		variant = hashVariant(exp, userID)
	}

	a := Assignment{Experiment: exp.Slug, Variant: variant}
	e.publisher.Publish(ctx, AssignmentEvent{
		Type:       "assignment",
		UserID:     userID,
		Experiment: a.Experiment,
		Variant:    a.Variant,
		Timestamp:  time.Now(),
	})
	return a, nil
}

func (e *engine) BulkAssign(ctx context.Context, userID string, experimentSlugs []string, attributes map[string]string) ([]Assignment, error) {
	experiments, err := e.reader.List(experimentSlugs, StatusRunning)
	if err != nil {
		return nil, fmt.Errorf("listing experiments: %w", err)
	}

	assignments := make([]Assignment, 0, len(experiments))
	for _, exp := range experiments {
		var variant string
		if v, ok := exp.Overrides[userID]; ok {
			variant = v
		} else {
			if !isIncludedByLayer(exp, userID) {
				continue
			}
			if !matchesTargeting(exp.TargetingRules, attributes) {
				slog.Warn("user does not match targeting rules",
					"experiment", exp.Slug,
					"user_id", userID,
					"targeting_rules", exp.TargetingRules,
					"provided_attributes", attributes,
				)
				continue
			}
			variant = hashVariant(exp, userID)
		}
		a := Assignment{Experiment: exp.Slug, Variant: variant}
		e.publisher.Publish(ctx, AssignmentEvent{
			Type:       "assignment",
			UserID:     userID,
			Experiment: a.Experiment,
			Variant:    a.Variant,
			Timestamp:  time.Now(),
		})
		assignments = append(assignments, a)
	}

	return assignments, nil
}

func matchesTargeting(rules []TargetingRule, attributes map[string]string) bool {
	for _, rule := range rules {
		val, ok := attributes[rule.Attribute]
		switch rule.Operator {
		case OperatorIn:
			if !ok || !slices.Contains(rule.Values, val) {
				return false
			}
		case OperatorNotIn:
			if ok && slices.Contains(rule.Values, val) {
				return false
			}
		}
	}
	return true
}

func hashBucket(key string, userID string, modulo uint32) uint32 {
	return murmur3.Sum32([]byte(key+userID)) % modulo
}

// isIncludedByLayer checks whether the user's bucket falls within the
// experiment's [from, to) range. No layer means 100% traffic (always included).
func isIncludedByLayer(exp Experiment, userID string) bool {
	if exp.Layer == (Layer{}) {
		return true
	}
	bucket := hashBucket(exp.Layer.Name, userID, 100)
	return bucket >= uint32(exp.Layer.From) && bucket < uint32(exp.Layer.To)
}

func hashVariant(exp Experiment, userID string) string {
	var totalWeight int
	for _, v := range exp.Variants {
		totalWeight += v.Weight
	}

	bucket := hashBucket(exp.Seed, userID, uint32(totalWeight))

	var cumulative uint32
	for _, v := range exp.Variants {
		cumulative += uint32(v.Weight)
		if bucket < cumulative {
			return v.Name
		}
	}

	return exp.Variants[len(exp.Variants)-1].Name
}
