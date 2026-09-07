//go:build wasm

package media

import (
	"syscall/js"

	"webtyp.com/await"
	"webtyp.com/jsvalue"
)

func getOrigin(win js.Value) string {
	loc := win.Get(propLocation)
	if loc.IsUndefined() || loc.IsNull() {
		return ""
	}
	origVal := loc.Get(propOrigin)
	if origVal.IsUndefined() || origVal.IsNull() {
		return ""
	}
	return origVal.String()
}

// mediaDevices returns navigator.mediaDevices, or ErrNotInBrowser when the
// current global has no such object.
func mediaDevices(win js.Value) (js.Value, error) {
	nav := win.Get(propNavigator)
	if nav.IsUndefined() || nav.IsNull() {
		return js.Value{}, ErrNotInBrowser
	}
	md := nav.Get(propMediaDevices)
	if md.IsUndefined() || md.IsNull() {
		return js.Value{}, ErrNotInBrowser
	}
	return md, nil
}

// isThenable reports whether v can be awaited as a promise.
func isThenable(v js.Value) bool {
	return !v.IsUndefined() && !v.IsNull() && v.Type() == js.TypeObject
}

// Request asks the user for access and resolves to a live Stream.
func Request(c Constraints) (*Stream, error) {
	if !c.requested() {
		return nil, ErrNoMediaRequested
	}

	win := js.Global()
	if isSec := win.Get(propIsSecureContext); !isSec.IsUndefined() && !isSec.IsNull() && !isSec.Bool() {
		return nil, buildInsecureContextErr(getOrigin(win))
	}

	md, err := mediaDevices(win)
	if err != nil {
		return nil, err
	}

	promise := md.Call(propGetUserMedia, jsvalue.ToJS(c))
	if !isThenable(promise) {
		return nil, ErrNotInBrowser
	}

	streamVal, err := await.Promise(promise)
	if err != nil {
		typed := errFromRejection(err)
		if typed == ErrInsecureContext {
			return nil, buildInsecureContextErr(getOrigin(win))
		}
		return nil, typed
	}

	return newStream(streamVal), nil
}
