package resolve

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/TorchPing/go-torch/pkg/target"
	"github.com/TorchPing/go-torch/pkg/utils"
)

// NewResolve Create Resolve Instane
func NewResolve() *Resolve {
	resolve := Resolve{
		done: make(chan struct{}),
		stop: make(chan struct{}),
	}
	return &resolve
}

func (resolve *Resolve) ensureChannels() {
	resolve.mu.Lock()
	defer resolve.mu.Unlock()
	if resolve.done == nil {
		resolve.done = make(chan struct{})
	}
	if resolve.stop == nil {
		resolve.stop = make(chan struct{})
	}
}

// SetTarget ..
func (resolve *Resolve) SetTarget(target *Target) {
	resolve.mu.Lock()
	defer resolve.mu.Unlock()
	if target == nil {
		resolve.target = nil
		resolve.result = nil
		return
	}
	targetCopy := *target
	resolve.target = &targetCopy
	resolve.result = &Result{Target: &targetCopy}
}

// Start the process
func (resolve *Resolve) Start() <-chan struct{} {
	return resolve.StartContext(context.Background())
}

// StartContext starts the process and stops it when ctx is canceled.
func (resolve *Resolve) StartContext(ctx context.Context) <-chan struct{} {
	resolve.ensureChannels()
	if ctx == nil {
		ctx = context.Background()
	}
	resolve.startOnce.Do(func() {
		resolve.mu.RLock()
		var target Target
		if resolve.target != nil {
			target = *resolve.target
		}
		result := resolve.result
		resolve.mu.RUnlock()
		go resolve.run(ctx, target, result)
	})
	return resolve.done
}

// Stop stops a running process. It is safe to call more than once.
func (resolve *Resolve) Stop() {
	resolve.ensureChannels()
	resolve.stopOnce.Do(func() {
		close(resolve.stop)
	})
}

func (resolve *Resolve) run(ctx context.Context, targetInfo Target, result *Result) {
	runCtx, cancel := context.WithCancel(ctx)
	stopWatcherDone := make(chan struct{})
	go func() {
		select {
		case <-resolve.stop:
			cancel()
		case <-runCtx.Done():
		}
		close(stopWatcherDone)
	}()
	defer func() {
		cancel()
		<-stopWatcherDone
		close(resolve.done)
	}()

	if result == nil || targetInfo.Counter <= 0 {
		return
	}

	for attempt := 0; attempt < targetInfo.Counter; attempt++ {
		if attempt > 0 && !resolve.wait(runCtx, targetInfo.Interval) {
			return
		}
		select {
		case <-runCtx.Done():
			return
		default:
		}

		duration, addrs := resolve.resolve(runCtx, targetInfo)
		resolve.mu.Lock()
		result.Counter++
		if duration == 0 {
			fmt.Printf("Resolve %s - failed: %s\n", targetInfo.Host, addrs)
		} else {
			if result.MinDuration == 0 {
				result.MinDuration = duration
			}
			if result.MaxDuration == 0 {
				result.MaxDuration = duration
			}
			result.SuccessCounter++
			if duration > result.MaxDuration {
				result.MaxDuration = duration
			} else if duration < result.MinDuration {
				result.MinDuration = duration
			}
			result.TotalDuration += duration
			result.Addrs = append([]string(nil), addrs.([]string)...)
		}
		resolve.mu.Unlock()
	}
}

func (resolve *Resolve) wait(ctx context.Context, interval time.Duration) bool {
	if interval <= 0 {
		return true
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// Result return the result
func (resolve *Resolve) Result() *Result {
	resolve.mu.RLock()
	defer resolve.mu.RUnlock()
	if resolve.result == nil {
		return nil
	}
	result := *resolve.result
	if resolve.result.Target != nil {
		target := *resolve.result.Target
		result.Target = &target
	}
	result.Addrs = append([]string(nil), resolve.result.Addrs...)
	return &result
}

func (resolve *Resolve) resolve(ctx context.Context, targetInfo Target) (time.Duration, interface{}) {
	duration, res, errIfce := utils.TimeItWithResult(func() (interface{}, interface{}) {
		ctx, cancel := context.WithTimeout(ctx, targetInfo.Timeout)
		defer cancel()
		addrs, err := target.ResolvePublic(ctx, fmt.Sprintf("%s", targetInfo.Host))

		return addrs, err
	})

	if errIfce != nil {
		err := errIfce.(error)
		return 0, err
	}

	addresses := make([]string, 0, len(res.([]net.IP)))
	for _, address := range res.([]net.IP) {
		addresses = append(addresses, address.String())
	}
	return time.Duration(duration), addresses
}
