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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Laisky/zap"
	"github.com/Laisky/zap/zapcore"
)

// TestQuotedValues uses independently specified wire values, including binary,
// full-range times, non-finite numbers and nested marshaled/reflected data.
func TestQuotedValues(t *testing.T) {
	special := "quote \" slash \\ newline\n\t🐙"
	values := []struct {
		name  string
		field zap.Field
		want  string
	}{
		{"boolean", zap.Bool("value", true), "true"},
		{"integer", zap.Int64("value", math.MaxInt64), "9223372036854775807"},
		{"unsigned", zap.Uint64("value", math.MaxUint64), "18446744073709551615"},
		{"float", zap.Float64("value", 3.5), "3.5"},
		{"nan", zap.Float64("value", math.NaN()), "\"NaN\""},
		{"infinity", zap.Float64("value", math.Inf(1)), "\"+Inf\""},
		{"binary", zap.Binary("value", []byte{0, 1, 255}), "\"AAH/\""},
		{"string", zap.String("value", special), "\"quote \\\" slash \\\\ newline\\n\\t🐙\""},
		{"invalid utf8", zap.ByteString("value", []byte{0xff}), "\"\\ufffd\""},
		{"complex", zap.Complex128("value", complex(1, 2)), "\"1+2i\""},
		{"duration", zap.Duration("value", time.Second), "\"1s\""},
		{"full time", zap.Time("value", time.Date(2300, 1, 2, 3, 4, 5, 0, time.UTC)), "\"2300-01-02T03:04:05Z\""},
		{"nil", zap.Reflect("value", nil), "null"},
		{"reflection", zap.Reflect("value", map[string]any{"nested": special}), "{\"nested\":\"quote \\\" slash \\\\ newline\\n\\t🐙\"}"},
		{"array", zap.Ints("value", []int{1, 2}), "[1,2]"},
		{"error", zap.NamedError("value", errors.New("failure")), "\"failure\""},
		{"object", zap.Dict("value", zap.String("nested", special), zap.Int("n", 3)), "{\"nested\":\"quote \\\" slash \\\\ newline\\n\\t🐙\",\"n\":3}"},
	}
	for _, tt := range values {
		t.Run(tt.name, func(t *testing.T) {
			cfg := zapcore.EncoderConfig{EncodeDuration: zapcore.StringDurationEncoder, EncodeTime: zapcore.RFC3339TimeEncoder}
			enc := zapcore.NewJSONEncoder(cfg)
			line, err := enc.EncodeEntry(zapcore.Entry{}, []zap.Field{zap.Quoted(tt.field), zap.String("after", "unchanged")})
			if err != nil {
				t.Fatal(err)
			}
			defer line.Free()
			var got map[string]any
			if err := json.Unmarshal(line.Bytes(), &got); err != nil {
				t.Fatalf("invalid outer JSON: %q: %v", line.String(), err)
			}
			value, ok := got["value"].(string)
			if !ok || value != tt.want {
				t.Fatalf("got %q (%T), want %q", got["value"], got["value"], tt.want)
			}
			if got["after"] != "unchanged" {
				t.Fatalf("following field corrupted: %v", got)
			}
			if !json.Valid([]byte(value)) {
				t.Fatalf("invalid embedded JSON: %q", value)
			}
		})
	}
}

// TestQuotedContextAndLaziness checks logger/core integration and ensures
// disabled entries do not evaluate user marshalers or mutate existing fields.
func TestQuotedContextAndLaziness(t *testing.T) {
	for _, console := range []bool{false, true} {
		t.Run(fmt.Sprint(console), func(t *testing.T) {
			calls := 0
			field := zap.Object("payload", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
				calls++
				enc.AddString("text", "safe")
				return nil
			}))
			quoted := zap.Quoted(field)
			if !zap.Quoted(quoted).Equals(quoted) {
				t.Fatal("repeated quoting must be idempotent")
			}
			var sink bytes.Buffer
			cfg := zapcore.EncoderConfig{EncodeDuration: zapcore.StringDurationEncoder}
			var enc zapcore.Encoder = zapcore.NewJSONEncoder(cfg)
			if console {
				enc = zapcore.NewConsoleEncoder(cfg)
			}
			logger := zap.New(zapcore.NewCore(enc, zapcore.AddSync(&sink), zap.InfoLevel))
			logger.Debug("disabled", quoted)
			if calls != 0 {
				t.Fatal("disabled log evaluated the marshaler")
			}
			child := logger.With(zap.Namespace("scope"), quoted)
			child.Info("child", zap.Int("after", 2))
			logger.Info("parent", field)
			lines := strings.Split(strings.TrimSpace(sink.String()), "\n")
			if len(lines) != 2 {
				t.Fatalf("unexpected log lines: %q", sink.String())
			}
			for i, line := range lines {
				var got map[string]any
				if err := json.Unmarshal([]byte(line), &got); err != nil {
					t.Fatal(err)
				}
				if i == 0 {
					scope := got["scope"].(map[string]any)
					payload := scope["payload"].(string)
					var nested map[string]string
					if err := json.Unmarshal([]byte(payload), &nested); err != nil || nested["text"] != "safe" || scope["after"] != float64(2) {
						t.Fatalf("quoted context corrupted: %v", got)
					}
				} else if _, ok := got["payload"].(map[string]any); !ok {
					t.Fatalf("original field was mutated: %v", got)
				}
			}
		})
	}
}

