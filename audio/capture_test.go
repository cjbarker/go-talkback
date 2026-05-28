package audio_test

import (
	"testing"

	"github.com/cbarker/go-talkback/audio"
)

// TestNew_SmokeTest verifies that a Recorder can be created and closed.
// The test is skipped when no audio context is available (e.g. CI without
// audio hardware).
func TestNew_SmokeTest(t *testing.T) {
	r, err := audio.New()
	if err != nil {
		t.Skipf("audio context unavailable (no hardware?): %v", err)
	}
	defer r.Close()
}

// TestStop_BeforeStart verifies that Stop returns nil when called without
// a preceding Start, rather than panicking or returning stale data.
func TestStop_BeforeStart(t *testing.T) {
	r, err := audio.New()
	if err != nil {
		t.Skipf("audio context unavailable: %v", err)
	}
	defer r.Close()

	samples := r.Stop()
	if samples != nil {
		t.Errorf("Stop before Start: got %d samples, want nil", len(samples))
	}
}

// TestStop_Idempotent verifies that calling Stop twice doesn't panic.
func TestStop_Idempotent(t *testing.T) {
	r, err := audio.New()
	if err != nil {
		t.Skipf("audio context unavailable: %v", err)
	}
	defer r.Close()

	r.Stop()
	r.Stop() // second call must not panic or deadlock
}
