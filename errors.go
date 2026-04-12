package pearcut

import "errors"

var (
	ErrExperimentNotFound   = errors.New("experiment not found")
	ErrExperimentNotRunning = errors.New("experiment not running")
	ErrExperimentExists     = errors.New("experiment already exists")
	ErrUserNotTargeted      = errors.New("user does not match targeting rules")
	ErrUserExcludedByLayer  = errors.New("user excluded by layer")
	ErrLayerRangesOverlap   = errors.New("layer ranges overlap")
)
