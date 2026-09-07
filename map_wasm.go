//go:build wasm

package media

import (
	"syscall/js"
)

func mapToJS(m map[string]any) js.Value {
	obj := js.Global().Get("Object").New()
	for k, v := range m {
		switch val := v.(type) {
		case bool:
			obj.Set(k, val)
		case string:
			obj.Set(k, val)
		case map[string]any:
			obj.Set(k, mapToJS(val))
		}
	}
	return obj
}
