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
	"errors"
	"github.com/Laisky/zap/zapcore"
	"testing"
)

// TestConfigBuildEncoderRegistry proves the public method uses registered
// factories and forwards their exact configuration and constructor error.
func TestConfigBuildEncoderRegistry(t *testing.T) {
	testEncoders(func() {
		name := "custom-config-encoder"
		wantErr := errors.New("custom encoder unavailable")
		wantConfig := zapcore.EncoderConfig{MessageKey: "custom-message"}
		calls := 0
		if err := RegisterEncoder(name, func(got zapcore.EncoderConfig) (zapcore.Encoder, error) {
			calls++
			if got.MessageKey != wantConfig.MessageKey {
				t.Errorf("constructor got message key %q", got.MessageKey)
			}
			return nil, wantErr
		}); err != nil {
			t.Fatal(err)
		}
		cfg := Config{Encoding: name, EncoderConfig: wantConfig}
		if enc, err := cfg.BuildEncoder(); enc != nil || !errors.Is(err, wantErr) {
			t.Fatalf("constructor error was not preserved: %v, %v", enc, err)
		}
		if calls != 1 {
			t.Fatalf("constructor invoked %d times", calls)
		}
	})
}
