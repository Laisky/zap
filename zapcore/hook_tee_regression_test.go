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

// TestHooksRespectOwnTeeBranch verifies ordinary hooks run only for entries
// accepted by their wrapped branch, without repeating filters or samplers.
func TestHooksRespectOwnTeeBranch(t *testing.T) {
	for _, mode := range []string{"accepted", "filter", "sampler", "filter_sampler", "level", "nop"} {
		t.Run(mode, func(t *testing.T) {
			for _, hookedFirst := range []bool{false, true} {
				name := "hooked_last"
				if hookedFirst {
					name = "hooked_first"
				}
				t.Run(name, func(t *testing.T) {
					sibling, siblingLogs := observer.New(zap.InfoLevel)
					wrapped, wrappedLogs := observer.New(zap.InfoLevel)
					checked := 0
					switch mode {
					case "filter":
						wrapped = zapcore.RegisterFilter(wrapped, func(zapcore.Entry, []zapcore.Field) bool { checked++; return false })
					case "sampler", "filter_sampler":
						wrapped = zapcore.NewSamplerWithOptions(wrapped, time.Hour, 0, 0,
							zapcore.SamplerHook(func(zapcore.Entry, zapcore.SamplingDecision) { checked++ }))
						if mode == "filter_sampler" {
							wrapped = zapcore.RegisterFilter(wrapped, func(zapcore.Entry, []zapcore.Field) bool { return true })
						}
					case "level":
						wrapped, wrappedLogs = observer.New(zap.ErrorLevel)
					case "nop":
						wrapped = zapcore.NewNopCore()
					}
					var hookEntries []zapcore.Entry
					wrapped = zapcore.RegisterHooks(wrapped, func(entry zapcore.Entry) error {
						hookEntries = append(hookEntries, entry)
						return nil
					})
					cores := []zapcore.Core{sibling, wrapped}
					if hookedFirst {
						cores[0], cores[1] = cores[1], cores[0]
					}
					zap.New(zapcore.NewTee(cores...)).With(zap.String("bound", "once")).Info("tee-entry", zap.Int("call", 1))
					require.Len(t, siblingLogs.All(), 1)
					require.Equal(t, "tee-entry", siblingLogs.All()[0].Message)
					if mode == "accepted" {
						require.Len(t, wrappedLogs.All(), 1)
						require.Equal(t, map[string]interface{}{"bound": "once", "call": int64(1)}, wrappedLogs.All()[0].ContextMap())
						require.Len(t, hookEntries, 1)
						require.Equal(t, "tee-entry", hookEntries[0].Message)
					} else {
						require.Empty(t, wrappedLogs.All())
						require.Empty(t, hookEntries, "a sibling's accepted entry must not activate a rejecting branch's hook")
					}
					if mode == "filter" || mode == "sampler" || mode == "filter_sampler" {
						require.Equal(t, 1, checked, "effectful checks must execute only once")
					}
				})
			}
		})
	}
}
