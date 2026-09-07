package media

import (
	"webtyp.com/model"
)

// Constraints declares what to capture. Build it with Camera(), Microphone(),
// or both; never construct it as a literal from outside this package.
type Constraints struct {
	hasVideo   bool
	hasAudio   bool
	facingMode string
	deviceID   string
}

// Camera requests video from the default camera.
func Camera() Constraints {
	return Constraints{hasVideo: true}
}

// Microphone requests audio from the default microphone.
func Microphone() Constraints {
	return Constraints{hasAudio: true}
}

// And combines two requests: media.Camera().And(media.Microphone()).
func (c Constraints) And(other Constraints) Constraints {
	res := c
	if other.hasVideo {
		res.hasVideo = true
	}
	if other.hasAudio {
		res.hasAudio = true
	}
	if other.facingMode != "" {
		res.facingMode = other.facingMode
	}
	if other.deviceID != "" {
		res.deviceID = other.deviceID
	}
	return res
}

// Front and Back select the facing mode on devices that have both. They are
// a preference, not a guarantee: a device with one camera returns it either way.
func (c Constraints) Front() Constraints {
	c.facingMode = facingUser
	return c
}

func (c Constraints) Back() Constraints {
	c.facingMode = facingEnvironment
	return c
}

// Device pins the request to one device id from Devices().
func (c Constraints) Device(id string) Constraints {
	c.deviceID = id
	return c
}

// requested reports whether any media was asked for.
func (c Constraints) requested() bool {
	return c.hasVideo || c.hasAudio
}

// EncodeFields writes the JavaScript MediaStreamConstraints object. It is the
// model codec contract consumed by webtyp.com/jsvalue — no map, no reflect.
//
// video/audio are emitted as `true` when unqualified, or as a nested object
// when a facing mode or a device id narrows them.
func (c Constraints) EncodeFields(w model.FieldWriter) {
	if c.hasVideo {
		if c.facingMode != "" || c.deviceID != "" {
			w.Object(propVideo, videoConstraint{facingMode: c.facingMode, deviceID: c.deviceID})
		} else {
			w.Bool(propVideo, true)
		}
	}
	if c.hasAudio {
		if c.deviceID != "" {
			w.Object(propAudio, trackConstraint{deviceID: c.deviceID})
		} else {
			w.Bool(propAudio, true)
		}
	}
}

// IsNil satisfies model.Encodable; a zero Constraints requests nothing.
func (c Constraints) IsNil() bool { return !c.requested() }

// videoConstraint is the nested `video: {...}` object.
type videoConstraint struct {
	facingMode string
	deviceID   string
}

func (v videoConstraint) EncodeFields(w model.FieldWriter) {
	if v.facingMode != "" {
		w.String(propFacingMode, v.facingMode)
	}
	if v.deviceID != "" {
		w.Object(propDeviceID, exactConstraint{id: v.deviceID})
	}
}

func (v videoConstraint) IsNil() bool { return v.facingMode == "" && v.deviceID == "" }

// trackConstraint is the nested `audio: {...}` object.
type trackConstraint struct {
	deviceID string
}

func (t trackConstraint) EncodeFields(w model.FieldWriter) {
	w.Object(propDeviceID, exactConstraint{id: t.deviceID})
}

func (t trackConstraint) IsNil() bool { return t.deviceID == "" }

// exactConstraint is the `deviceId: {exact: "..."}` object.
type exactConstraint struct {
	id string
}

func (e exactConstraint) EncodeFields(w model.FieldWriter) {
	w.String(propExact, e.id)
}

func (e exactConstraint) IsNil() bool { return e.id == "" }
