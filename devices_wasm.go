//go:build wasm

package media

import (
	"syscall/js"

	"webtyp.com/await"
	"webtyp.com/jsvalue"
	"webtyp.com/model"
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

// DecodeFields reads one MediaDeviceInfo through the model codec (webtyp.com/jsvalue).
func (d *Device) DecodeFields(r model.FieldReader) {
	if v, ok := r.String(propDeviceID); ok {
		d.ID = v
	}
	if v, ok := r.String(propLabel); ok {
		d.Label = v
	}
	if v, ok := r.String(propKind); ok {
		d.Kind = kindFromJS(v)
	}
}

// IsNil satisfies model.Decodable.
func (d *Device) IsNil() bool { return d == nil }

// kindFromJS maps a MediaDeviceInfo.kind string to a Kind. Output devices
// ("audiooutput") and anything unrecognised map to the zero Kind.
func kindFromJS(kind string) Kind {
	switch kind {
	case kindVideoInput:
		return KindCamera
	case kindAudioInput:
		return KindMicrophone
	default:
		return 0
	}
}

// Devices lists the available capture devices.
func Devices() ([]Device, error) {
	md, err := mediaDevices(js.Global())
	if err != nil {
		return nil, err
	}

	promise := md.Call(propEnumerateDevs)
	if !isThenable(promise) {
		return nil, ErrNotInBrowser
	}

	list, err := await.Promise(promise)
	if err != nil {
		return nil, errFromRejection(err)
	}

	n := list.Length()
	devs := make([]Device, 0, n)
	for i := 0; i < n; i++ {
		var d Device
		if err := jsvalue.ToGo(list.Index(i), &d); err != nil {
			return nil, err
		}
		if d.Kind == 0 {
			continue // output device or unknown kind
		}
		devs = append(devs, d)
	}
	return devs, nil
}
