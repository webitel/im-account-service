package pg

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/webitel/im-account-service/infra/x/logx"
)

func debugLog(logger *slog.Logger) pgx.QueryTracer {
	// stdlog := logger.With( // slog.Default().With(
	// 	"db.system", "postgresql",
	// )
	if !logx.Debug("postgres", "db") {
		return nil
	}
	stdlog := logx.ModuleLogger("postgres", logger)
	// traces := stdlog.Enabled(
	// 	context.TODO(), (slog.LevelDebug - 4),
	// )
	return &tracelog.TraceLog{
		Config: &tracelog.TraceLogConfig{
			TimeKey: "time",
		},
		LogLevel: tracelog.LogLevelTrace, // tracelogLevel(?)
		Logger: tracelog.LoggerFunc(func(ctx context.Context, v tracelog.LogLevel, event string, data map[string]interface{}) {
			level := slogLevel(v)
			if v >= tracelog.LogLevelInfo {
				level = (slog.LevelDebug) // DEBUG // - 4) // TRACE
			}
			// reduce slogAttrs() convertion
			if !stdlog.Enabled(ctx, level) {
				return
			}
			// stdlog.LogAttrs(ctx, level, ("[ POSTGRES ]: " + event), slogAttrs(data)...)
			stdlog.LogAttrs(ctx, level, event, slogAttrs(data)...)
		}),
	}
}

// converts [log/slog.Level] to [github.com/jackc/pgx/v5/tracelog.LogLevel]
func tracelogLevel(v slog.Level) tracelog.LogLevel {

	// LevelDebug Level = -4
	// LevelInfo  Level = 0
	// LevelWarn  Level = 4
	// LevelError Level = 8

	//     slog.Level((tracelog.LogLevelInfo - v) * 4)
	return tracelog.LogLevelInfo - tracelog.LogLevel((v / 4))

	// LogLevelTrace = LogLevel(6)
	// LogLevelDebug = LogLevel(5)
	// LogLevelInfo  = LogLevel(4)
	// LogLevelWarn  = LogLevel(3)
	// LogLevelError = LogLevel(2)
	// LogLevelNone  = LogLevel(1)
}

// converts [github.com/jackc/pgx/v5/tracelog.LogLevel] to [log/slog.Level]
func slogLevel(v tracelog.LogLevel) slog.Level {

	// LogLevelTrace = LogLevel(6)
	// LogLevelDebug = LogLevel(5)
	// LogLevelInfo  = LogLevel(4)
	// LogLevelWarn  = LogLevel(3)
	// LogLevelError = LogLevel(2)
	// LogLevelNone  = LogLevel(1) // FATAL

	return slog.Level((tracelog.LogLevelInfo - v) * 4)

	// LevelDebug Level = -4
	// LevelInfo  Level = 0
	// LevelWarn  Level = 4
	// LevelError Level = 8
}

// https://opentelemetry.io/docs/specs/semconv/database/sql/
func slogAttrs(data map[string]any) []slog.Attr {
	n := len(data)
	if n == 0 {
		return nil
	}
	var (
		i int
		m = make([]slog.Attr, n)
	)
	for key, value := range data {
		switch key {
		case "err": // error
			key = "error"
		case "sql": // string
			key = "db.query.text"
		case "args": // []any
			key = "db.query.params"
		case "commandTag": // string
			key = "db.query.tag" // "db.operation.name"
			tag := value.(string)
			op, n := tag, len(tag)
			for i := n - 1; i >= 0; i-- {
				if '0' <= op[i] && op[i] <= '9' {
					// n = (n * 10) + int64(op[i] - '0')
					n = i
				} else {
					break
				}
			}
			value = strings.TrimSpace(op[:n])
			if n < len(tag) {
				c, _ := strconv.ParseUint(tag[n:], 10, 64)
				m = append(m, slog.Attr{
					Key:   "db.query.rows", // "db.operation.rows",
					Value: slog.Uint64Value(c),
				})
			}
		// COPY FROM
		case "tableName": // pgx.Identifier.([]string)
			key = "db.copy.from"
		case "columnNames": // []string
			key = "db.copy.cols"
		case "rowCount": // int64
			key = "db.copy.rows"
		// Connect
		case "host":
		case "port":
		case "database":
			// Prepare
		case "name":
		case "alreadyPrepared": // bool
		}
		e := &m[i]
		e.Key = key
		e.Value = slogValue(value)
		i++
	}
	return m
}

func indirect(v any) any {
	switch e := v.(type) {
	case bool:
	case int:
	case int64:
	case uint64:
	case []byte:
		return string(e)
	case string:
	case float64:
	case time.Time:
	case time.Duration:
	case []any:
		{
			// args
		optionLoop:
			for len(e) > 0 {
				switch e[0].(type) {
				case pgx.NamedArgs:
					// break optionLoop
					return indirect(e[0]) // .(pgx.NamedArgs)
				case pgx.QueryRewriter:
				case pgx.QueryExecMode:
				case pgx.QueryResultFormats:
				case pgx.QueryResultFormatsByOID:
				default:
					break optionLoop
				}
				e = e[1:]
				// continue
			}
			for i, v := range e {
				e[i] = indirect(v)
			}
			return e
		}
	case pgx.NamedArgs:
		{
			v2 := make(map[string]any, len(e))
			for param, value := range e {
				v2[param] = indirect(value)
			}
			return v2
		}
	default:
		{
			// error ?
			if e, is := v.(error); is {
				return e
			}
			// // github.com/jackc/pgtype.TextEncoder
			// if e, is := v.(pgtype.TextEncoder); is {
			// 	text, err := e.EncodeText(nil, nil)
			// 	if err == nil {
			// 		return string(text)
			// 	}
			// 	return err
			// }
			// database/sql/driver.Valuer
			if e, is := v.(driver.Valuer); is {
				vs, err := e.Value()
				if err == nil {
					return indirect(vs)
				}
				return err
			}
			if e, is := v.(json.Marshaler); is {
				jsonb, err := e.MarshalJSON()
				if err == nil {
					return string(jsonb)
				}
				return err
			}
			rv := reflect.ValueOf(v)
			if rv.Kind() == reflect.Pointer {
				if rv.IsNil() {
					return nil
				}
				rv = reflect.Indirect(rv)
				return indirect(rv.Interface())
			}
		}
	}
	return v
}

func slogValue(v any) slog.Value {
	v = indirect(v)
	switch e := v.(type) {
	case error:
		return slog.StringValue(e.Error())
	case bool:
		return slog.BoolValue(e)
	case int:
		return slog.IntValue(e)
	case int64:
		return slog.Int64Value(e)
	case uint64:
		return slog.Uint64Value(e)
	case string:
		return slog.StringValue(e)
	case float64:
		return slog.Float64Value(e)
	case time.Time:
		return slog.TimeValue(e)
	case time.Duration:
		// return slog.DurationValue(e)
		return slog.StringValue(
			e.Round(time.Millisecond).String(),
		)
	case pgx.NamedArgs:
		return slog.StringValue(fmt.Sprintf(
			"%v", (map[string]any)(e),
		))
	case slog.Value:
		return e
		// case []any: // args
		// 	{

		// 	}
	}
	return slog.AnyValue(v)
	// log.Printf("%#v", v)
	// return slog.StringValue(
	// 	fmt.Sprintf("%+v", v),
	// )
}
