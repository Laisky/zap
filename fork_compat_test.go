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

package zap

import (
	"testing"

	"github.com/Laisky/zap/zapcore"
	"github.com/Laisky/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Keep the fork's field-aware extensions compatible with eager and lazy context
// and with the upstream level-filter wrapper.
func TestForkExtensionsCompatibility(t *testing.T) {
	for _, lazy := range []bool{false, true} {
		name := "With"
		if lazy {
			name = "WithLazy"
		}
		t.Run(name, func(t *testing.T) {
			core, logs := observer.New(DebugLevel)
			var filtered [][]zapcore.Field
			var hooked [][]zapcore.Field
			logger := New(core,
				Fields(String("base", "one")),
				Filter(func(ent zapcore.Entry, fields []zapcore.Field) bool {
					filtered = append(filtered, append([]zapcore.Field(nil), fields...))
					return ent.Message == "keep"
				}),
				HooksWithFields(func(_ zapcore.Entry, fields []zapcore.Field) error {
					hooked = append(hooked, append([]zapcore.Field(nil), fields...))
					return nil
				}),
				IncreaseLevel(InfoLevel),
			)
			child := logger.With(String("child", "two"))
			if lazy {
				child = logger.WithLazy(String("child", "two"))
			}
			child.Info("drop", String("call", "three"))
			child.Info("keep", String("call", "three"))

			expectedContext := []zapcore.Field{String("base", "one"), String("child", "two")}
			require.Len(t, filtered, 2)
			for _, fields := range filtered {
				assert.Equal(t, expectedContext, fields)
			}
			require.Len(t, hooked, 1)
			assert.Equal(t, append([]zapcore.Field{String("call", "three")}, expectedContext...), hooked[0])
			assert.Equal(t, expectedContext, child.Core().Fields())
			assert.Equal(t, []zapcore.Field{String("base", "one")}, logger.Core().Fields())
			require.Len(t, logs.All(), 1)
			assert.Equal(t, "keep", logs.All()[0].Message)
			assert.Equal(t, map[string]interface{}{"base": "one", "child": "two", "call": "three"}, logs.All()[0].ContextMap())
		})
	}
}
