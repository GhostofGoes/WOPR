package ui

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// testWatchdog runs a watchdog with short limits; tripped receives its reason.
func testWatchdog(t *testing.T, stuck, grace time.Duration) (w *watchdog, sig chan os.Signal, tripped chan string) {
	t.Helper()
	sig = make(chan os.Signal, 1)
	tripped = make(chan string, 1)
	w = newWatchdog(sig, func(reason string) { tripped <- reason })
	w.stuck, w.grace = stuck, grace
	go w.watch()
	return w, sig, tripped
}

func noTrip(t *testing.T, tripped chan string) {
	t.Helper()
	select {
	case r := <-tripped:
		t.Fatalf("tripped: %s", r)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestWatchdogQuietWhenSteady(t *testing.T) {
	t.Parallel()
	w, sig, tripped := testWatchdog(t, 200*time.Millisecond, 100*time.Millisecond)
	for range 100 {
		w.do(func() {})
	}
	sig <- syscall.SIGINT // idle: Bubble Tea's own handler deals with it
	w.do(func() {})
	noTrip(t, tripped)
}

func TestWatchdogTripsOnAStuckUpdate(t *testing.T) {
	t.Parallel()
	w, _, tripped := testWatchdog(t, 200*time.Millisecond, 100*time.Millisecond)
	release := make(chan struct{})
	go w.do(func() { <-release })
	defer close(release)
	select {
	case r := <-tripped:
		if !strings.Contains(r, "stopped responding") {
			t.Errorf("reason %q", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a stuck Update did not trip the watchdog")
	}
}

func TestWatchdogTripsOnASignalWhileStuck(t *testing.T) {
	t.Parallel()
	w, sig, tripped := testWatchdog(t, time.Hour, 100*time.Millisecond) // only the signal can trip it
	release := make(chan struct{})
	started := make(chan struct{})
	go w.do(func() { close(started); <-release })
	defer close(release)
	<-started
	sig <- syscall.SIGTERM
	select {
	case r := <-tripped:
		if !strings.Contains(r, "not responding") {
			t.Errorf("reason %q", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a signal during a stuck Update did not trip the watchdog")
	}
}

func TestWatchdogLetsASignalledUpdateFinish(t *testing.T) {
	t.Parallel()
	w, sig, tripped := testWatchdog(t, time.Hour, 5*time.Second)
	release := make(chan struct{})
	started := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		w.do(func() { close(started); <-release })
		close(finished)
	}()
	<-started
	sig <- syscall.SIGINT
	close(release)
	<-finished
	noTrip(t, tripped)
}
