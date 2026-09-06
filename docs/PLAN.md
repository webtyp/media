---
PLAN: "feat: typed camera and microphone access for WebTyp"
EXECUTOR: jules
REVIEWER: none
STATUS: running
SESSION: 93705424840277936
---

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

## Prerequisite — install the test runner

External agents run in isolated environments where `gotest` is not installed.
Run this **before anything else**; the acceptance criteria depend on it:

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

Then use `gotest` for the whole suite and `gotest -run TestName` for one test.
Never call `go test` directly: `gotest` handles `-vet`, `-race`, `-cover`, the
WASM suite and the README badges.

# Plan — `webtyp.com/media`

## Context (the executing agent has none — read this fully)

This repository is new and empty apart from a placeholder `media.go`, which this
plan replaces entirely.

It gives WebTyp applications access to the camera and microphone. WebTyp
compiles Go to WebAssembly and runs it in the browser, so this package wraps
`navigator.mediaDevices` — `getUserMedia`, `enumerateDevices`, and the
`MediaStream` lifecycle.

### The rules of this ecosystem — violating any of these fails review

- **This package compiles to WASM.** Never import the Go standard library. Use
  `webtyp.com/fmt` in place of `fmt`, `errors`, `strconv` and `strings`;
  `webtyp.com/json` in place of `encoding/json`.
- **`syscall/js` is required here, and that is correct.** The ecosystem rule
  "never use `syscall/js`" governs **application** code. A capability library
  that wraps a browser API has no alternative, and the ecosystem's own libraries
  do it: `webtyp.com/fetch` (`client_wasm.go`) imports it and drives
  `then`/`catch` directly. `webtyp.com/jsvalue` is **not** a general bridge — it
  is a codec that marshals Go values to and from `js.Value`, with no way to call
  a method or await a promise. Use `jsvalue` to convert the constraints object
  and the device list; use `syscall/js` to reach `navigator.mediaDevices`.
- **Await promises with the channel bridge, not with a spin loop.** TinyGo's
  WASM target uses the `asyncify` scheduler, so a goroutine blocking on a
  channel yields to the JS event loop and resumes correctly. The pattern is
  `js.FuncOf` callbacks sending into a channel; it is proven in
  `webtyp.com/indexdb`, which is the reference implementation to copy.
- **`webtyp.com/dom` is the only DOM access library** — for `AttachTo`, reach
  the element through `dom`, never by re-querying the document.
- **Avoid `map` in WASM code** (skill: wasm) — it inflates the binary. Use
  structs or slices for small collections.
- **Embed `dom.Element` as a value, never as a pointer.** Pointer embeds double
  the heap allocation under TinyGo's GC.
- **No generic holes.** No `func(...any)`, no `interface{}` in the public API.
  Methods are typed by intent.
- **Minimal surface.** Export only what an application calls.

### Why this is its own repository

Every browser capability that is not DOM manipulation already lives in its own
repository: `webtyp.com/fetch` (network), `webtyp.com/js` (workers),
`webtyp.com/indexdb` (storage), `webtyp.com/webauthn` (credentials). Media
capture follows that pattern. It also keeps the code out of every application's
WASM binary: `dom` is imported by every WebTyp app, and an app with no camera
must not pay for one.

Depending on `webtyp.com/dom` from here is correct and expected; the reverse
would not be.

## Design gate

**1. Prior art.** The web platform exposes one entry point,
`navigator.mediaDevices.getUserMedia(constraints)`, returning a promise of a
`MediaStream`. Wrappers converge on the same three moves: request with
constraints, attach the stream to an element, stop the tracks. React libraries
(`react-webcam`), Vue (`vue-web-cam`) and the WebRTC adapter all keep that
shape. There is no reason to invent a different model, and every reason not to:
a developer who knows the platform must recognise this API immediately.

Where wrappers consistently fail, and where this one earns its place: the
failure modes. `getUserMedia` rejects with a `DOMException` whose `name` is one
of `NotAllowedError`, `NotFoundError`, `NotReadableError`,
`OverconstrainedError`, `SecurityError` or `AbortError`. Almost every wrapper
passes that string through untouched, so every application re-invents the same
string comparison and gets the cases wrong.

