//go:build wasm

package media

import (
	"syscall/js"
	"testing"

	"webtyp.com/dom"
	"webtyp.com/fmt"
	"webtyp.com/jsvalue"
)

const isWasm = true

func TestAll(t *testing.T) {
	runTests(t)
	testConstraintsJS(t)
	testWASMSuite(t)
}

// testConstraintsJS covers plan tests 1-4: the JavaScript MediaStreamConstraints
// object produced by webtyp.com/jsvalue from a Constraints value.
func testConstraintsJS(t *testing.T) {
	// 1. Camera() -> { video: true }, no audio key.
	c1 := jsvalue.ToJS(Camera())
	if c1.Get(propVideo).Type() != js.TypeBoolean || !c1.Get(propVideo).Bool() {
		t.Errorf("Camera() video = %v, want true", c1.Get(propVideo))
	}
	if !c1.Get(propAudio).IsUndefined() {
		t.Errorf("Camera() audio key present, want absent")
	}

	// 2. Camera().And(Microphone()) -> both keys true.
	c2 := jsvalue.ToJS(Camera().And(Microphone()))
	if !c2.Get(propVideo).Bool() || !c2.Get(propAudio).Bool() {
		t.Errorf("And() = video:%v audio:%v, want both true", c2.Get(propVideo), c2.Get(propAudio))
	}

	// 3. Front() -> facingMode "user"; Back() -> "environment".
	if got := jsvalue.ToJS(Camera().Front()).Get(propVideo).Get(propFacingMode).String(); got != facingUser {
		t.Errorf("Front() facingMode = %q, want %q", got, facingUser)
	}
	if got := jsvalue.ToJS(Camera().Back()).Get(propVideo).Get(propFacingMode).String(); got != facingEnvironment {
		t.Errorf("Back() facingMode = %q, want %q", got, facingEnvironment)
	}

	// 4. Device("abc") -> deviceId: { exact: "abc" }.
	if got := jsvalue.ToJS(Camera().Device("abc")).Get(propVideo).Get(propDeviceID).Get(propExact).String(); got != "abc" {
		t.Errorf("Device() deviceId.exact = %q, want \"abc\"", got)
	}
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
				// getUserMedia rejects with a DOMException; it stringifies as
				// "<name>: <message>", which is how the name reaches the caller.
				errObj := win.Get("DOMException").New("simulated rejection", errName)
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
