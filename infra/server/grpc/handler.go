package grpc

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	"github.com/webitel/im-account-service/infra/log/slogx"
	"github.com/webitel/im-account-service/internal/handler"
	"github.com/webitel/im-account-service/internal/model"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	rpcpb "github.com/webitel/im-account-service/proto/gen/rpc"
)

type ServiceHandler struct {
	debugLvl slog.Level
	debugLog *slog.Logger
	connSeq  uint64 // conn.id
	callSeq  uint64 // rpc.id
	ctxOpts  []handler.ContextFunc
}

func newServiceHandler(debugLog *slog.Logger) *ServiceHandler {
	if debugLog == nil {
		debugLog = slog.Default()
	}
	h := &ServiceHandler{
		debugLvl: (slog.LevelDebug - 3),
		debugLog: debugLog,
	}
	// const debug4 = slog.LevelDebug - 4
	// if debugLog.Enabled(context.TODO(), debug4) {
	// 	h.debugLog = func(ctx context.Context, message string, params ...any) {
	// 		debugLog.Log(ctx, debug4, message, params...)
	// 	}
	// }
	return h
}

var _ stats.Handler = (*ServiceHandler)(nil)

// ------------------------------- [CONN] ---------------------------------- //

type (
	gRPConnKey struct{}
	gRPCallKey struct{}
)

type gRPConnTag struct {
	id uint64
	stats.ConnTagInfo
	date time.Time     // begin
	time time.Duration // duration
}

// TagConn can attach some information to the given context.
// The returned context will be used for stats handling.
// For conn stats handling, the context used in HandleConn for this
// connection will be derived from the context returned.
// For RPC stats handling,
//   - On server side, the context used in HandleRPC for all RPCs on this
//
// connection will be derived from the context returned.
//   - On client side, the context is not derived from the context returned.
func (h *ServiceHandler) TagConn(ctx context.Context, conn *stats.ConnTagInfo) context.Context {
	nextId := atomic.AddUint64(&h.connSeq, 1) // advance sequence
	ctx = context.WithValue(
		ctx, gRPConnKey{}, &gRPConnTag{
			ConnTagInfo: (*conn), // shallowcopy
			date:        model.LocalTime.Now(),
			time:        0,
			id:          nextId,
		},
	)
	return ctx
}

func (h *ServiceHandler) onConnBegin(ctx context.Context, event *stats.ConnBegin) {
	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)

	// buf := buffer.New()
	// defer buf.Free()

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ CONN::BEGIN ]",
		// "host", conn.net.LocalAddr.String(),
		"conn", conn.id,
		"peer", conn.RemoteAddr.String(),
	)
}

func (h *ServiceHandler) onConnEnd(ctx context.Context, event *stats.ConnEnd) {
	// log.Printf(
	//
	//	"=== [ %s ] --X-- [ %s ] ===  %p",
	//	conn.LocalAddr,
	//	conn.RemoteAddr,
	//	ctx,
	//
	// )

	date := time.Now()
	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)
	conn.time = date.Sub(conn.date)

	// buf := buffer.New()
	// defer buf.Free()

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ CONN::END ]",
		// "host", conn.net.LocalAddr.String(),
		"conn", conn.id,
		"peer", conn.RemoteAddr.String(),
		"time", conn.time.Round(time.Millisecond).String(),
	)
}

// HandleConn processes the Conn stats.
func (h *ServiceHandler) HandleConn(ctx context.Context, state stats.ConnStats) {
	switch event := state.(type) {
	case *stats.ConnBegin:
		h.onConnBegin(ctx, event)
	case *stats.ConnEnd:
		h.onConnEnd(ctx, event)
	default:
		h.debugLog.WarnContext(
			ctx, "Unknown grpc/stats.ConnStats",
			"typeOf", fmt.Sprintf("%T", state),
		)
	}
}

// ------------------------------- [RPC] ---------------------------------- //

type gRPCallTag struct {
	id uint64
	*stats.RPCTagInfo

	*stats.InHeader
	*stats.Begin
	*stats.InPayload
	*stats.OutHeader
	*stats.OutPayload
	*stats.OutTrailer
	*stats.End

	attrs []any // []slog.Attr
}

