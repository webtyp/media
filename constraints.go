package media

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

func (c Constraints) toMap() map[string]any {
	m := make(map[string]any)
	if c.hasVideo {
		if c.facingMode != "" || c.deviceID != "" {
			videoSpec := make(map[string]any)
			if c.facingMode != "" {
				videoSpec[propFacingMode] = c.facingMode
			}
			if c.deviceID != "" {
				videoSpec[propDeviceID] = map[string]any{propExact: c.deviceID}
			}
			m[propVideo] = videoSpec
		} else {
			m[propVideo] = true
		}
	}
	if c.hasAudio {
		if c.deviceID != "" {
			audioSpec := make(map[string]any)
			audioSpec[propDeviceID] = map[string]any{propExact: c.deviceID}
			m[propAudio] = audioSpec
		} else {
			m[propAudio] = true
		}
	}
	return m
}
