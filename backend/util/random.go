package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const validBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandomString(n int) (string, error) {
	b := strings.Builder{}
	b.Grow(n)
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(validBytes))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		b.WriteByte(validBytes[idx.Int64()])
	}
	return b.String(), nil
}