// TagRPC can attach some information to the given context.
// The context used for the rest lifetime of the RPC will be derived from
// the returned context.
func (h *ServiceHandler) TagRPC(ctx context.Context, rpc *stats.RPCTagInfo) context.Context {
	// before
	// date := ad.Local.Now()
	conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)

	nextId := atomic.AddUint64(&h.callSeq, 1)
	call := gRPCallTag{
		id:         nextId,
		RPCTagInfo: rpc,
		attrs: []any{
			// slog.Int64("conn", conn.id),
			// slog.Int64("rpc", seq),
			slog.String("rpc", fmt.Sprintf("%d.%d", conn.id, nextId)),
			slog.String("peer", conn.RemoteAddr.String()),
			slog.String("path", rpc.FullMethodName),
		},
	}

	span := trace.SpanFromContext(ctx)
	traceId := span.SpanContext().TraceID()
	if traceId.IsValid() {
		call.attrs = append(call.attrs,
			slog.String("trace.id", traceId.String()),
		)
	}

	ctx = context.WithValue(
		ctx, gRPCallKey{}, &call,
	)

	// [BIND] Authentication Context !

	// tx, _ := GetContext(ctx) // bind
	// tx.Date = date           // just now !
	// ctx = tx.Context         // chain ...

	tx, err := handler.NewContext(ctx, h.ctxOpts...)
	_ = tx.Init(func(ctx *handler.Context) (_ error) {
		ctx.Logger = cmp.Or(ctx.Logger, h.debugLog).With(call.attrs...)
		device, ok := model.GetDeviceAuthorization(ctx.Context)
		if ok && device.Addr == nil {
			device.Addr = conn.RemoteAddr
		}
		ctx.Device = &device
		return // nil
	})

	if err != nil {
		tx.Error = cmp.Or(tx.Error, err) // critical
	}

	ctx = handler.WithContext(ctx, tx)
	return ctx
}

// HandleRPC processes the RPC stats.
func (h *ServiceHandler) HandleRPC(ctx context.Context, state stats.RPCStats) {
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	_ = call

	switch event := state.(type) {
	case *stats.InHeader:
		h.onRpcHeader(ctx, event)
	case *stats.Begin:
		h.onRpcBegin(ctx, event)
	case *stats.InPayload:
		h.onRpcData(ctx, event)
	case *stats.OutHeader:
		h.onOutHeader(ctx, event)
	case *stats.OutPayload:
		h.onOutData(ctx, event)
	case *stats.OutTrailer:
		h.onOutTrailer(ctx, event)
	case *stats.End:
		h.onRpcEnd(ctx, event)
	default:
		h.debugLog.WarnContext(
			ctx, "Unknown grpc/stats.RPCStats",
			"typeOf", fmt.Sprintf("%T", state),
			// call.attrs...,
		)
	}
}

func (h *ServiceHandler) onRpcHeader(ctx context.Context, event *stats.InHeader) {
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.InHeader = event

	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	// n := len(call.attrs)
	// attrs := make([]any, n, (n + len(event.Header) + 3))
	// copy(attrs, call.attrs)

	// call.attrs = append(call.attrs,
	// 	slog.String("req.path", event.FullMethod),
	// 	// slog.String("req.pack", event.Compression),
	// 	// slog.Int("req.head", event.WireLength),
	// )

	params := call.attrs
	for h, vs := range event.Header {
		switch h {
		case ":authority",
			"content-type":
			continue
		case "authorization",
			"x-webitel-access",
			"x-webitel-device",
			"x-webitel-client":
			{
				if n := len(vs); n > 0 {
					v2 := slices.Clone(vs)
					for e, v := range v2 {
						v2[e] = slogx.SecureString(v)
					}
					vs = v2
				}
			}
		}
		if len(vs) == 1 {
			params = append(params,
				slog.String("req."+h, vs[0]),
			)
		} else {
			params = append(params,
				slog.Any("req."+h, vs),
			)
		}
	}

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ READ::HEAD ]",
		params...,
	)

	// {
	// 	rpc.InHeader = event

	// 	buf := &rpc.dump
	// 	buf.Reset()
	// 	_, _ = fmt.Fprintf(
	// 		buf, "[ READ::HEAD %p ] addr=%s\n"+
	// 			":path: [%s]",
	// 		ctx,
	// 		event.RemoteAddr,
	// 		event.FullMethod,
	// 	)
	// 	// header
	// 	for h, vs := range event.Header {
	// 		_, _ = fmt.Fprintf(
	// 			buf, "\n%s: %v", h, vs,
	// 		)
	// 	}
	// 	buf.WriteString("\n\n")
	// 	log.Printf("\n\n%s", buf.String())
	// }
}

func (h *ServiceHandler) onRpcBegin(ctx context.Context, event *stats.Begin) {
	//	{
	//		rpc.Begin = event
	//		log.Printf(
	//			"\n\n[ CONN::BEGIN %p ] time=%s\n\n",
	//			ctx, event.BeginTime.Format(timeStamp),
	//		)
	//	}

	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.Begin = event

	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ CALL::BEGIN ]",
		call.attrs...,
	)
}

