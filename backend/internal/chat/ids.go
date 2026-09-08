package chat

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return prefix + "_" + hex.EncodeToString(buf)
}

func pairKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
