package main

import "time"

type ExperimentStatus string

const (
	StatusDraft   ExperimentStatus = "draft"
	StatusRunning ExperimentStatus = "running"
	StatusPaused  ExperimentStatus = "paused"
	StatusStopped ExperimentStatus = "stopped"
)

type Variant struct {
	Name   string `json:"name" yaml:"name"`
	Weight int    `json:"weight" yaml:"weight"`
}

type Experiment struct {
	Slug      string            `json:"slug" yaml:"slug"`
	Status    ExperimentStatus  `json:"status" yaml:"status"`
	Variants  []Variant         `json:"variants" yaml:"variants"`
	Overrides map[string]string `json:"overrides,omitempty" yaml:"overrides,omitempty"`
	Seed      string            `json:"seed,omitempty" yaml:"seed,omitempty"`
	CreatedAt time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" yaml:"updated_at"`
}

type Assignment struct {
	Experiment string `json:"experiment"`
	Variant    string `json:"variant"`
	UserID     string `json:"user_id"`
}

type Engine interface {
	Assign(experimentSlug string, userID string) (Assignment, error)
}
