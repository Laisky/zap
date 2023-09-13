// Copyright (c) 2017 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Laisky/zap/zapcore"
	. "github.com/Laisky/zap/zapcore"
	"go.uber.org/multierr"
)

type errTooManyUsers int

func (e errTooManyUsers) Error() string {
	return fmt.Sprintf("%d too many users", int(e))
}

func (e errTooManyUsers) Format(s fmt.State, verb rune) {
	// Implement fmt.Formatter, but don't add any information beyond the basic
	// Error method.
	if verb == 'v' && s.Flag('+') {
		io.WriteString(s, e.Error())
	}
}

type customMultierr struct{}

func (e customMultierr) Error() string {
	return "great sadness"
}

func (e customMultierr) Errors() []error {
	return []error{
		errors.New("foo"),
		nil,
		multierr.Append(
			errors.New("bar"),
			errors.New("baz"),
		),
	}
}

func TestErrorEncoding(t *testing.T) {
	tests := []struct {
		k     string
		t     FieldType // defaults to ErrorType
		iface interface{}
		want  map[string]interface{}
	}{
		{
			k:     "k",
			iface: errTooManyUsers(2),
			want: map[string]interface{}{
				"k": "2 too many users",
			},
		},
		{
			k: "err",
			iface: multierr.Combine(
				errors.New("foo"),
				errors.New("bar"),
				errors.New("baz"),
			),
			want: map[string]interface{}{
				"err": "foo; bar; baz",
				"errCauses": []interface{}{
					map[string]interface{}{"error": "foo"},
					map[string]interface{}{"error": "bar"},
					map[string]interface{}{"error": "baz"},
				},
			},
		},
		{
			k:     "e",
			iface: customMultierr{},
			want: map[string]interface{}{
				"e": "great sadness",
				"eCauses": []interface{}{
					map[string]interface{}{"error": "foo"},
					map[string]interface{}{
						"error": "bar; baz",
						"errorCauses": []interface{}{
							map[string]interface{}{"error": "bar"},
							map[string]interface{}{"error": "baz"},
						},
					},
				},
			},
		},
		{
			k:     "k",
			iface: fmt.Errorf("failed: %w", errors.New("egad")),
			want: map[string]interface{}{
				"k": "failed: egad",
			},
		},
		{
			k: "error",
			iface: multierr.Combine(
				fmt.Errorf("hello: %w",
					multierr.Combine(errors.New("foo"), errors.New("bar")),
				),
				errors.New("baz"),
				fmt.Errorf("world: %w", errors.New("qux")),
			),
			want: map[string]interface{}{
				"error": "hello: foo; bar; baz; world: qux",
				"errorCauses": []interface{}{
					map[string]interface{}{
						"error": "hello: foo; bar",
					},
					map[string]interface{}{"error": "baz"},
					map[string]interface{}{"error": "world: qux"},
				},
			},
		},
	}

	for _, tt := range tests {
		if tt.t == UnknownType {
			tt.t = ErrorType
		}

		enc := NewMapObjectEncoder()
		f := Field{Key: tt.k, Type: tt.t, Interface: tt.iface}
		f.AddTo(enc)
		assert.Equal(t, tt.want, enc.Fields, "Unexpected output from field %+v.", f)
	}
}

func TestRichErrorSupport(t *testing.T) {
	f := Field{
		Type:      ErrorType,
		Interface: fmt.Errorf("failed: %w", errors.New("egad")),
		Key:       "k",
	}
	enc := NewMapObjectEncoder()
	f.AddTo(enc)
	assert.Equal(t, "failed: egad", enc.Fields["k"], "Unexpected basic error message.")
}

func TestErrArrayBrokenEncoder(t *testing.T) {
	t.Parallel()

	f := Field{
		Key:  "foo",
		Type: ErrorType,
		Interface: multierr.Combine(
			errors.New("foo"),
			errors.New("bar"),
		),
	}

	failWith := errors.New("great sadness")
	enc := NewMapObjectEncoder()
	f.AddTo(brokenArrayObjectEncoder{
		Err:           failWith,
		ObjectEncoder: enc,
	})

	// Failure to add the field to the encoder
	// causes the error to be added as a string field.
	assert.Equal(t, "great sadness", enc.Fields["fooError"],
		"Unexpected error message.")
}

// brokenArrayObjectEncoder is an ObjectEncoder
// that builds a broken ArrayEncoder.
type brokenArrayObjectEncoder struct {
	ObjectEncoder
	ArrayEncoder

	Err error // error to return
}

func (enc brokenArrayObjectEncoder) AddArray(key string, marshaler ArrayMarshaler) error {
	return enc.ObjectEncoder.AddArray(key,
		ArrayMarshalerFunc(func(ae ArrayEncoder) error {
			enc.ArrayEncoder = ae
			return marshaler.MarshalLogArray(enc)
		}))
}

func (enc brokenArrayObjectEncoder) AppendObject(zapcore.ObjectMarshaler) error {
	return enc.Err
}
