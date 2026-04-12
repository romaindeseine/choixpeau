package pearcut

import "errors"

var (
	ErrExperimentNotFound    = errors.New("experiment not found")
	ErrExperimentNotRunning  = errors.New("experiment not running")
	ErrExperimentExists      = errors.New("experiment already exists")
	ErrUserNotTargeted       = errors.New("user does not match targeting rules")
	ErrUserExcludedByTraffic = errors.New("user excluded by traffic percentage")
	ErrLayerTrafficExceeded  = errors.New("layer traffic percentage sum exceeds 100")
)
