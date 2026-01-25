package slogx

import (
	"log/slog"
)

// DeferValue implements slog.LogValuer, used to defer [args|attrs] evaluation.
// Helpful when you won't know whether Level is enabled before emit record(s).
// In other words: [slog.Value] on demand.
type DeferValue func() slog.Value

var _ slog.LogValuer = DeferValue(nil)

// A LogValuer is any Go value that can convert itself into a Value for logging.
//
// This mechanism may be used to defer expensive operations until they are needed,
// or to expand a single value into a sequence of components.
func (fn DeferValue) LogValue() slog.Value {
	if fn != nil {
		return fn()
	}
	return slog.Value{} // nil
}
