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

package zap_test

import (
	"github.com/Laisky/zap"
	"github.com/Laisky/zap/zapcore"
	"testing"
)

type equalityInlineFields []zap.Field

func (fields equalityInlineFields) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for _, field := range fields {
		field.AddTo(enc)
	}
	return nil
}

// TestQuotedFieldEquality preserves the public Field.Equals value contract.
func TestQuotedFieldEquality(t *testing.T) {
	a := zap.Quoted(zap.Dict("value", zap.Int("n", 1)))
	b := zap.Quoted(zap.Dict("value", zap.Int("n", 1)))
	c := zap.Quoted(zap.Dict("value", zap.Int("n", 2)))
	if !a.Equals(b) || !b.Equals(a) {
		t.Fatal("equivalent quoted fields do not compare equal")
	}
	if a.Equals(c) || c.Equals(a) {
		t.Fatal("different quoted values compare equal")
	}
}

// TestInlineFieldEquality checks non-comparable marshalers against the same
// documented deep-equality behavior as object and array marshaler fields.
func TestInlineFieldEquality(t *testing.T) {
	a := zap.Inline(equalityInlineFields{zap.Int("n", 1)})
	b := zap.Inline(equalityInlineFields{zap.Int("n", 1)})
	if !a.Equals(b) || !b.Equals(a) {
		t.Fatal("equivalent inline fields do not compare equal")
	}
	c := zap.Inline(equalityInlineFields{zap.Int("n", 2)})
	if a.Equals(c) || c.Equals(a) {
		t.Fatal("different inline values compare equal")
	}
}
