// Copyright (C) 2022-2025, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	runFunc[T any]       func(*testing.T, T)
	beforeRunFunc[T any] func(*testing.T) T
	afterRunFunc[T any]  func(*testing.T, T)
)

// Creates a new runner. [T] type will be created with beforeRun func, passed to run func and then to afterRun func.
func New[T any](
	beforeRun beforeRunFunc[T],
	afterRun afterRunFunc[T],
) *Runner[T] {
	return &Runner[T]{
		beforeRun: beforeRun,
		afterRun:  afterRun,
		funcs:     make(map[string]runFunc[T]),
	}
}

// Not safe for concurrent use.
type Runner[T any] struct {
	beforeRun beforeRunFunc[T]
	afterRun  afterRunFunc[T]
	funcs     map[string]runFunc[T]
}

func (r *Runner[T]) Register(t *testing.T, name string, f runFunc[T]) {
	_, ok := r.funcs[name]
	require.False(t, ok)
	r.funcs[name] = f
}

func (r *Runner[T]) Run(t *testing.T) {
	for name, test := range r.funcs {
		t.Run(name, func(t *testing.T) {
			var tt T

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, tt)
				}
			})

			if r.beforeRun != nil {
				tt = r.beforeRun(t)
			}
			test(t, tt)
		})
	}
}

func (r *Runner[T]) RunParallel(t *testing.T) {
	for name, test := range r.funcs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var tt T

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, tt)
				}
			})

			if r.beforeRun != nil {
				tt = r.beforeRun(t)
			}
			test(t, tt)
		})
	}
}
