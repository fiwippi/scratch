package stringutil

import (
	"math/rand/v2"
	"strings"
)

var encoding = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

func RandString(n int) string {
	var b strings.Builder
	for range n {
		i := rand.IntN(len(encoding))
		b.WriteByte(encoding[i])
	}
	return b.String()
}
