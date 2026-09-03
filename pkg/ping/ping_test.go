package ping

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
	pinger := NewPing()
	pinger.SetTarget(&Target{Host: "8.8.8.8", Port: 443})
	waitForDone(t, pinger.Start())
}

func TestCanceledContextFinishes(t *testing.T) {
	pinger := NewPing()
	pinger.SetTarget(&Target{Host: "8.8.8.8", Port: 443, Counter: 3, Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waitForDone(t, pinger.StartContext(ctx))
}

func TestStopBeforeStartFinishes(t *testing.T) {
	pinger := NewPing()
	pinger.SetTarget(&Target{Host: "8.8.8.8", Port: 443, Counter: 3, Timeout: time.Second})
	pinger.Stop()
	waitForDone(t, pinger.Start())
}
