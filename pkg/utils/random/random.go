package random

import (
	"crypto/rand"
	"fmt"
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
	Byte(length int) ([]byte, error)
}

type random struct{}

func New() Random {
	return random{}
}

func (r random) String(length uint, base string) (string, error) {
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

func (r random) Byte(length int) ([]byte, error) {
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