// TestQuotedFailures verifies atomic output and ordinary Error fields rather
// than historical panics or unterminated JSON, across repeated encoder reuse.
func TestQuotedFailures(t *testing.T) {
	marshalErr := errors.New("marshal failed")
	values := []zap.Field{
		zap.Binary("binary", []byte("ok")),
		zap.NamedError("error", marshalErr),
		zap.Namespace("scope"),
		zap.Inline(zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error { enc.AddString("inline", "value"); return nil })),
		zap.Object("bad", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error { enc.AddString("partial", "value"); return marshalErr })),
		zap.Reflect("channel", make(chan int)),
	}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	for _, value := range values {
		line, err := enc.EncodeEntry(zapcore.Entry{}, []zap.Field{zap.Quoted(value), zap.String("after", "safe"), zap.Quoted(zap.Skip())})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(line.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON after %q: %s", value.Key, line.String())
		}
		line.Free()
		if got["after"] != "safe" {
			t.Fatalf("following field was lost: %v", got)
		}
		if value.Key == "binary" || value.Key == "error" {
			if _, ok := got[value.Key].(string); !ok {
				t.Fatalf("valid value failed: %v", got)
			}
		} else {
			if _, ok := got[value.Key]; ok {
				t.Fatalf("failed field emitted partial data: %v", got)
			}
			if _, ok := got[value.Key+"Error"].(string); !ok {
				t.Fatalf("missing diagnostic: %v", got)
			}
		}
	}
}

// TestQuotedThirdPartyEncoder exercises an unchanged ObjectEncoder interface.
func TestQuotedThirdPartyEncoder(t *testing.T) {
	enc := zapcore.NewMapObjectEncoder()
	zap.Quoted(zap.Dict("payload", zap.Int("answer", 42))).AddTo(enc)
	if got := enc.Fields["payload"]; got != "{\"answer\":42}" {
		t.Fatalf("third-party encoder fallback got %v", got)
	}
}

type quotedReflectedEncoder struct {
	writer  io.Writer
	payload string
	err     error
}

func (enc quotedReflectedEncoder) Encode(any) error {
	if _, err := io.WriteString(enc.writer, enc.payload); err != nil {
		return err
	}
	return enc.err
}

// TestQuotedReflectionSettings preserves custom reflection factories and
// rejects their invalid or failed output before changing the parent encoder.
func TestQuotedReflectionSettings(t *testing.T) {
	for _, tt := range []struct {
		name, payload string
		err           error
		success       bool
	}{
		{"custom output", "{\"custom\":true}", nil, true},
		{"invalid output", "invalid-json", nil, false},
		{"failed output", "{\"partial\":true}", errors.New("reflection failed"), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			cfg := zapcore.EncoderConfig{NewReflectedEncoder: func(writer io.Writer) zapcore.ReflectedEncoder {
				calls++
				return quotedReflectedEncoder{writer, tt.payload, tt.err}
			}}
			enc := zapcore.NewJSONEncoder(cfg)
			line, err := enc.EncodeEntry(zapcore.Entry{}, []zap.Field{zap.Quoted(zap.Reflect("value", 42)), zap.Int("after", 7)})
			if err != nil {
				t.Fatal(err)
			}
			defer line.Free()
			var got map[string]any
			if err := json.Unmarshal(line.Bytes(), &got); err != nil {
				t.Fatalf("parent JSON corrupted: %s", line.String())
			}
			if calls != 1 || got["after"] != float64(7) {
				t.Fatalf("lost factory or following field: %d, %v", calls, got)
			}
			if tt.success {
				if got["value"] != tt.payload {
					t.Fatalf("custom reflection settings lost: %v", got)
				}
			} else {
				if _, ok := got["value"]; ok {
					t.Fatalf("partial output escaped the temporary encoder: %v", got)
				}
				if _, ok := got["valueError"].(string); !ok {
					t.Fatalf("missing error diagnostic: %v", got)
				}
			}
		})
	}
}

