package media

import (
	"testing"
)

func runTests(t *testing.T) {
	testConstraints(t)
	testZeroConstraints(t)
	testNonWASMStubs(t)
}

func testConstraints(t *testing.T) {
	// 1. Camera() -> constraints object with video: true, no audio key
	c1 := Camera()
	m1 := c1.toMap()
	if v, ok := m1["video"]; !ok || v != true {
		t.Errorf("Camera() video constraint = %v, want true", v)
	}
	if _, ok := m1["audio"]; ok {
		t.Errorf("Camera() audio constraint present, want absent")
	}

	// 2. Camera().And(Microphone()) -> both keys true
	c2 := Camera().And(Microphone())
	m2 := c2.toMap()
	if v, ok := m2["video"]; !ok || v != true {
		t.Errorf("And() video constraint = %v, want true", v)
	}
	if v, ok := m2["audio"]; !ok || v != true {
		t.Errorf("And() audio constraint = %v, want true", v)
	}

	// 3. Camera().Front() -> facingMode: "user"; .Back() -> "environment"
	c3Front := Camera().Front()
	m3Front := c3Front.toMap()
	v3Front, ok := m3Front["video"].(map[string]any)
	if !ok || v3Front["facingMode"] != "user" {
		t.Errorf("Front() video constraint = %v, want facingMode: user", m3Front["video"])
	}

	c3Back := Camera().Back()
	m3Back := c3Back.toMap()
	v3Back, ok := m3Back["video"].(map[string]any)
	if !ok || v3Back["facingMode"] != "environment" {
		t.Errorf("Back() video constraint = %v, want facingMode: environment", m3Back["video"])
	}

	// 4. Device("abc") -> deviceId: {exact: "abc"}
	c4 := Camera().Device("abc")
	m4 := c4.toMap()
	v4, ok := m4["video"].(map[string]any)
	if !ok {
		t.Fatalf("Device() video constraint not a map: %v", m4["video"])
	}
	devIDMap, ok := v4["deviceId"].(map[string]any)
	if !ok || devIDMap["exact"] != "abc" {
		t.Errorf("Device() deviceId constraint = %v, want {exact: abc}", v4["deviceId"])
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
