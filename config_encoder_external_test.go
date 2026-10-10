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
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Laisky/zap"
	"github.com/Laisky/zap/zapcore"
)

var configEncoderSequence uint64

// TestConfigBuildEncoderCustomCore exercises the public API used by consumers
// that supply their own in-memory core instead of Config's output paths.
func TestConfigBuildEncoderCustomCore(t *testing.T) {
	for _, encoding := range []string{"json", "console"} {
		t.Run(encoding, func(t *testing.T) {
			cfg := zap.NewProductionConfig()
			cfg.Encoding = encoding
			cfg.EncoderConfig.TimeKey = ""
			cfg.EncoderConfig.LevelKey = ""
			cfg.EncoderConfig.MessageKey = "message"
			cfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
			output := t.TempDir() + "/must-not-open.log"
			cfg.OutputPaths = []string{output}
			cfg.ErrorOutputPaths = []string{"not-registered://must-not-open"}
			cfg.Level = zap.AtomicLevel{} // encoder construction needs no level or sinks
			enc, err := cfg.BuildEncoder()
			if err != nil {
				t.Fatal(err)
			}
			var sink bytes.Buffer
			logger := zap.New(zapcore.NewCore(enc, zapcore.AddSync(&sink), zap.DebugLevel))
			logger.Info("custom core", zap.Duration("elapsed", time.Second), zap.String("value", "quoted \" text"))
			if got := sink.String(); !strings.Contains(got, "\"elapsed\":") ||
				!strings.Contains(got, "\"1s\"") || !strings.Contains(got, "quoted \\\" text") {
				t.Fatalf("configured encoder lost field settings: %q", got)
			}
			if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("encoder construction opened an output path: %v", err)
			}
		})
	}
}

// TestConfigBuildEncoderErrors preserves the factory's validation and errors.
func TestConfigBuildEncoderErrors(t *testing.T) {
	for _, cfg := range []zap.Config{
		{Encoding: ""},
		{Encoding: "not-registered"},
		{Encoding: "json", EncoderConfig: zapcore.EncoderConfig{TimeKey: "time"}},
	} {
		enc, err := cfg.BuildEncoder()
		if err == nil || enc != nil {
			t.Fatalf("invalid encoder configuration returned %v, %v", enc, err)
		}
	}
}

// TestConfigBuildEncoderRegistry proves the public method uses registered
// factories and forwards their exact configuration and constructor error.
func TestConfigBuildEncoderRegistry(t *testing.T) {
	name := fmt.Sprintf("%s-%d", t.Name(), atomic.AddUint64(&configEncoderSequence, 1))
	wantErr := errors.New("custom encoder unavailable")
	wantConfig := zapcore.EncoderConfig{MessageKey: "custom-message"}
	calls := 0
	if err := zap.RegisterEncoder(name, func(got zapcore.EncoderConfig) (zapcore.Encoder, error) {
		calls++
		if got.MessageKey != wantConfig.MessageKey {
			t.Errorf("constructor got message key %q", got.MessageKey)
		}
		return nil, wantErr
	}); err != nil {
		t.Fatal(err)
	}
	cfg := zap.Config{Encoding: name, EncoderConfig: wantConfig}
	if enc, err := cfg.BuildEncoder(); enc != nil || !errors.Is(err, wantErr) {
		t.Fatalf("constructor error was not preserved: %v, %v", enc, err)
	}
	if calls != 1 {
		t.Fatalf("constructor invoked %d times", calls)
	}
}
