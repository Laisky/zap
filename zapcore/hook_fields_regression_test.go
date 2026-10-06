// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package zapcore_test

import (
	"testing"
	"time"

	"github.com/Laisky/zap"
	"github.com/Laisky/zap/zapcore"
	"github.com/Laisky/zap/zaptest/observer"

	"github.com/stretchr/testify/require"
)

// TestHooksWithFieldsContextOnce verifies every callback receives call-site and
// bound context fields exactly once for eager and lazy child loggers.
func TestHooksWithFieldsContextOnce(t *testing.T) {
	for _, lazy := range []bool{false, true} {
		name := "With"
		if lazy {
			name = "WithLazy"
		}
		t.Run(name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			var received [][]zapcore.Field
			hook := func(_ zapcore.Entry, fields []zapcore.Field) error {
				received = append(received, append([]zapcore.Field(nil), fields...))
				return nil
			}
			logger := zap.New(core, zap.HooksWithFields(hook, hook))
			context := []zapcore.Field{zap.String("bound", "one")}
			if lazy {
				logger = logger.WithLazy(context...)
			} else {
				logger = logger.With(context...)
			}

			callFields := []zapcore.Field{zap.Int("call", 1)}
			logger.Info("entry", callFields...)

			expected := append(append([]zapcore.Field(nil), callFields...), context...)
			require.Len(t, received, 2)
			require.Equal(t, expected, received[0])
			require.Equal(t, expected, received[1])
			require.Equal(t, context, logger.Core().Fields())
			require.Len(t, logs.All(), 1)
			require.Equal(t, map[string]interface{}{"bound": "one", "call": int64(1)}, logs.All()[0].ContextMap())
		})
	}
}

// TestHooksWithFieldsCallbackIsolation verifies scalar field mutation in one
// callback does not alter another callback, call-site fields, or core context.
func TestHooksWithFieldsCallbackIsolation(t *testing.T) {
	for _, bound := range []bool{false, true} {
		name := "call_site_only"
		if bound {
			name = "bound_context"
		}
		t.Run(name, func(t *testing.T) {
			core, _ := observer.New(zap.InfoLevel)
			var received [][]zapcore.Field
			first := func(_ zapcore.Entry, fields []zapcore.Field) error {
				received = append(received, append([]zapcore.Field(nil), fields...))
				for i := range fields {
					fields[i] = zap.String("mutated", "first_callback")
				}
				return nil
			}
			second := func(_ zapcore.Entry, fields []zapcore.Field) error {
				received = append(received, append([]zapcore.Field(nil), fields...))
				return nil
			}
			logger := zap.New(core, zap.HooksWithFields(first, second))
			var context []zapcore.Field
			if bound {
				context = []zapcore.Field{zap.String("bound", "one")}
				logger = logger.With(context...)
			}

			callFields := []zapcore.Field{zap.Int("call", 1)}
			logger.Info("entry", callFields...)

			expected := append([]zapcore.Field{zap.Int("call", 1)}, context...)
			require.Len(t, received, 2)
			require.Equal(t, expected, received[0])
			require.Equal(t, expected, received[1])
			require.Equal(t, []zapcore.Field{zap.Int("call", 1)}, callFields)
			require.Equal(t, context, logger.Core().Fields())
		})
	}
}

// TestFilterPreservesTeeSibling verifies a rejecting filter preserves entries
// accepted by another tee branch regardless of branch order.
func TestFilterPreservesTeeSibling(t *testing.T) {
	for _, rejectingFirst := range []bool{false, true} {
		name := "rejecting_last"
		if rejectingFirst {
			name = "rejecting_first"
		}
		t.Run(name, func(t *testing.T) {
			accepted, acceptedLogs := observer.New(zap.InfoLevel)
			rejected, rejectedLogs := observer.New(zap.InfoLevel)
			rejected = zapcore.RegisterFilter(rejected, func(zapcore.Entry, []zapcore.Field) bool {
				return false
			})
			cores := []zapcore.Core{accepted, rejected}
			if rejectingFirst {
				cores[0], cores[1] = cores[1], cores[0]
			}

			zap.New(zapcore.NewTee(cores...)).Info("accepted-by-sibling")

			require.Len(t, acceptedLogs.All(), 1)
			require.Equal(t, "accepted-by-sibling", acceptedLogs.All()[0].Message)
			require.Empty(t, rejectedLogs.All())
		})
	}
}

