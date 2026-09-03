package resolve

import (
	"context"
	"testing"
	"time"
)

func waitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("operation did not finish")
	}
}

func TestZeroCounterFinishes(t *testing.T) {
	resolver := NewResolve()
	resolver.SetTarget(&Target{Host: "8.8.8.8"})
	waitForDone(t, resolver.Start())
}

func TestCanceledContextFinishes(t *testing.T) {
	resolver := NewResolve()
	resolver.SetTarget(&Target{Host: "8.8.8.8", Counter: 3, Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waitForDone(t, resolver.StartContext(ctx))
}

func TestStopBeforeStartFinishes(t *testing.T) {
	resolver := NewResolve()
	resolver.SetTarget(&Target{Host: "8.8.8.8", Counter: 3, Timeout: time.Second})
	resolver.Stop()
	waitForDone(t, resolver.Start())
}
