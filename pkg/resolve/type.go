package resolve

import (
	"sync"
	"time"
)

// Ping data struct
type Resolve struct {
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

	Counter  int
	Interval time.Duration
	Timeout  time.Duration
}

// Result data struct
type Result struct {
	Counter        int
	SuccessCounter int
	Target         *Target
	Addrs          []string

	MinDuration   time.Duration
	MaxDuration   time.Duration
	TotalDuration time.Duration
}
