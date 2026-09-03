package ping

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/TorchPing/go-torch/pkg/target"
	"github.com/TorchPing/go-torch/pkg/utils"
)

// NewPing Create Ping Instane
func NewPing() *Ping {
	ping := Ping{
		done: make(chan struct{}),
		stop: make(chan struct{}),
	}
	return &ping
}

func (ping *Ping) ensureChannels() {
	ping.mu.Lock()
	defer ping.mu.Unlock()
	if ping.done == nil {
		ping.done = make(chan struct{})
	}
	if ping.stop == nil {
		ping.stop = make(chan struct{})
	}
}

// SetTarget ..
func (ping *Ping) SetTarget(target *Target) {
	ping.mu.Lock()
	defer ping.mu.Unlock()
	if target == nil {
		ping.target = nil
		ping.result = nil
		return
	}
	targetCopy := *target
	ping.target = &targetCopy
	ping.result = &Result{Target: &targetCopy}
}

// Start the process
func (ping *Ping) Start() <-chan struct{} {
	return ping.StartContext(context.Background())
}

// StartContext starts the process and stops it when ctx is canceled.
func (ping *Ping) StartContext(ctx context.Context) <-chan struct{} {
	ping.ensureChannels()
	if ctx == nil {
		ctx = context.Background()
	}
	ping.startOnce.Do(func() {
		ping.mu.RLock()
		var target Target
		if ping.target != nil {
			target = *ping.target
		}
		result := ping.result
		ping.mu.RUnlock()
		go ping.run(ctx, target, result)
	})
	return ping.done
}

// Stop stops a running process. It is safe to call more than once.
func (ping *Ping) Stop() {
	ping.ensureChannels()
	ping.stopOnce.Do(func() {
		close(ping.stop)
	})
}

func (ping *Ping) run(ctx context.Context, targetInfo Target, result *Result) {
	runCtx, cancel := context.WithCancel(ctx)
	stopWatcherDone := make(chan struct{})
	go func() {
		select {
		case <-ping.stop:
			cancel()
		case <-runCtx.Done():
		}
		close(stopWatcherDone)
	}()
	defer func() {
		cancel()
		<-stopWatcherDone
		close(ping.done)
	}()

	if result == nil || targetInfo.Counter <= 0 {
		return
	}

	for attempt := 0; attempt < targetInfo.Counter; attempt++ {
		if attempt > 0 && !ping.wait(runCtx, targetInfo.Interval) {
			return
		}
		select {
		case <-runCtx.Done():
			return
		default:
		}

		duration, err := ping.ping(runCtx, targetInfo)
		ping.mu.Lock()
		result.Counter++
		if err != nil {
			fmt.Printf("Ping %s - failed: %s\n", targetInfo.Host, err)
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
		}
		ping.mu.Unlock()
	}
}

func (ping *Ping) wait(ctx context.Context, interval time.Duration) bool {
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
func (ping *Ping) Result() *Result {
	ping.mu.RLock()
	defer ping.mu.RUnlock()
	if ping.result == nil {
		return nil
	}
	result := *ping.result
	if ping.result.Target != nil {
		target := *ping.result.Target
		result.Target = &target
	}
	return &result
}

func (ping *Ping) ping(ctx context.Context, targetInfo Target) (time.Duration, error) {
	duration, errIfce := utils.TimeIt(func() interface{} {
		ctx, cancel := context.WithTimeout(ctx, targetInfo.Timeout)
		defer cancel()

		addresses, err := target.ResolvePublic(ctx, targetInfo.Host)
		if err != nil {
			return err
		}
		dialer := net.Dialer{}
		var lastErr error
		for _, address := range addresses {
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(address.String(), fmt.Sprintf("%d", targetInfo.Port)))
			if err != nil {
				lastErr = err
				continue
			}
			if err := conn.Close(); err != nil {
				return err
			}
			return nil
		}
		if lastErr != nil {
			return lastErr
		}
		return target.ErrNoPublicAddress
	})

	if errIfce != nil {
		err := errIfce.(error)
		return 0, err
	}
	return time.Duration(duration), nil
}
