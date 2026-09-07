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
