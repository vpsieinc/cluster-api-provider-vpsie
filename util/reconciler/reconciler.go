package reconciler

import "time"

const (
	DefaulteLoopTimeout   = 90 * time.Minute
	DefaultMappingTimeout = 60 * time.Second
)

func DefaultedLoopTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaulteLoopTimeout
	}
	return timeout
}
