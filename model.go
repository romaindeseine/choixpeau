package choixpeau

import "time"

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

type Assignment struct {
	Experiment string
	Variant    string
}

type ExperimentFilter struct {
	Status *ExperimentStatus
	Slugs  []string
}

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

type Engine interface {
	Assign(experimentSlug string, userID string) (Assignment, error)
	BulkAssign(userID string, experimentSlugs []string) ([]Assignment, error)
}
