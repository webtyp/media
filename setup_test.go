//go:build !wasm

package media

import (
	"testing"
)

const isWasm = false

func TestAll(t *testing.T) {
	runTests(t)
}
