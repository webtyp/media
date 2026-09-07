//go:build wasm

package media

import (
	"syscall/js"
	"testing"

	"webtyp.com/dom"
	"webtyp.com/fmt"
)

const isWasm = true

func TestAll(t *testing.T) {
	runTests(t)
	testWASMSuite(t)
}

func testWASMSuite(t *testing.T) {
	win := js.Global()
	doc := win.Get("document")

	// Helper to set or modify properties safely on JS objects
	setJSProperty := func(target js.Value, prop string, val js.Value) {
		objCls := win.Get("Object")
		desc := objCls.New()
		desc.Set("value", val)
		desc.Set("writable", true)
		desc.Set("configurable", true)
		objCls.Call("defineProperty", target, prop, desc)
	}

	// Save existing navigator and isSecureContext
	origNav := win.Get(propNavigator)
	origSec := win.Get(propIsSecureContext)

	defer func() {
		setJSProperty(win, propNavigator, origNav)
		setJSProperty(win, propIsSecureContext, origSec)
	}()

	rejectPromise := func(errName string) js.Value {
		var executor js.Func
		executor = js.FuncOf(func(this js.Value, args []js.Value) any {
			reject := args[1]
			var cb js.Func
			cb = js.FuncOf(func(this js.Value, cbArgs []js.Value) any {
				errObj := win.Get("Object").New()
				errObj.Set("name", errName)
				reject.Invoke(errObj)
				cb.Release()
				return nil
			})
			win.Call("setTimeout", cb, 0)
			executor.Release()
			return nil
		})
		return win.Get("Promise").New(executor)
	}

	resolvePromise := func(val js.Value) js.Value {
		var executor js.Func
		executor = js.FuncOf(func(this js.Value, args []js.Value) any {
			resolve := args[0]
			var cb js.Func
			cb = js.FuncOf(func(this js.Value, cbArgs []js.Value) any {
				resolve.Invoke(val)
				cb.Release()
				return nil
			})
			win.Call("setTimeout", cb, 0)
			executor.Release()
			return nil
		})
		return win.Get("Promise").New(executor)
	}

	// Test 6: Each of the six DOMException names -> exact typed error.
	domExceptions := []struct {
		domName string
		wantErr error
	}{
		{"NotAllowedError", ErrPermissionDenied},
		{"NotFoundError", ErrNoDevice},
		{"NotReadableError", ErrDeviceBusy},
		{"OverconstrainedError", ErrConstraintsUnsatisfiable},
		{"SecurityError", ErrInsecureContext},
		{"AbortError", ErrAborted},
	}

	for _, tc := range domExceptions {
		nameToTest := tc.domName
		wantToTest := tc.wantErr
		t.Run(nameToTest, func(t *testing.T) {
			fakeGetUserMedia := js.FuncOf(func(this js.Value, args []js.Value) any {
				return rejectPromise(nameToTest)
			})

			fakeMediaDevices := win.Get("Object").New()
			fakeMediaDevices.Set(propGetUserMedia, fakeGetUserMedia)

			fakeNav := win.Get("Object").New()
			fakeNav.Set(propMediaDevices, fakeMediaDevices)

			setJSProperty(win, propNavigator, fakeNav)
			setJSProperty(win, propIsSecureContext, js.ValueOf(true))

			_, err := Request(Camera())
			fakeGetUserMedia.Release()

			if nameToTest == "SecurityError" {
				if err == nil || !fmt.Contains(fmt.Convert(err.Error()).String(), "secure context") {
					t.Errorf("Request() for SecurityError err = %v, want ErrInsecureContext msg", err)
				}
			} else if err != wantToTest {
				t.Errorf("Request() for %s err = %v, want %v", nameToTest, err, wantToTest)
			}
		})
	}

	// Test 7: isSecureContext == false -> ErrInsecureContext, platform never called, message contains origin.
	t.Run("InsecureContext", func(t *testing.T) {
		called := false
		fakeGetUserMedia := js.FuncOf(func(this js.Value, args []js.Value) any {
			called = true
			return resolvePromise(js.Null())
		})

		fakeMediaDevices := win.Get("Object").New()
		fakeMediaDevices.Set(propGetUserMedia, fakeGetUserMedia)

		fakeNav := win.Get("Object").New()
		fakeNav.Set(propMediaDevices, fakeMediaDevices)

		setJSProperty(win, propNavigator, fakeNav)
		setJSProperty(win, propIsSecureContext, js.ValueOf(false))

		_, err := Request(Camera())
		fakeGetUserMedia.Release()

		if called {
			t.Errorf("Request() called platform in insecure context")
		}
		currOrigin := getOrigin(win)
		if err == nil || (currOrigin != "" && !fmt.Contains(fmt.Convert(err.Error()).String(), currOrigin)) {
			t.Errorf("Request() insecure err = %v, want origin in msg", err)
		}
	})

	// Test 8: AttachTo a <div> -> error, not a silent no-op.
	t.Run("AttachToNonMedia", func(t *testing.T) {
		body := doc.Get("body")
		divEl := doc.Call("createElement", "div")
		divEl.Set("id", "test-div-id")
		body.Call("appendChild", divEl)
		defer body.Call("removeChild", divEl)

		s := newStream(win.Get("Object").New())
		el := dom.NewElement("div")
		el.SetID("test-div-id")

		err := s.AttachTo(*el)

		if err != ErrElementNotMedia {
			t.Errorf("AttachTo(div) err = %v, want ErrElementNotMedia", err)
		}
	})

	// Test 9: Stop twice -> no panic; Live() false after first.
	t.Run("StopTwice", func(t *testing.T) {
		stopCalledCount := 0
		track := win.Get("Object").New()
		track.Set("readyState", "live")
		stopFn := js.FuncOf(func(this js.Value, args []js.Value) any {
			stopCalledCount++
			track.Set("readyState", "ended")
			return nil
		})
		defer stopFn.Release()
		track.Set("stop", stopFn)

		tracks := win.Get("Array").New()
		tracks.Call("push", track)

		streamVal := win.Get("Object").New()
		getTracksFn := js.FuncOf(func(this js.Value, args []js.Value) any {
			return tracks
		})
		defer getTracksFn.Release()
		streamVal.Set("getTracks", getTracksFn)

		s := newStream(streamVal)
		if !s.Live() {
			t.Errorf("Stream.Live() before Stop = false, want true")
		}

		s.Stop()
		if s.Live() {
			t.Errorf("Stream.Live() after Stop = true, want false")
		}
		if stopCalledCount != 1 {
			t.Errorf("track.stop() called %d times, want 1", stopCalledCount)
		}

		// Stop second time
		s.Stop()
		if stopCalledCount != 1 {
			t.Errorf("track.stop() called %d times after second Stop, want 1", stopCalledCount)
		}
	})

	// Test 10: Devices() with permission not yet granted -> entries returned with empty Label.
	t.Run("DevicesWithoutPermission", func(t *testing.T) {
		fakeEnumerateDevices := js.FuncOf(func(this js.Value, args []js.Value) any {
			dev1 := win.Get("Object").New()
			dev1.Set("kind", "videoinput")
			dev1.Set("deviceId", "cam1")
			dev1.Set("label", "")

			dev2 := win.Get("Object").New()
			dev2.Set("kind", "audioinput")
			dev2.Set("deviceId", "mic1")
			dev2.Set("label", "")

			devList := win.Get("Array").New()
			devList.Call("push", dev1)
			devList.Call("push", dev2)

			return resolvePromise(devList)
		})

		fakeMediaDevices := win.Get("Object").New()
		fakeMediaDevices.Set(propEnumerateDevs, fakeEnumerateDevices)

		fakeNav := win.Get("Object").New()
		fakeNav.Set(propMediaDevices, fakeMediaDevices)

		setJSProperty(win, propNavigator, fakeNav)

		devs, err := Devices()
		fakeEnumerateDevices.Release()

		if err != nil {
			t.Fatalf("Devices() err = %v", err)
		}
		if len(devs) != 2 {
			t.Fatalf("Devices() returned %d devices, want 2", len(devs))
		}
		if devs[0].Kind != KindCamera || devs[0].ID != "cam1" || devs[0].Label != "" {
			t.Errorf("Devices()[0] = %+v, want camera cam1 with empty label", devs[0])
		}
		if devs[1].Kind != KindMicrophone || devs[1].ID != "mic1" || devs[1].Label != "" {
			t.Errorf("Devices()[1] = %+v, want microphone mic1 with empty label", devs[1])
		}
	})
}
