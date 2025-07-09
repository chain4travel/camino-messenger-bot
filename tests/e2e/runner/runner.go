// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	Test[T any] interface {
		Setup(T)
		Run(*testing.T)
	}
	BeforeRunFunc[T any] func(*testing.T, Test[T])
	AfterRunFunc[T any]  func(*testing.T, T)
)

// Creates a new runner. [T] type will be created with beforeRun func, passed to run func and then to afterRun func.
func New[T any](
	beforeRun BeforeRunFunc[T],
	afterRun AfterRunFunc[T],
	filter []string,
) *Runner[T] {
	return &Runner[T]{
		beforeRun:  beforeRun,
		afterRun:   afterRun,
		tests:      make(map[string]Test[T]),
		testFilter: filter,
	}
}

// Not safe for concurrent use.
type Runner[T any] struct {
	beforeRun  BeforeRunFunc[T]
	afterRun   AfterRunFunc[T]
	tests      map[string]Test[T]
	testFilter []string
}

func (r *Runner[T]) Register(t *testing.T, name string, test Test[T]) {
	if len(r.testFilter) > 0 && !slices.Contains(r.testFilter, name) {
		return
	}

	_, ok := r.tests[name]
	require.False(t, ok)
	r.tests[name] = test
}

func (r *Runner[T]) Run(t *testing.T) {
	for name, test := range r.tests {
		t.Run(name, func(t *testing.T) {
			var tt T

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, tt)
				}
			})

			if r.beforeRun != nil {
				r.beforeRun(t, test)
			}
			test.Run(t)
		})
	}
}

func (r *Runner[T]) RunParallel(t *testing.T) {
	for name, test := range r.tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var tt T

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, tt)
				}
			})

			if r.beforeRun != nil {
				r.beforeRun(t, test)
			}
			test.Run(t)
		})
	}
}
