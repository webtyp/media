# media

Typed camera, microphone and media device access for WebTyp — getUserMedia, device enumeration and typed permission errors

## Status

Under construction. The API is specified in [docs/PLAN.md](docs/PLAN.md);
nothing is implemented yet.

```go
stream, err := media.Request(media.Camera().And(media.Microphone()))
if err == media.ErrPermissionDenied {
    // the user said no
}
defer stream.Stop()
stream.AttachTo(videoEl)
```

Requires a secure context: the camera and microphone are unavailable over plain
HTTP on any origin other than `localhost`. `webtyp dev` serves HTTPS.
