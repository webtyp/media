//go:build wasm

package media

import (
	"syscall/js"
)

// Device is one camera or microphone.
type Device struct {
	ID    string
	Label string // empty until permission has been granted at least once
	Kind  Kind
}

type Kind uint8

const (
	KindCamera Kind = iota + 1
	KindMicrophone
)

// Devices lists the available capture devices.
func Devices() ([]Device, error) {
	win := js.Global()
	nav := win.Get(propNavigator)
	if nav.IsUndefined() || nav.IsNull() {
		return nil, ErrNotInBrowser
	}
	mediaDevices := nav.Get(propMediaDevices)
	if mediaDevices.IsUndefined() || mediaDevices.IsNull() {
		return nil, ErrNotInBrowser
	}

	type result struct {
		devices []Device
		err     error
	}
	ch := make(chan result, 1)

	var onFulfilled, onRejected js.Func

	onFulfilled = js.FuncOf(func(this js.Value, args []js.Value) any {
		var devs []Device
		if len(args) > 0 {
			devList := args[0]
			length := devList.Get("length").Int()
			for i := 0; i < length; i++ {
				dObj := devList.Index(i)
				dKindStr := dObj.Get("kind").String()
				var k Kind
				if dKindStr == "videoinput" {
					k = KindCamera
				} else if dKindStr == "audioinput" {
					k = KindMicrophone
				} else {
					continue
				}
				devs = append(devs, Device{
					ID:    dObj.Get("deviceId").String(),
					Label: dObj.Get("label").String(),
					Kind:  k,
				})
			}
		}
		ch <- result{devices: devs}
		return nil
	})

	onRejected = js.FuncOf(func(this js.Value, args []js.Value) any {
		var errRes error = ErrPermissionDenied
		if len(args) > 0 {
			errObj := args[0]
			nameVal := errObj.Get("name")
			if !nameVal.IsUndefined() && !nameVal.IsNull() {
				errRes = mapDOMException(nameVal.String())
			}
		}
		ch <- result{err: errRes}
		return nil
	})

	promise := mediaDevices.Call(propEnumerateDevs)
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
	return res.devices, nil
}
