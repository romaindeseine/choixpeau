package pearcut

import (
	"context"
	"time"
)

// Experiment domain

type ExperimentStatus string

const (
	StatusDraft   ExperimentStatus = "draft"
	StatusRunning ExperimentStatus = "running"
	StatusPaused  ExperimentStatus = "paused"
	StatusStopped ExperimentStatus = "stopped"
)

type Variant struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type TargetingOperator string

const (
	OperatorIn    TargetingOperator = "in"
	OperatorNotIn TargetingOperator = "not_in"
)

type TargetingRule struct {
	Attribute string            `json:"attribute"`
	Operator  TargetingOperator `json:"operator"`
	Values    []string          `json:"values"`
}

type Layer struct {
	Name string `json:"name"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

type Experiment struct {
	Slug           string            `json:"slug"`
	Status         ExperimentStatus  `json:"status"`
	Variants       []Variant         `json:"variants"`
	Overrides      map[string]string `json:"overrides,omitempty"`
	Seed           string            `json:"seed,omitempty"`
	TargetingRules []TargetingRule   `json:"targeting_rules,omitempty"`
	Layer          Layer             `json:"layer,omitempty"`
	Description    string            `json:"description,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Owner          string            `json:"owner,omitempty"`
	Hypothesis     string            `json:"hypothesis,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type ExperimentFilter struct {
	Status *ExperimentStatus
	Tags   []string
}

type ListOptions struct {
	Page    int // 1-based; 0 means no pagination
	PerPage int // 0 means no limit
}

type ExperimentListResult struct {
	Experiments []Experiment
	Total       int
}

type ExperimentStore interface {
	Get(slug string) (Experiment, error)
	List(filter ExperimentFilter, opts ListOptions) (ExperimentListResult, error)
	Create(exp Experiment) error
	Update(exp Experiment) error
	Delete(slug string) error
}

// Assign domain

type Assignment struct {
	Experiment string
	Variant    string
}

type AssignReader interface {
	Get(slug string) (Experiment, error)
	List(slugs []string, status ExperimentStatus) ([]Experiment, error)
}

type AssignWriter interface {
	Set(exp Experiment)
	Delete(slug string)
}

type AssignStore interface {
	AssignReader
	AssignWriter
}

type Engine interface {
	Assign(ctx context.Context, userID string, experimentSlug string, attributes map[string]string) (Assignment, error)
	BulkAssign(ctx context.Context, userID string, experimentSlugs []string, attributes map[string]string) ([]Assignment, error)
}

// Event publishing

type AssignmentEvent struct {
	Type       string    `json:"type"`
	UserID     string    `json:"user_id"`
	Experiment string    `json:"experiment"`
	Variant    string    `json:"variant"`
	Timestamp  time.Time `json:"timestamp"`
}

type EventPublisher interface {
	Publish(ctx context.Context, event AssignmentEvent)
	Close() error
}