**2. Novice-name test.** `media.Request(media.Camera())` reads as "request the
camera". `stream.Stop()` stops it. Rejected: `GetUserMedia` (transliterates the
platform instead of stating intent), `Capture` (ambiguous with taking a photo),
`Open`/`Close` (file vocabulary for a device that can be revoked by the user at
any moment).

**3. Ledger.**

```
Concepts to learn                   +3   (Request, Stream, Constraints)
Repositories in the org             +1
Bytes in an app that uses no media   0   (separate module; nothing is linked)
Error cases handled by hand         −6   (six DOMException names become typed values)
Ways to access the camera           +1   (from 0 — new capability)
```

**4. Where it belongs.** Media capture is one concern and owns this repository.

**5. What it deletes.** The placeholder `media.go` and its `Media` type.

## Stage 1 — constraints and the request

Replace `media.go` entirely.

```go
// Constraints declares what to capture. Build it with Camera(), Microphone(),
// or both; never construct it as a literal from outside this package.
type Constraints struct { /* unexported fields */ }

// Camera requests video from the default camera.
func Camera() Constraints

// Microphone requests audio from the default microphone.
func Microphone() Constraints

// And combines two requests: media.Camera().And(media.Microphone()).
func (c Constraints) And(other Constraints) Constraints

// Front and Back select the facing mode on devices that have both. They are
// a preference, not a guarantee: a device with one camera returns it either way.
func (c Constraints) Front() Constraints
func (c Constraints) Back() Constraints

// Device pins the request to one device id from Devices().
func (c Constraints) Device(id string) Constraints

// Request asks the user for access and resolves to a live Stream.
//
// The browser shows a permission prompt the first time. The returned error is
// always one of the typed errors below.
func Request(c Constraints) (*Stream, error)
```

`Constraints` builds the JavaScript constraints object through
`webtyp.com/jsvalue`. A zero `Constraints` requests nothing and `Request`
returns `ErrNoMediaRequested` rather than calling the platform — asking for
nothing is a programming error, not a runtime outcome.

## Stage 2 — typed errors

This is the reason the package exists. Each is a distinct exported value, not a
string to compare.

| Exported error | `DOMException.name` | Meaning for the developer |
|---|---|---|
| `ErrPermissionDenied` | `NotAllowedError` | the user refused, or the page is not allowed |
| `ErrNoDevice` | `NotFoundError` | no camera or microphone exists |
| `ErrDeviceBusy` | `NotReadableError` | the hardware is held by another application |
| `ErrConstraintsUnsatisfiable` | `OverconstrainedError` | no device matches what was asked |
| `ErrInsecureContext` | `SecurityError` | the page is not a secure context |
| `ErrAborted` | `AbortError` | the request was cancelled |
| `ErrNoMediaRequested` | — | `Request` was called with zero constraints |

`ErrInsecureContext` carries the diagnostic the platform does not give. Its
message must name the actual origin and the way out:

```
media: camera and microphone require a secure context; this page is served over
http from <origin>. Run `webtyp dev`, which serves HTTPS, or install the
development certificate on the device.
```

Read the origin through `webtyp.com/dom`; do not hardcode it. This message is
the difference between a junior losing an afternoon and losing a minute — it is
a feature of the package, not a nicety.

Detect the insecure case **before** calling the platform where possible
(`window.isSecureContext` is false), so the failure is immediate and specific
rather than a rejected promise with a vague name.

## Stage 3 — the stream

```go
// Stream is a live capture. It stays live until Stop, or until the user revokes
// access from the browser UI.
type Stream struct { /* unexported */ }

// AttachTo binds the stream to a media element so it renders. The element must
// be a <video> for video, or an <audio> for audio-only.
func (s *Stream) AttachTo(el dom.Element) error

// Stop ends every track and releases the hardware. Safe to call twice.
//
// A stream that is not stopped keeps the camera light on after the view is
// gone: call this from the component's teardown.
func (s *Stream) Stop()

// Live reports whether any track is still running. It becomes false when the
// user revokes access, which no callback in this API announces.
func (s *Stream) Live() bool

// HasVideo and HasAudio report what the browser actually granted, which can be
// less than what was requested.
func (s *Stream) HasVideo() bool
func (s *Stream) HasAudio() bool
```

