package utils

import (
	"math/rand"
	"time"
)

// ─── Random utilities ───────────────────────────────────────
// Ref: packages/core/src/utils/random.ts

const alphanumChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomAlphaNum generates a random alphanumeric string of given length.
func RandomAlphaNum(length int) string {
	if length <= 0 {
		return ""
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = alphanumChars[rand.Intn(len(alphanumChars))]
	}
	return string(b)
}

// RandomStringSpecial generates a random string with dash and underscore inserted.
// Matches randomString(length, special) from random.ts.
func RandomStringSpecial(length int) string {
	if length <= 0 {
		return ""
	}
	if length < 3 {
		return RandomAlphaNum(length)
	}

	base := RandomAlphaNum(length)
	runes := []rune(base)

	dashPos := rand.Intn(length-2) + 1
	underscorePos := rand.Intn(length-2) + 1
	for dashPos == underscorePos {
		underscorePos = rand.Intn(length-2) + 1
	}

	// Insert dash and underscore
	result := make([]rune, 0, length+2)
	for i, r := range runes {
		if i == dashPos {
			result = append(result, '-')
		}
		if i == underscorePos {
			result = append(result, '_')
		}
		result = append(result, r)
	}

	return string(result)
}

// RandomStringWithPrefix generates "PREFIX-xxxxxxxxxx".
func RandomStringWithPrefix(prefix string, length int) string {
	if prefix == "" {
		prefix = "A"
	}
	if length <= 0 {
		return prefix
	}
	return prefix + "-" + RandomAlphaNum(length)
}

// RandomNumber generates a random numeric string of given length.
func RandomNumber(length int) string {
	if length <= 0 {
		return ""
	}
	if length > 15 {
		length = 15
	}

	digits := "0123456789"
	b := make([]byte, length)

	// First digit: 1-9 (no leading zero for multi-digit)
	if length > 1 {
		b[0] = digits[rand.Intn(9)+1]
		for i := 1; i < length; i++ {
			b[i] = digits[rand.Intn(10)]
		}
	} else {
		b[0] = digits[rand.Intn(10)]
	}

	return string(b)
}
