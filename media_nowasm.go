//go:build !wasm

package media

import (
	"webtyp.com/dom"
)

// Stream is a live capture stub for non-WASM builds.
type Stream struct{}

// AttachTo returns ErrNotInBrowser under non-WASM builds.
func (s *Stream) AttachTo(el dom.Element) error {
	return ErrNotInBrowser
}

// Stop is a no-op stub.
func (s *Stream) Stop() {}

// Live reports false.
func (s *Stream) Live() bool {
	return false
}

// HasVideo reports false.
func (s *Stream) HasVideo() bool {
	return false
}

// HasAudio reports false.
func (s *Stream) HasAudio() bool {
	return false
}

// Device is one camera or microphone.
type Device struct {
	ID    string
	Label string
	Kind  Kind
}

type Kind uint8

const (
	KindCamera Kind = iota + 1
	KindMicrophone
)

// Request returns ErrNotInBrowser under non-WASM builds.
func Request(c Constraints) (*Stream, error) {
	if !c.hasVideo && !c.hasAudio {
		return nil, ErrNoMediaRequested
	}
	return nil, ErrNotInBrowser
}

// Devices returns ErrNotInBrowser under non-WASM builds.
func Devices() ([]Device, error) {
	return nil, ErrNotInBrowser
}
