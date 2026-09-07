//go:build wasm

package media

import (
	"syscall/js"

	"webtyp.com/dom"
	"webtyp.com/fmt"
)

// Stream is a live capture. It stays live until Stop, or until the user revokes
// access from the browser UI.
type Stream struct {
	value   js.Value
	stopped bool
}

func newStream(v js.Value) *Stream {
	return &Stream{value: v}
}

// AttachTo binds the stream to a media element so it renders. The element must
// be a <video> for video, or an <audio> for audio-only.
func (s *Stream) AttachTo(el dom.Element) error {
	id := el.GetID()
	if id == "" {
		return ErrElementNotMedia
	}
	doc := js.Global().Get("document")
	if doc.IsUndefined() || doc.IsNull() {
		return ErrNotInBrowser
	}
	domEl := doc.Call("getElementById", id)
	if domEl.IsUndefined() || domEl.IsNull() {
		return ErrElementNotMedia
	}

	tagVal := domEl.Get(propTagName)
	if tagVal.IsUndefined() || tagVal.IsNull() {
		return ErrElementNotMedia
	}
	tag := fmt.ToUpper(tagVal.String())
	if tag != tagVideo && tag != tagAudio {
		return ErrElementNotMedia
	}

	domEl.Set(propSrcObject, s.value)
	return nil
}

// Stop ends every track and releases the hardware. Safe to call twice.
func (s *Stream) Stop() {
	if s == nil || s.stopped || s.value.IsUndefined() || s.value.IsNull() {
		if s != nil {
			s.stopped = true
		}
		return
	}
	tracks := s.value.Call("getTracks")
	length := tracks.Get("length").Int()
	for i := 0; i < length; i++ {
		track := tracks.Index(i)
		track.Call("stop")
	}
	s.stopped = true
}

// Live reports whether any track is still running.
func (s *Stream) Live() bool {
	if s == nil || s.stopped || s.value.IsUndefined() || s.value.IsNull() {
		return false
	}
	tracks := s.value.Call("getTracks")
	length := tracks.Get("length").Int()
	for i := 0; i < length; i++ {
		track := tracks.Index(i)
		readyState := track.Get("readyState").String()
		if readyState == "live" {
			return true
		}
	}
	return false
}

// HasVideo reports whether video is active/granted.
func (s *Stream) HasVideo() bool {
	if s == nil || s.value.IsUndefined() || s.value.IsNull() {
		return false
	}
	videoTracks := s.value.Call("getVideoTracks")
	return videoTracks.Get("length").Int() > 0
}

// HasAudio reports whether audio is active/granted.
func (s *Stream) HasAudio() bool {
	if s == nil || s.value.IsUndefined() || s.value.IsNull() {
		return false
	}
	audioTracks := s.value.Call("getAudioTracks")
	return audioTracks.Get("length").Int() > 0
}