// TestQuotedNestedEscapes validates each JSON layer independently, including a
// key and value with quotes, controls and invalid UTF-8.
func TestQuotedNestedEscapes(t *testing.T) {
	special := "key\"\n\\"
	value := "value\"\t\\🐙"
	field := zap.Quoted(zap.Dict("outer", zap.Quoted(zap.Dict("inner", zap.String(special, value)))))
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
	line, err := enc.EncodeEntry(zapcore.Entry{}, []zap.Field{field})
	if err != nil {
		t.Fatal(err)
	}
	defer line.Free()
	var outer map[string]string
	if err := json.Unmarshal(line.Bytes(), &outer); err != nil {
		t.Fatal(err)
	}
	var middle map[string]string
	if err := json.Unmarshal([]byte(outer["outer"]), &middle); err != nil {
		t.Fatal(err)
	}
	var inner map[string]string
	if err := json.Unmarshal([]byte(middle["inner"]), &inner); err != nil {
		t.Fatal(err)
	}
	if inner[special] != value {
		t.Fatalf("nested escaping changed data: %#v", inner)
	}
}

// ExampleQuoted demonstrates a structured JSON object stored in a string.
func ExampleQuoted() {
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{MessageKey: "msg"}),
		zapcore.AddSync(os.Stdout), zap.InfoLevel,
	))
	logger.Info("request", zap.Quoted(zap.Dict("payload", zap.Int("attempt", 3))))
	// Output: {"msg":"request","payload":"{\"attempt\":3}"}
}

// TestQuotedConcurrentReuse guards pooled scratch encoders when the same lazy
// field is written concurrently through cloned encoders.
func TestQuotedConcurrentReuse(t *testing.T) {
	var sink bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{}),
		zapcore.Lock(zapcore.AddSync(&sink)), zap.InfoLevel,
	))
	field := zap.Quoted(zap.Dict("payload", zap.String("text", "quoted \" 🐙")))
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 25; j++ {
				logger.Info("ignored", field)
			}
		}()
	}
	workers.Wait()
	lines := strings.Split(strings.TrimSpace(sink.String()), "\n")
	if len(lines) != 100 {
		t.Fatalf("lost writes: %d", len(lines))
	}
	for _, line := range lines {
		var outer map[string]string
		if err := json.Unmarshal([]byte(line), &outer); err != nil {
			t.Fatal(err)
		}
		var inner map[string]string
		if err := json.Unmarshal([]byte(outer["payload"]), &inner); err != nil {
			t.Fatal(err)
		}
		if inner["text"] != "quoted \" 🐙" {
			t.Fatalf("pooled encoder corrupted data: %v", inner)
		}
	}
}

type quotedVerboseError struct{}

func (quotedVerboseError) Error() string { return "failure" }
func (quotedVerboseError) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, "verbose detail")
}

type quotedErrorGroup []error

func (quotedErrorGroup) Error() string     { return "many failures" }
func (g quotedErrorGroup) Errors() []error { return []error(g) }

// TestQuotedErrorDetails preserves the supported diagnostics attached to an
// error rather than retaining only its primary message.
func TestQuotedErrorDetails(t *testing.T) {
	for _, tt := range []struct {
		field        zap.Field
		suffix, want string
	}{
		{zap.NamedError("error", quotedVerboseError{}), "Verbose", "\"verbose detail\""},
		{zap.NamedError("error", quotedErrorGroup{errors.New("first")}), "Causes", "[{\"error\":\"first\"}]"},
	} {
		enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{})
		line, err := enc.EncodeEntry(zapcore.Entry{}, []zap.Field{zap.Quoted(tt.field)})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]string
		if err := json.Unmarshal(line.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		line.Free()
		if got["error"+tt.suffix] != tt.want {
			t.Fatalf("lost error details: %v, want %s", got, tt.want)
		}
	}
}