func (h *ServiceHandler) onRpcData(ctx context.Context, event *stats.InPayload) {
	// {
	// 	rpc.InPayload = event
	// 	// Just received NEW DATA of the request parameters
	// 	// Force update current time of undelying context authentication
	// 	// Affects each client streaming request !
	// 	tx, _ := GetContext(ctx) // bind
	// 	tx.Date = event.RecvTime  // just now !

	// 	switch event.Payload.(proto.Message).(type) {
	// 	case *portal.UploadRequest:
	// 		return // Hide logs
	// 	}

	// 	since := rpc.Begin.BeginTime
	// 	if rpc.OutPayload != nil {
	// 		since = rpc.OutPayload.SentTime
	// 	}

	// 	buf := &rpc.dump
	// 	buf.Reset()
	// 	// content
	// 	_, _ = fmt.Fprintf(
	// 		buf, "[ READ::DATA %p ] len=%d time=%s idle=%s\n"+
	// 			// ":len: [%d]\n"+
	// 			"\n",
	// 		ctx, event.CompressedLength,
	// 		event.RecvTime.Format(timeStamp),
	// 		event.RecvTime.Sub(since),
	// 	)
	// 	// payload
	// 	data, _ := event.Payload.(proto.Message)
	// 	if data != nil {
	// 		_, _ = fmt.Fprintf(
	// 			buf, "[%s]:\n%s\n", // emit: \n\n
	// 			data.ProtoReflect().Descriptor().FullName(),
	// 			prototextMarshalWrap(data),
	// 		)
	// 	}
	// 	log.Printf("\n\n%s", buf.String())
	// }

	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.InPayload = event

	// Just received NEW DATA of the request parameters
	// Force update current time of undelying context authentication
	// Affects each client streaming request !
	tx, _ := handler.FromContext(ctx) // bind
	tx.Date = event.RecvTime          // just now !

	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	params := call.attrs
	params = append(params,
		slog.Int("req.size", event.CompressedLength),
	)
	if event.Payload != nil {
		if data, is := event.Payload.(proto.Message); is {
			params = append(params,
				slog.String("req.type", "*"+string(
					data.ProtoReflect().Descriptor().FullName(),
				)),
				slog.String("req.data", protojson.MarshalOptions{
					Multiline:         false,
					Indent:            "",
					AllowPartial:      true,
					UseProtoNames:     true,
					UseEnumNumbers:    false,
					EmitUnpopulated:   false,
					EmitDefaultValues: false,
					Resolver:          nil,
				}.Format(data)),
			)
		} else {
			params = append(params,
				slog.String("req.type", fmt.Sprintf("%T", event.Payload)),
			)
		}
	}

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ READ::DATA ]",
		params...,
	)
}

func (h *ServiceHandler) onOutHeader(ctx context.Context, event *stats.OutHeader) {
	// {
	// 	if in := rpc.InHeader; in != nil {
	// 		if event.FullMethod == "" {
	// 			event.FullMethod = in.FullMethod
	// 		}
	// 		if event.LocalAddr == nil {
	// 			event.LocalAddr = in.LocalAddr
	// 		}
	// 		if event.RemoteAddr == nil {
	// 			event.RemoteAddr = in.RemoteAddr
	// 		}
	// 	}

	// 	rpc.OutHeader = event

	// 	if len(event.Header) == 0 {
	// 		break // nothing
	// 	}

	// 	buf := &rpc.dump
	// 	buf.Reset()
	// 	_, _ = fmt.Fprintf(
	// 		buf, "[ SEND::HEAD %p ] addr=%s"+
	// 			"\n:path: [%s]",
	// 		ctx,
	// 		event.RemoteAddr,
	// 		event.FullMethod,
	// 	)
	// 	// header
	// 	for h, vs := range event.Header {
	// 		_, _ = fmt.Fprintf(
	// 			buf, "\n%s: %v", h, vs,
	// 		)
	// 	}
	// 	buf.WriteString("\n\n")
	// 	log.Printf("\n\n%s", buf.String())
	// }

	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.OutHeader = event

	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	// // n := len(call.attrs)
	// // attrs := make([]any, n, (n + len(event.Header) + 3))
	// // copy(attrs, call.attrs)

	// call.attrs = append(call.attrs,
	// 	slog.String("out.size", event.Compression),
	// )

	params := call.attrs
	for h, vs := range event.Header {
		if len(vs) == 1 {
			params = append(params,
				slog.String("res."+h, vs[0]),
			)
		} else {
			params = append(params,
				slog.Any("res."+h, vs),
			)
		}
	}

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ SEND::HEAD ]",
		params...,
	)
}

