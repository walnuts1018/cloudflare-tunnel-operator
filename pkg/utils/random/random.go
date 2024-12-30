package random

import (
	"crypto/rand"
	"fmt"
	mathrand "math/rand/v2"
)

const UpperLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const LowerLetters = "abcdefghijklmnopqrstuvwxyz"
const Numbers = "0123456789"
const Symbols = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
const Alphabets = UpperLetters + LowerLetters
const Alphanumeric = Alphabets + Numbers
const AlphanumericSymbols = Alphanumeric + Symbols

type Random interface {
	String(length uint, base string) (string, error)
}

type secure struct{}

func NewSecure() Random {
	return secure{}
}

func (r secure) String(length uint, base string) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random: %w", err)
	}

	var result string
	for _, v := range b {
		result += string(base[int(v)%len(base)])
	}
	return result, nil
}

func (r secure) Byte(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to read random: %w", err)
	}
	return b, nil
}

type dummy struct{}

func NewDummy() Random {
	return dummy{}
}

func (d dummy) String(length uint, base string) (string, error) {
	return "dummy", nil
}

func (d dummy) Byte(length int) ([]byte, error) {
	return []byte("dummy"), nil
}

type insecure struct{}

func NewInsecure() Random {
	return insecure{}
}

func (r insecure) String(length uint, base string) (string, error) {
	runes := []rune(base)
	result := make([]rune, length)
	for i := range result {
		result[i] = runes[mathrand.IntN(len(runes))]
	}
	return string(result), nil
}
