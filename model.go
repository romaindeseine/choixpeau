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

type Experiment struct {
	Slug      string            `json:"slug"`
	Status    ExperimentStatus  `json:"status"`
	Variants  []Variant         `json:"variants"`
	Overrides map[string]string `json:"overrides,omitempty"`
	Seed      string            `json:"seed,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type ExperimentFilter struct {
	Status *ExperimentStatus
	Slugs  []string
}

// Store interfaces

type ReadStore interface {
	Get(slug string) (Experiment, error)
	List(filter ExperimentFilter) ([]Experiment, error)
}

type WriteStore interface {
	Create(exp Experiment) error
	Update(exp Experiment) error
	Delete(slug string) error
}

type Store interface {
	ReadStore
	WriteStore
}

// Engine domain

type Assignment struct {
	Experiment string
	Variant    string
}

type Engine interface {
	Assign(ctx context.Context, experimentSlug string, userID string) (Assignment, error)
	BulkAssign(ctx context.Context, userID string, experimentSlugs []string) ([]Assignment, error)
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