func (h *ServiceHandler) onOutData(ctx context.Context, event *stats.OutPayload) {
	// {
	// 	rpc.OutPayload = event
	// 	// recvLast := stat.SentTime
	// 	// if rpc.InPayload != nil {
	// 	// 	recvLast = rpc.InPayload.RecvTime
	// 	// }

	// 	// buf := &rpc.dump
	// 	// buf.Reset()

	// 	// _, _ = fmt.Fprintf(
	// 	// 	buf, "[ SEND::DATA %p ] len=%d time=%s busy=%s\n"+
	// 	// 		// ":len: [%d]\n"+
	// 	// 		"\n",
	// 	// 	ctx, stat.CompressedLength,
	// 	// 	stat.SentTime.Format(timeStamp),
	// 	// 	stat.SentTime.Sub(recvLast),
	// 	// )
	// 	// // payload
	// 	// data, _ := stat.Payload.(proto.Message)
	// 	// if data != nil {
	// 	// 	_, _ = fmt.Fprintf(
	// 	// 		buf, "[%s]:\n%s\n", // emit: \n\n
	// 	// 		data.ProtoReflect().Descriptor().FullName(),
	// 	// 		prototextMarshalWrap(data),
	// 	// 	)
	// 	// } // else {
	// 	// // 	// &github.com/micro/micro/v3/util/codec/bytes.Frame{Data.([]byte)}
	// 	// // 	_, _ = fmt.Fprintf(
	// 	// // 		buf, "[%T] %[1]q\n\n", stat.Payload,
	// 	// // 	)
	// 	// // }
	// 	// log.Printf("\n\n%s", buf.String())
	// }

	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.OutPayload = event

	// [DEBUG-4] Enabled ?
	if !h.debugLog.Enabled(ctx, h.debugLvl) {
		return
	}

	// call.attrs = append(call.attrs,
	// 	slog.String("out.type", fmt.Sprintf("%T", event.Payload)),
	// 	slog.String("out.data", fmt.Sprintf("{%+v}", event.Payload)),
	// 	slog.Int("out.data.size", event.Length),
	// 	slog.Int("out.data.pack", event.CompressedLength),
	// )

	params := call.attrs
	params = append(params,
		slog.Int("res.size", event.CompressedLength),
	)

	if event.Payload != nil {
		if data, is := event.Payload.(proto.Message); is {
			params = append(params,
				slog.String("res.type", "*"+string(
					data.ProtoReflect().Descriptor().FullName(),
				)),
				slog.String("res.data", protojson.MarshalOptions{
					Multiline:         false,
					Indent:            "",
					AllowPartial:      true,
					UseProtoNames:     true,
					UseEnumNumbers:    false,
					EmitUnpopulated:   false,
					EmitDefaultValues: false,
					Resolver:          nil,
				}.Format(data)),
			)
		} else {
			params = append(params,
				slog.String("res.type", fmt.Sprintf("%T", event.Payload)),
			)
		}
	}

	h.debugLog.Log(
		ctx, h.debugLvl,
		"[ SEND::DATA ]",
		params...,
	)
}

func (h *ServiceHandler) onOutTrailer(ctx context.Context, event *stats.OutTrailer) {
	// // rpc.OutTrailer = event
	// call, _ := ctx.Value(grpcCallKey{}).(*callTag)
	// call.OutTrailer = event

	// h.debugLog.Log(
	// 	// ctx, h.debugLvl, // context canceled
	// 	context.TODO(), h.debugLvl,
	// 	"[ SEND::TRAIL ]",
	// 	call.attrs...,
	// )
}

func (h *ServiceHandler) onRpcEnd(ctx context.Context, event *stats.End) {
	// {
	// 	rpc.End = event
	// 	// status
	// 	res, _ := status.FromError(event.Error)

	// 	log.Printf(
	// 		"\n\n[ CONN::END %p ] time=%s spent=%s (%d) %[4]s ; %s\n\n",
	// 		ctx, event.EndTime.Format(timeStamp),
	// 		event.EndTime.Sub(event.BeginTime),
	// 		res.Code(), res.Message(),
	// 	)
	// }

	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.End = event

	status, _ := status.FromError(event.Error)

	// [DEBUG-4] Enabled ?
	var level = h.debugLvl
	if event.Error != nil {
		level = (slog.LevelError + 3)
	}
	if !h.debugLog.Enabled(ctx, level) {
		return
	}

	code := status.Code()
	call.attrs = append(call.attrs,
		slog.Int("grpc.code", int(code)),
		slog.String("grpc.status", code.String()),
	)

	if event.Error != nil {
		call.attrs = append(call.attrs,
			slog.Any("grpc.message", status.Message()),
		)
	}

	// disclose status.message detail(s)
	for _, nested := range status.Details() {
		switch data := nested.(type) {
		case *rpcpb.Error:
			call.attrs = append(call.attrs,
				slog.Any("error.message", data.Message),
			)
		}
	}

	call.attrs = append(call.attrs,
		slog.String("time", event.EndTime.Sub(event.BeginTime).Round(time.Microsecond).String()),
	)

	h.debugLog.Log(
		// ctx, level,
		context.TODO(), level,
		"[ CALL::END ]",
		call.attrs...,
	)
}
