package cache

import (
	"encoding/hex"
	"fmt"
	"log/slog"
)

// default level at which [cache] logs it's activity ...
const debugLog = (slog.LevelDebug) // - 4) // TRACE

// slog.LogValuer as a function, used to defer evaluation.
// Helpful when you won't know whether Level is enabled before emit.
// In other words: [slog.Value] on demand.
type deferValue func() slog.Value

var _ slog.LogValuer = deferValue(nil)

// A LogValuer is any Go value that can convert itself into a Value for logging.
//
// This mechanism may be used to defer expensive operations until they are needed,
// or to expand a single value into a sequence of components.
func (fn deferValue) LogValue() slog.Value {
	if fn != nil {
		return fn()
	}
	return slog.Value{} // nil
}

// func slogKeys(vs []any) slog.Value {
// 	// display: primary ; + secondary count
// 	if n := len(vs); n > 1 {
// 		// vs = []any{vs[0], fmt.Sprintf("%+d", (n - 1))}
// 		rv := slogValue(vs[0]).Resolve()
// 		switch rv.Kind() {
// 		case slog.KindString:
// 			{
// 				return slog.StringValue(fmt.Sprintf(
// 					"[%s;%+d]", rv.String(), (n - 1),
// 				))
// 			}
// 		}
// 	}

// 	return slogValue(vs)
// }

func slogKeys(vs []any) slog.Value {
	keys := make([]slog.Attr, 0, len(vs))
	for _, v := range vs {
		keys = append(keys, slog.String(
			fmt.Sprintf("%T", v), fmt.Sprintf("%+v", v),
		))
	}
	return slog.GroupValue(keys...)
}

func slogValue(v any) slog.Value {
	if e, is := v.(slog.LogValuer); is {
		return e.LogValue()
	}
	switch e := v.(type) {
	case slog.Value:
		return e
	case []any:
		{
			n := len(e)
			m := make([]any, n)
			for i := range n {
				m[i] = slogValue(e[i])
			}
			v = m
		}
	case string:
		return slog.StringValue(e)
	case [16]byte:
		return slog.StringValue(hex.EncodeToString(e[:]))
	case map[string]any:
		{
			n := len(e)
			if n == 0 {
				return slog.Value{} // nil
			}
			var (
				i  int
				w  *slog.Attr
				vs = make([]slog.Attr, n)
			)
			for h, v := range e {
				w = &vs[i]
				w.Key = h
				w.Value = slogValue(v)
			}
			return slog.GroupValue(vs...)
		}
	}
	// return fmt.Sprintf("%v", v)
	return slog.AnyValue(v)
}
