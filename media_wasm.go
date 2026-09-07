//go:build wasm

package media

import (
	"syscall/js"
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

// Request asks the user for access and resolves to a live Stream.
func Request(c Constraints) (*Stream, error) {
	if !c.hasVideo && !c.hasAudio {
		return nil, ErrNoMediaRequested
	}

	win := js.Global()
	if isSec := win.Get(propIsSecureContext); !isSec.IsUndefined() && !isSec.IsNull() && !isSec.Bool() {
		return nil, buildInsecureContextErr(getOrigin(win))
	}

	nav := win.Get(propNavigator)
	if nav.IsUndefined() || nav.IsNull() {
		return nil, ErrNotInBrowser
	}
	mediaDevices := nav.Get(propMediaDevices)
	if mediaDevices.IsUndefined() || mediaDevices.IsNull() {
		return nil, ErrNotInBrowser
	}

	constraintsJS := mapToJS(c.toMap())

	type result struct {
		streamVal js.Value
		err       error
	}
	ch := make(chan result, 1)

	var onFulfilled, onRejected js.Func

	onFulfilled = js.FuncOf(func(this js.Value, args []js.Value) any {
		var sVal js.Value
		if len(args) > 0 {
			sVal = args[0]
		}
		ch <- result{streamVal: sVal}
		return nil
	})

	onRejected = js.FuncOf(func(this js.Value, args []js.Value) any {
		var errRes error = ErrPermissionDenied
		if len(args) > 0 {
			errObj := args[0]
			nameVal := errObj.Get("name")
			if !nameVal.IsUndefined() && !nameVal.IsNull() {
				errRes = mapDOMException(nameVal.String())
				if errRes == ErrInsecureContext {
					errRes = buildInsecureContextErr(getOrigin(win))
				}
			}
		}
		ch <- result{err: errRes}
		return nil
	})

	promise := mediaDevices.Call(propGetUserMedia, constraintsJS)
	if promise.IsUndefined() || promise.IsNull() || promise.Type() != js.TypeObject {
		onFulfilled.Release()
		onRejected.Release()
		return nil, ErrNotInBrowser
	}
	promise.Call("then", onFulfilled, onRejected)

	res := <-ch
	onFulfilled.Release()
	onRejected.Release()
	if res.err != nil {
		return nil, res.err
	}

	return newStream(res.streamVal), nil
}
