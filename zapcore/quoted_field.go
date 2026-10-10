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

package zapcore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Laisky/zap/internal/bufferpool"
)

// AddQuotedField writes the JSON representation of one field's value as a
// string. Built-in JSON and console encoders preserve their encoding settings.
// Other ObjectEncoders receive a string using default production time and
// duration formats. ObjectEncoder's interface is intentionally unchanged.
func AddQuotedField(enc ObjectEncoder, field Field) error {
	if encoder, ok := enc.(interface{ addQuotedField(Field) error }); ok {
		return encoder.addQuotedField(field)
	}
	cfg := EncoderConfig{
		EncodeTime:          EpochTimeEncoder,
		EncodeDuration:      SecondsDurationEncoder,
		NewReflectedEncoder: defaultReflectedEncoder,
	}
	temporary := _jsonPool.Get()
	temporary.EncoderConfig = &cfg
	temporary.buf = bufferpool.Get()
	defer releaseQuotedEncoder(temporary)
	return encodeQuotedField(enc, temporary, field)
}

// addQuotedField isolates serialization from the parent encoder so a failed
// marshaler cannot leave an unterminated string or alter namespace state.
func (enc *jsonEncoder) addQuotedField(field Field) error {
	temporary := enc.clone()
	temporary.openNamespaces = 0
	defer releaseQuotedEncoder(temporary)
	return encodeQuotedField(enc, temporary, field)
}

func releaseQuotedEncoder(enc *jsonEncoder) {
	enc.buf.Free()
	putJSONEncoder(enc)
}

func encodeQuotedField(destination ObjectEncoder, temporary *jsonEncoder, field Field) error {
	switch field.Type {
	case SkipType:
		return nil
	case UnknownType, NamespaceType, InlineMarshalerType:
		return fmt.Errorf("cannot quote field type %d: a single named value is required", field.Type)
	}
	key := field.Key
	field.Key = "_value"
	field.AddTo(temporary)
	encoded := make([]byte, 0, temporary.buf.Len()+2)
	encoded = append(encoded, '{')
	encoded = append(encoded, temporary.buf.Bytes()...)
	encoded = append(encoded, '}')
	var values map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &values); err != nil {
		return fmt.Errorf("cannot quote invalid JSON value: %w", err)
	}
	if valueError, ok := values["_valueError"]; ok {
		var message string
		if err := json.Unmarshal(valueError, &message); err != nil {
			return fmt.Errorf("cannot decode quoted field error: %w", err)
		}
		return errors.New(message)
	}
	value, ok := values["_value"]
	if !ok {
		return fmt.Errorf("quoted field did not encode a value")
	}
	destination.AddString(key, string(value))
	// Errors can emit additional verbose or causal properties. Preserve all
	// of them under their usual suffixed keys instead of dropping diagnostics.
	var extras []string
	for name := range values {
		if name != "_value" {
			extras = append(extras, name)
		}
	}
	sort.Strings(extras)
	for _, name := range extras {
		destination.AddString(key+strings.TrimPrefix(name, "_value"), string(values[name]))
	}
	return nil
}
