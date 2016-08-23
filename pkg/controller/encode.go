package controller

import (
	"crypto/sha1"
	"math/big"
)

// sha1 hash an arbitrary string and represent it using the full range of
// printable ascii characters (less space)
func shortName(arbitrary string) string {
	symbols := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	sum := sha1.Sum([]byte(arbitrary))
	i := big.NewInt(0).SetBytes(sum[:])
	base := big.NewInt(int64(len(symbols)))

	result := make([]byte, 0)

	for i.Cmp(base) >= 0 {
		remainder := new(big.Int).Mod(i, base)
		i.Sub(i, remainder)
		i.Div(i, base)
		result = append(result, symbols[remainder.Int64()])
	}

	return string(result)
}