`AttachTo` sets `srcObject` through `webtyp.com/jsvalue`. It returns an error
when the element is not a media element, rather than failing silently with a
blank rectangle.

## Stage 4 — device enumeration

```go
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
//
// Labels are empty until the user has granted access once — the browser hides
// them to prevent fingerprinting. Call Request first if the labels matter.
func Devices() ([]Device, error)
```

`Kind` starts at 1 so the zero value is not a valid kind: an uninitialised
`Device` cannot pass as a camera.

## Constraints

- **No hardcoded strings.** Every `DOMException` name, every JS property name
  (`srcObject`, `getUserMedia`, `enumerateDevices`) and every message is a named
  constant.
- **No stdlib replacements skipped.** `webtyp.com/fmt` for `fmt`/`errors`/
  `strings`/`strconv`, `webtyp.com/json` for `encoding/json`, `webtyp.com/time`
  for `time`. `math` is **not** on that list and is allowed — but this package
  needs none.
- **`syscall/js` only inside `//go:build wasm` files**, and only to reach
  `navigator.mediaDevices`. Everything that crosses the Go/JS boundary as data
  goes through `webtyp.com/jsvalue`.
- The whole public API is `//go:build wasm`. A `//go:build !wasm` stub must
  exist so a project's server-side build still compiles: every function returns
  `ErrNotInBrowser`. Without it, importing this package breaks every consumer's
  `go vet` and stdlib tests.

## Tests

Follow the WASM/stdlib dual pattern from skill **testing**: a shared runner
called from a `//go:build wasm` file and a `//go:build !wasm` file, with one
`setup_test.go`.

Against a fake `mediaDevices` injected through `jsvalue`:

1. `Camera()` → constraints object with `video: true`, no `audio` key.
2. `Camera().And(Microphone())` → both keys true.
3. `Camera().Front()` → `facingMode: "user"`; `.Back()` → `"environment"`.
4. `Device("abc")` → `deviceId: {exact: "abc"}`.
5. `Constraints{}` → `ErrNoMediaRequested`, and the platform is never called.
6. Each of the six `DOMException` names → its exact typed error. Table-driven;
   this is the core test of the package.
7. `isSecureContext == false` → `ErrInsecureContext`, platform never called, and
   the message contains the origin.
8. `AttachTo` a `<div>` → error, not a silent no-op.
9. `Stop` twice → no panic; `Live()` false after the first.
10. `Devices()` with permission not yet granted → entries returned with empty
    `Label`.
11. Under `!wasm`, every exported function returns `ErrNotInBrowser` and the
    package compiles.

## Acceptance criteria

1. `grep -rnE '"(fmt|errors|strings|strconv|encoding/json|time)"' --include='*.go' .` → empty.
2. `grep -rn "syscall/js" --include='*.go' .` → present **only** in files tagged
   `//go:build wasm`. It must never appear in the `!wasm` stubs.
3. `go build ./... && go vet ./...` → clean under both `!wasm` and `wasm`.
4. `gotest` passes, including the WASM suite.
5. Test 6 covers all six names.
6. `grep -rn "type Media struct" .` → empty (the placeholder is gone).

## Stages

| # | Stage | File(s) | Gate |
|---|---|---|---|
| 1 | constraints + `Request` | `media.go`, `constraints.go` | tests 1–5 |
| 2 | typed errors | `errors.go` | tests 6, 7 |
| 3 | `Stream` | `stream.go` | tests 8, 9 |
| 4 | devices | `devices.go` | test 10 |
| 5 | `!wasm` stubs | `media_nowasm.go` | test 11, criterion 3 |

Sequential.

## Out of scope

- `MediaRecorder` (recording to a file). A separate concern, and nothing needs
  it yet.
- Screen capture (`getDisplayMedia`).
- WebRTC peer connections.

Adding any of them to this plan is scope creep: report the need, do not build it.