// TestHooksWithFieldsOnlyForAcceptedTeeBranch verifies field-aware hooks run only
// when their own branch accepts an entry, even if another tee branch accepts it.
func TestHooksWithFieldsOnlyForAcceptedTeeBranch(t *testing.T) {
	for _, rejection := range []string{"filter", "sampler", "filter_sampler"} {
		t.Run(rejection, func(t *testing.T) {
			for _, rejectingFirst := range []bool{false, true} {
				name := "rejecting_last"
				if rejectingFirst {
					name = "rejecting_first"
				}
				t.Run(name, func(t *testing.T) {
					accepted, acceptedLogs := observer.New(zap.InfoLevel)
					rejected, rejectedLogs := observer.New(zap.InfoLevel)
					var checked int
					if rejection == "filter" {
						rejected = zapcore.RegisterFilter(rejected, func(zapcore.Entry, []zapcore.Field) bool {
							checked++
							return false
						})
					} else {
						rejected = zapcore.NewSamplerWithOptions(rejected, time.Hour, 0, 0,
							zapcore.SamplerHook(func(zapcore.Entry, zapcore.SamplingDecision) {
								checked++
							}))
						if rejection == "filter_sampler" {
							rejected = zapcore.RegisterFilter(rejected, func(zapcore.Entry, []zapcore.Field) bool {
								return true
							})
						}
					}
					var hookCalls int
					rejected = zapcore.RegisterHooksWithFields(rejected, func(zapcore.Entry, []zapcore.Field) error {
						hookCalls++
						return nil
					})
					cores := []zapcore.Core{accepted, rejected}
					if rejectingFirst {
						cores[0], cores[1] = cores[1], cores[0]
					}

					zap.New(zapcore.NewTee(cores...)).Info("accepted-by-sibling")

					require.Len(t, acceptedLogs.All(), 1)
					require.Empty(t, rejectedLogs.All())
					require.Equal(t, 1, checked, "branch checks must execute only once")
					require.Zero(t, hookCalls, "a rejected branch must not run its hook")
				})
			}
		})
	}
}

// TestHooksWithFieldsAcceptedTeeBranch verifies an accepted hooked branch still
// writes and invokes its hook exactly once when another tee branch also accepts.
func TestHooksWithFieldsAcceptedTeeBranch(t *testing.T) {
	for _, hookedFirst := range []bool{false, true} {
		name := "hooked_last"
		if hookedFirst {
			name = "hooked_first"
		}
		t.Run(name, func(t *testing.T) {
			sibling, siblingLogs := observer.New(zap.InfoLevel)
			hooked, hookedLogs := observer.New(zap.InfoLevel)
			var received [][]zapcore.Field
			hooked = zapcore.RegisterHooksWithFields(hooked, func(_ zapcore.Entry, fields []zapcore.Field) error {
				received = append(received, append([]zapcore.Field(nil), fields...))
				return nil
			})
			cores := []zapcore.Core{sibling, hooked}
			if hookedFirst {
				cores[0], cores[1] = cores[1], cores[0]
			}
			logger := zap.New(zapcore.NewTee(cores...)).With(zap.String("bound", "one"))

			logger.Info("accepted-by-both", zap.Int("call", 1))

			require.Len(t, siblingLogs.All(), 1)
			require.Len(t, hookedLogs.All(), 1)
			require.Equal(t, [][]zapcore.Field{{zap.Int("call", 1), zap.String("bound", "one")}}, received)
		})
	}
}
