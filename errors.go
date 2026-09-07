package media

import (
	"webtyp.com/fmt"
)

var (
	ErrPermissionDenied          = fmt.Err("media: permission denied")
	ErrNoDevice                  = fmt.Err("media: no device found")
	ErrDeviceBusy                = fmt.Err("media: device busy")
	ErrConstraintsUnsatisfiable  = fmt.Err("media: constraints unsatisfiable")
	ErrInsecureContext           = fmt.Err("media: camera and microphone require a secure context")
	ErrAborted                   = fmt.Err("media: request aborted")
	ErrNoMediaRequested          = fmt.Err("media: no media requested")
	ErrNotInBrowser              = fmt.Err("media: not in browser environment")
	ErrElementNotMedia           = fmt.Err("media: element is not a media element (<video> or <audio>)")
)

const (
	domErrNotAllowed      = "NotAllowedError"
	domErrNotFound        = "NotFoundError"
	domErrNotReadable     = "NotReadableError"
	domErrOverconstrained = "OverconstrainedError"
	domErrSecurity        = "SecurityError"
	domErrAbort           = "AbortError"
)

func buildInsecureContextErr(origin string) error {
	msg := fmt.Sprintf("media: camera and microphone require a secure context; this page is served over http from %s. Run `webtyp dev`, which serves HTTPS, or install the development certificate on the device.", origin)
	return fmt.Err(msg)
}

func mapDOMException(name string) error {
	switch name {
	case domErrNotAllowed:
		return ErrPermissionDenied
	case domErrNotFound:
		return ErrNoDevice
	case domErrNotReadable:
		return ErrDeviceBusy
	case domErrOverconstrained:
		return ErrConstraintsUnsatisfiable
	case domErrSecurity:
		return ErrInsecureContext
	case domErrAbort:
		return ErrAborted
	default:
		return fmt.Err(fmt.Sprintf("media: %s", name))
	}
}

// errFromRejection maps a promise rejection surfaced by webtyp.com/await to a
// typed media error. A DOMException stringifies as "<name>: <message>", so the
// name is the text before the first ": "; a rejection with no usable name
// (await.ErrRejected) is treated as a permission failure — the safe default the
// platform itself falls back to.
func errFromRejection(err error) error {
	if err == nil {
		return nil
	}
	name := err.Error()
	if i := fmt.Index(name, ": "); i > 0 {
		name = name[:i]
	}
	switch name {
	case domErrNotAllowed, domErrNotFound, domErrNotReadable,
		domErrOverconstrained, domErrSecurity, domErrAbort:
		return mapDOMException(name)
	default:
		return ErrPermissionDenied
	}
}
