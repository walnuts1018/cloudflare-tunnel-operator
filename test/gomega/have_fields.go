package gomega

import (
	"fmt"
	"reflect"

	"dario.cat/mergo"
	"github.com/google/go-cmp/cmp"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HaveFieldsGomegaMatcher[T any] struct {
	expected T
}

func HaveFields[T any](expected T) types.GomegaMatcher {
	return &HaveFieldsGomegaMatcher[T]{
		expected: expected,
	}
}

type timeTransformer struct {
}

func (t timeTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ == reflect.TypeOf(metav1.Time{}) {
		return func(dst, src reflect.Value) error {
			if dst.CanSet() {
				isZero := dst.FieldByName("Time").MethodByName("IsZero")
				result := isZero.Call([]reflect.Value{})
				if result[0].Bool() {
					dst.Set(src)
				}
			}
			return nil
		}
	}
	return nil
}

func (a HaveFieldsGomegaMatcher[T]) merged(actual T) T {
	merged := a.expected
	if err := mergo.Merge(&merged, actual, mergo.WithTransformers(timeTransformer{})); err != nil {
		panic(err)
	}
	return merged
}

func (a HaveFieldsGomegaMatcher[T]) Match(actual any) (success bool, err error) {
	t, ok := actual.(T)
	if !ok {
		return false, nil
	}
	return gomega.Equal(a.merged(t)).Match(actual)
}

func (a HaveFieldsGomegaMatcher[T]) FailureMessage(actual any) (message string) {
	t, ok := actual.(T)
	if !ok {
		return "Type assertion failed"
	}
	diff := cmp.Diff(actual, a.merged(t))
	return fmt.Sprintf("diff: \n%s", diff)
}

func (a HaveFieldsGomegaMatcher[T]) NegatedFailureMessage(actual any) (message string) {
	return a.FailureMessage(actual)
}
