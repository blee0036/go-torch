package ping

import (
	"sync"
	"time"
)

// Ping data struct
type Ping struct {
	target *Target
	done   chan struct{}
	stop   chan struct{}

	mu        sync.RWMutex
	startOnce sync.Once
	stopOnce  sync.Once
	result    *Result
}

// Target data struct
type Target struct {
	Host string
	Port uint16

	Counter  int
	Interval time.Duration
	Timeout  time.Duration
}

// Result data struct
type Result struct {
	Counter        int
	SuccessCounter int
	Target         *Target

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration
}
