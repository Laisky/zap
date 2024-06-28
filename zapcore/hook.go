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

import "go.uber.org/multierr"

type hooked struct {
	Core
	funcs []func(Entry) error
}

var (
	_ Core           = (*hooked)(nil)
	_ leveledEnabler = (*hooked)(nil)
)

// RegisterHooks wraps a Core and runs a collection of user-defined callback
// hooks each time a message is logged. Execution of the callbacks is blocking.
//
// This offers users an easy way to register simple callbacks (e.g., metrics
// collection) without implementing the full Core interface.
func RegisterHooks(core Core, hooks ...func(Entry) error) Core {
	funcs := append([]func(Entry) error{}, hooks...)
	return &hooked{
		Core:  core,
		funcs: funcs,
	}
}

func (h *hooked) Level() Level {
	return LevelOf(h.Core)
}

func (h *hooked) Check(ent Entry, ce *CheckedEntry) *CheckedEntry {
	// Let the wrapped Core decide whether to log this message or not. This
	// also gives the downstream a chance to register itself directly with the
	// CheckedEntry.
	if downstream := h.Core.Check(ent, ce); downstream != nil {
		return downstream.AddCore(ent, h)
	}
	return ce
}

func (h *hooked) With(fields []Field) Core {
	return &hooked{
		Core:  h.Core.With(fields),
		funcs: h.funcs,
	}
}

func (h *hooked) Write(ent Entry, _ []Field) error {
	// Since our downstream had a chance to register itself directly with the
	// CheckedMessage, we don't need to call it here.
	var err error
	for i := range h.funcs {
		err = multierr.Append(err, h.funcs[i](ent))
	}
	return err
}

type hookedWithFields struct {
	Core
	funcs []func(Entry, []Field) error
}

// RegisterHooksWithFields like RegisterHooks but and invoke hooks with arbitary fileds
func RegisterHooksWithFields(core Core, hooks ...func(Entry, []Field) error) Core {
	return &hookedWithFields{
		Core:  core,
		funcs: hooks[:len(hooks):len(hooks)],
	}
}

func (h *hookedWithFields) Check(ent Entry, ce *CheckedEntry) *CheckedEntry {
	// Let the wrapped Core decide whether to log this message or not. This
	// also gives the downstream a chance to register itself directly with the
	// CheckedEntry.
	if downstream := h.Core.Check(ent, ce); downstream != nil {
		return downstream.AddCore(ent, h)
	}
	return ce
}

func (h *hookedWithFields) With(fields []Field) Core {
	return &hookedWithFields{
		Core:  h.Core.With(fields),
		funcs: h.funcs,
	}
}

func (h *hookedWithFields) Write(ent Entry, fs []Field) error {
	// Since our downstream had a chance to register itself directly with the
	// CheckedMessage, we don't need to call it here.
	var err error
	for i := range h.funcs {
		fs = append(fs[:len(fs):len(fs)], h.Core.Fields()...)
		err = multierr.Append(err, h.funcs[i](ent, fs))
	}
	return err
}

// filter is a Core that wraps another Core and filters messages based on
// user-defined criteria.
type filter struct {
	Core
	filter func(Entry, []Field) bool
}

// RegisterFilter wraps a Core and filters messages based on a user-defined
func RegisterFilter(core Core, userFilter func(Entry, []Field) bool) Core {
	return &filter{
		Core:   core,
		filter: userFilter,
	}
}

// Check calls the underlying Core only if the filter function returns true.
func (f *filter) Check(ent Entry, ce *CheckedEntry) *CheckedEntry {
	if !f.filter(ent, f.Core.Fields()) {
		return nil
	}

	if downstream := f.Core.Check(ent, ce); downstream != nil {
		return downstream.AddCore(ent, f)
	}

	return ce
}

func (f *filter) With(fields []Field) Core {
	return &filter{
		Core:   f.Core.With(fields),
		filter: f.filter,
	}
}

// Write noop
func (f *filter) Write(ent Entry, fields []Field) error {
	return nil
}
