package main

import "errors"

var (
	ErrExperimentNotFound   = errors.New("experiment not found")
	ErrExperimentNotRunning = errors.New("experiment not running")
)
