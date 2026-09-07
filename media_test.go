package media

import (
	"testing"
)

func runTests(t *testing.T) {
	testConstraintsBuilder(t)
	testZeroConstraints(t)
	testNonWASMStubs(t)
}

// testConstraintsBuilder covers the builder state that is target-agnostic. The
// JavaScript object shape (plan tests 1-4) needs js.Value and is asserted in
// testConstraintsJS, under //go:build wasm.
func testConstraintsBuilder(t *testing.T) {
	if c := Camera(); !c.hasVideo || c.hasAudio {
		t.Errorf("Camera() = %+v, want video only", c)
	}
	if c := Microphone(); c.hasVideo || !c.hasAudio {
		t.Errorf("Microphone() = %+v, want audio only", c)
	}

	both := Camera().And(Microphone())
	if !both.hasVideo || !both.hasAudio {
		t.Errorf("Camera().And(Microphone()) = %+v, want video and audio", both)
	}

	if c := Camera().Front(); c.facingMode != facingUser {
		t.Errorf("Front() facingMode = %q, want %q", c.facingMode, facingUser)
	}
	if c := Camera().Back(); c.facingMode != facingEnvironment {
		t.Errorf("Back() facingMode = %q, want %q", c.facingMode, facingEnvironment)
	}
	if c := Camera().Device("abc"); c.deviceID != "abc" {
		t.Errorf("Device(\"abc\") deviceID = %q, want \"abc\"", c.deviceID)
	}

	if Camera().IsNil() {
		t.Errorf("Camera().IsNil() = true, want false")
	}
	if !(Constraints{}).IsNil() {
		t.Errorf("Constraints{}.IsNil() = false, want true")
	}
}

func testZeroConstraints(t *testing.T) {
	// 5. Constraints{} -> ErrNoMediaRequested
	_, err := Request(Constraints{})
	if err != ErrNoMediaRequested {
		t.Errorf("Request(Constraints{}) err = %v, want %v", err, ErrNoMediaRequested)
	}
}

func testNonWASMStubs(t *testing.T) {
	if isWasm {
		return
	}

	// 11. Under !wasm, every exported function returns ErrNotInBrowser
	_, err := Request(Camera())
	if err != ErrNotInBrowser {
		t.Errorf("Request() under !wasm err = %v, want %v", err, ErrNotInBrowser)
	}

	_, err = Devices()
	if err != ErrNotInBrowser {
		t.Errorf("Devices() under !wasm err = %v, want %v", err, ErrNotInBrowser)
	}

	s := &Stream{}
	if s.Live() {
		t.Errorf("Stream.Live() under !wasm = true, want false")
	}
	if s.HasVideo() {
		t.Errorf("Stream.HasVideo() under !wasm = true, want false")
	}
	if s.HasAudio() {
		t.Errorf("Stream.HasAudio() under !wasm = true, want false")
	}
	s.Stop()
}
