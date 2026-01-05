// Copyright (C) 2022-2026, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package runner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	Test[E any] interface {
		Setup(E)
		Run(*testing.T)
	}
	BeforeRunFunc[E any] func(*testing.T, Test[E]) E
	AfterRunFunc[E any]  func(*testing.T, E)
)

// Creates a new runner. [E] type will be created with beforeRun func passed to afterRun func after run func run.
func New[E any](
	beforeRun BeforeRunFunc[E],
	afterRun AfterRunFunc[E],
) *Runner[E] {
	return &Runner[E]{
		beforeRun: beforeRun,
		afterRun:  afterRun,
		tests:     make(map[string]Test[E]),
	}
}

// Not safe for concurrent use.
type Runner[E any] struct {
	beforeRun BeforeRunFunc[E]
	afterRun  AfterRunFunc[E]
	tests     map[string]Test[E]
}

func (r *Runner[E]) Register(t *testing.T, name string, test Test[E]) {
	_, ok := r.tests[name]
	require.False(t, ok)
	r.tests[name] = test
}

func (r *Runner[E]) Run(t *testing.T) {
	for name, test := range r.tests {
		t.Run(name, func(t *testing.T) {
			var e E

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, e)
				}
			})

			if r.beforeRun != nil {
				e = r.beforeRun(t, test)
			}
			test.Run(t)
		})
	}
}

func (r *Runner[E]) RunParallel(t *testing.T) {
	for name, test := range r.tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var e E

			t.Cleanup(func() {
				if r.afterRun != nil {
					r.afterRun(t, e)
				}
			})

			if r.beforeRun != nil {
				e = r.beforeRun(t, test)
			}
			test.Run(t)
		})
	}
}
