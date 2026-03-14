package pearcut

import "context"

// NoopPublisher discards all events. Used as the default when no publisher is configured.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, AssignmentEvent) {}
func (NoopPublisher) Close() error                             { return nil }
