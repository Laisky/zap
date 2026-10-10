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

import "github.com/Laisky/zap/zapcore"

// Quoted stores a field's JSON representation as a string, for consumers that
// require structured data inside a string-valued log property. Built-in JSON
// and console encoders honor their time, duration and reflection settings;
// other encoders use JSON's default production time and duration formats.
// Verbose and causal error properties are retained as JSON strings under
// their usual suffixed keys. Evaluation remains lazy. Applying Quoted again is a no-op, as is quoting Skip.
// Namespace and Inline describe scopes rather than values; quoting either
// reports a normal "<key>Error" field instead of changing the surrounding scope.
func Quoted(val Field) Field {
	if val.Type == zapcore.SkipType {
		return val
	}
	if _, ok := val.Interface.(*quotedField); ok && val.Type == zapcore.InlineMarshalerType {
		return val
	}
	return Field{Key: val.Key, Type: zapcore.InlineMarshalerType, Interface: &quotedField{val}}
}

type quotedField struct {
	field Field
}

func (q *quotedField) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return zapcore.AddQuotedField(enc, q.field)
}
