package controller

import (
	"crypto/sha1"
	"math/big"
)

// sha1 hash an arbitrary string and represent it using the full range of
// printable ascii characters (less space)
func shortName(arbitrary string) string {
	// This array:
	// * Must only include ASCII characters
	// * Must be at least of length 85 (`len("weave-") + l(2^160)/l(85)` equals 31, the maximum ipset name length
	// * Must not include commas as those are treated specially by `ipset add` when adding a set name to a list:set
	// * Should not include space for readability
	// * Should not include single quote or backslash to be nice to shell users
	symbols := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789(){}[]<>_$%^&*|/?.;:@#~")
	sum := sha1.Sum([]byte(arbitrary))
	i := big.NewInt(0).SetBytes(sum[:])
	base := big.NewInt(int64(len(symbols)))

	result := make([]byte, 0)

	// TODO pad to generate constant length strings
	// TODO should this predicate be `while i != 0`?
	for i.Cmp(base) >= 0 {
		remainder := new(big.Int).Mod(i, base)
		i.Sub(i, remainder)
		i.Div(i, base)
		result = append(result, symbols[remainder.Int64()])
	}

	return string(result)
}
