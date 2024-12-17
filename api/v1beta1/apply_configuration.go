package v1beta1

import (
	"encoding/json"
)

type WithDeepCopy[T any] struct {
	raw *T
}

func (a WithDeepCopy[T]) DeepCopy() WithDeepCopy[T] {
	var out WithDeepCopy[T]
	bytes, err := json.Marshal(a.raw)
	if err != nil {
		panic("Failed to marshal")
	}

	if err := json.Unmarshal(bytes, &out); err != nil {
		panic("Failed to unmarshal")
	}

	return out
}

func (a WithDeepCopy[T]) Raw() *T {
	return a.raw
}

type WithDeepCopyList[T any] []WithDeepCopy[T]

func (a WithDeepCopyList[T]) Raw() []*T {
	out := make([]*T, 0, len(a))
	for _, item := range a {
		out = append(out, item.raw)
	}
	return out
}
