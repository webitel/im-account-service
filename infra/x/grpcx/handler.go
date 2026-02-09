package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/webitel/im-account-service/infra/log/slogx"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	rpcpb "github.com/webitel/im-account-service/proto/gen/rpc"
)

type DumpOptions struct {

	Debug slog.Level // [DEBUG] slog.Level to use
	Logger *slog.Logger
	Codec ProtoCodec // Codec to marshal grpc [http2] protobuf [data] body
	
	TagRPC func(ctx context.Context, rpc *stats.RPCTagInfo) context.Context
}

type ProtoCodec interface {
	Format(proto.Message) string
}

func (opts *DumpOptions) can(ctx context.Context, level slog.Level) bool {
	return opts.Logger != nil && opts.Logger.Enabled(ctx, level)
}

func (opts *DumpOptions) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	opts.Logger.Log(ctx, level, msg, args...)
}

func (opts *DumpOptions) debug(ctx context.Context, msg string, args ...any) {
	opts.Logger.Log(ctx, opts.Debug, msg, args...)
}

type HandlerOption func(*DumpOptions)
func newHandlerOptions(opts []HandlerOption) DumpOptions {
	options := DumpOptions{
		// Debug: slog.LevelInfo,
		// Logger: slog.New(slog.DiscardHandler),
	}
	for _, setup := range opts {
		setup(&options)
	}
	// Defaults ...
	if options.Logger == nil {
		options.Logger = slog.New(slog.DiscardHandler)
	}
	if options.Codec == nil {
		// options.Codec = protojson.MarshalOptions{
		// 	Multiline:         false,
		// 	Indent:            "",
		// 	AllowPartial:      true,
		// 	UseProtoNames:     true,
		// 	UseEnumNumbers:    false,
		// 	EmitUnpopulated:   false,
		// 	EmitDefaultValues: false,
		// 	Resolver:          nil,
		// }
		options.Codec = prototext.MarshalOptions{
			Multiline:         false,
			Indent:            "",
			EmitASCII:         false,
			AllowPartial:      true,
			EmitUnknown:       true, // !!!
			Resolver:          nil,
		}
	}
	return options
}

// DumpHandler dumps http2 [data] trafic for gRPC client/server.
func DumpHandler(opts ...HandlerOption) stats.Handler {
	return &dumpHandler{
		opts: newHandlerOptions(opts),
	}
}

// LogDump dumpHandler for grpc/stats
type dumpHandler struct {
	opts DumpOptions
	connSeq  uint64 // conn.id
	callSeq  uint64 // rpc.id
}

var _ stats.Handler = (*dumpHandler)(nil)

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
func (h *dumpHandler) TagConn(ctx context.Context, conn *stats.ConnTagInfo) context.Context {
	
	// h.opts.debug(
	// 	ctx, "[ grpc.ConnTag ]",
	// 	"args", fmt.Sprintf("%#v", conn),
	// )
	
	nextId := atomic.AddUint64(&h.connSeq, 1) // advance sequence
	ctx = context.WithValue(
		ctx, gRPConnKey{}, &gRPConnTag{
			ConnTagInfo: (*conn), // shallowcopy
			date:        time.Now(), // model.LocalTime.Now(),
			time:        0,
			id:          nextId,
		},
	)
	return ctx
}

// HandleConn processes the Conn stats.
func (h *dumpHandler) HandleConn(ctx context.Context, state stats.ConnStats) {
	
	// h.opts.debug(
	// 	ctx, "[ grpc.Conn ]",
	// 	"args", fmt.Sprintf("%#v", state),
	// )
	
	switch event := state.(type) {
	case *stats.ConnBegin:
		h.onConnBegin(ctx, event)
	case *stats.ConnEnd:
		h.onConnEnd(ctx, event)
	default:
		h.opts.Logger.WarnContext(
			ctx, "Unknown grpc/stats.ConnStats",
			"typeof", fmt.Sprintf("%T", state),
		)
	}
}

const (
	server = "server"
	client = "client"
)

func (h *dumpHandler) onConnBegin(ctx context.Context, event *stats.ConnBegin) {
	// [DEBUG-4] Enabled ?
	if !h.opts.can(ctx, h.opts.Debug) {
		return
	}

	conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)

	// buf := buffer.New()
	// defer buf.Free()

	h.opts.log(
		ctx, h.opts.Debug,
		"[ CONN::BEGIN ]",
		// "host", conn.net.LocalAddr.String(),
		"part", ternary(event.Client, client, server),
		"conn", conn.id,
		"peer", conn.RemoteAddr.String(),
	)
}

func ternary[T any](cond bool, true, false T) T {
	if cond {	return true }
	return false
}

func (h *dumpHandler) onConnEnd(ctx context.Context, event *stats.ConnEnd) {
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
	if !h.opts.can(ctx, h.opts.Debug) {
		return
	}

	conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)
	conn.time = date.Sub(conn.date)

	h.opts.debug(
		ctx, "[ CONN::END ]",
		// "host", conn.net.LocalAddr.String(),
		"part", ternary(event.Client, client, server),
		"conn", conn.id,
		"peer", conn.RemoteAddr.String(),
		"time", conn.time.Round(time.Millisecond).String(),
	)
}

// ------------------------------- [RPC] ---------------------------------- //

type gRPCallTag struct {
	
	seqId uint64
	stats.RPCTagInfo

	stats.ConnTagInfo

	stats.Begin
	stats.InHeader
	stats.InTrailer
	stats.InPayload
	stats.OutHeader
	stats.OutTrailer
	stats.OutPayload
	stats.End

	attrs []any // []slog.Attr
}

// type record struct {
// 	Out bool
// 	Header metadata.MD
// 	Trailer metadata.MD
// 	Payload any
// }

// TagRPC can attach some information to the given context.
// The context used for the rest lifetime of the RPC will be derived from
// the returned context.
func (h *dumpHandler) TagRPC(ctx context.Context, rpc *stats.RPCTagInfo) context.Context {
	// before
	// date := ad.Local.Now()

	// h.opts.debug(
	// 	ctx, "[ grpc.RpcTag ]",
	// 	"args", fmt.Sprintf("%#v", rpc),
	// )

	nextId := atomic.AddUint64(&h.callSeq, 1)
	call := gRPCallTag{
		RPCTagInfo: (*rpc), // shalowcopy
		seqId:      nextId,
		attrs:      []any{
			// slog.String("call", fmt.Sprintf("%d", nextId)),
			// slog.String("path", rpc.FullMethodName),
		},
	}

	// conn, _ := ctx.Value(gRPConnKey{}).(*gRPConnTag)
	
	// if conn != nil {
	// 	// server received
	// 	call.attrs = append(call.attrs,
	// 		// slog.Int64("conn", conn.id),
	// 		// slog.Int64("rpc", seq),
	// 		slog.String("rpc", fmt.Sprintf("%d.%d", conn.id, nextId)),
	// 		slog.String("peer", conn.RemoteAddr.String()),
	// 		slog.String("path", rpc.FullMethodName),
	// 	)
	// } else {
		// client sending
		// call.attrs = append(call.attrs,
		// 	slog.String("call", fmt.Sprintf("%d", nextId)),
		// 	slog.String("path", rpc.FullMethodName),
		// )
	// }

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

	// tx, err := handler.NewContext(ctx, h.ctxOpts...)
	// _ = tx.Init(func(ctx *handler.Context) (_ error) {
	// 	ctx.Logger = cmp.Or(ctx.Logger, h.debugLog).With(call.attrs...)
	// 	device, ok := model.GetDeviceAuthorization(ctx.Context)
	// 	if ok && device.Addr == nil {
	// 		device.Addr = conn.RemoteAddr
	// 	}
	// 	ctx.Device = &device
	// 	return // nil
	// })

	// if err != nil {
	// 	tx.Error = cmp.Or(tx.Error, err) // critical
	// }

	// ctx = handler.WithContext(ctx, tx)

	hook := h.opts.TagRPC
	if hook != nil {
		ctx = hook(ctx, rpc)
	}

	return ctx
}

// HandleRPC processes the RPC stats.
func (h *dumpHandler) HandleRPC(ctx context.Context, state stats.RPCStats) {
	
	// h.opts.debug(
	// 	ctx, "[ grpc.RPC ]",
	// 	"args", fmt.Sprintf("%#v", state),
	// )
	
	// call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	// _ = call

	switch event := state.(type) {

	case *stats.Begin:
		h.onRpcBegin(ctx, event)
	case *stats.DelayedPickComplete:
		// h.onDelayedPickComplete(ctx, event)
	
	case *stats.InHeader:
		h.onRpcHeader(ctx, event)
	case *stats.InTrailer: // grpc.stream (unary) close !
		h.onRpcTrailer(ctx, event)
	case *stats.InPayload:
		h.onRpcData(ctx, event)

	case *stats.OutHeader:
		h.onOutHeader(ctx, event)
	case *stats.OutTrailer:  // grpc.stream (unary) close !
		h.onOutTrailer(ctx, event)
	case *stats.OutPayload:
		h.onOutData(ctx, event)

	case *stats.End:
		h.onRpcEnd(ctx, event)

	default:
		h.opts.Logger.WarnContext(
			ctx, "Unknown grpc/stats.RPCStats",
			"typeof", fmt.Sprintf("%T", state),
			// call.attrs...,
		)
	}
}

func (h *dumpHandler) onRpcBegin(ctx context.Context, event *stats.Begin) {
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.Begin = (*event) // shallowcopy
	return

	// [DEBUG-4] Enabled ?
	if !h.opts.can(ctx, h.opts.Debug) {
		return
	}

	stream := "unary"
	if event.IsClientStream {
		if event.IsServerStream {
			stream = "bidi"
		} else {
			stream = "client"
		}
	} else if event.IsServerStream {
			stream = "server"
	}

	call.attrs = append(call.attrs,
		slog.String("stream", stream),
		slog.String("part", ternary(event.Client, "client", "server")),
	)

	h.opts.Logger.Log(
		ctx, h.opts.Debug,
		"[ CALL::BEGIN ]",
		call.attrs...,
	)
}

func (h *dumpHandler) onRpcHeader(ctx context.Context, event *stats.InHeader) {
	
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.InHeader = (*event) // shallowcopy

	if !event.Client {
		// request receiving
		call.ConnTagInfo = stats.ConnTagInfo{
			RemoteAddr: event.RemoteAddr,
			LocalAddr:  event.LocalAddr,
		}
	}

	return // silent

	// [DEBUG-4] Enabled ?
	if !h.opts.can(ctx, h.opts.Debug) {
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

	// side := "req." // server side
	// if event.Client {
	// 	side = "res." // client side
	// }

	head := call.attrs

	const group = "head."
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
			head = append(head,
				slog.String(group+h, vs[0]),
			)
		} else {
			head = append(head,
				slog.Any(group+h, vs),
			)
		}
	}

	// h.opts.Logger.Log(
	// 	ctx, h.opts.Debug,
	// 	"[ READ::HEAD ]",
	// 	params...,
	// )

	call.attrs = head

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

func (h *dumpHandler) onRpcTrailer(ctx context.Context, event *stats.InTrailer) {
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.InTrailer = (*event) // shallowcopy
}

func (h *dumpHandler) onRpcData(ctx context.Context, event *stats.InPayload) {
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
	call.InPayload = (*event) // shalowcopy

	// Just received NEW DATA of the request parameters
	// Force update current time of undelying context authentication
	// Affects each client streaming request !
	// tx, _ := handler.FromContext(ctx) // bind
	// tx.Date = event.RecvTime          // just now !

	// [DEBUG-4] Enabled ?
	if !h.opts.can(ctx, h.opts.Debug) {
		return
	}

	// group := "req." // server side
	// if event.Client {
	// 	group = "res." // client side
	// }

	args := call.attrs

	// basic
	args = append(args,
		slog.String("part", ternary(event.Client, client, server)),
		slog.Uint64("call", call.seqId), // local identifier
		slog.String("peer", call.ConnTagInfo.RemoteAddr.String()),
		slog.String("path", call.RPCTagInfo.FullMethodName),
	)

	// kind := "unary"
	// if call.Begin.IsClientStream {
	// 	if call.Begin.IsServerStream {
	// 		kind = "bidi"
	// 	} else {
	// 		kind = "client"
	// 	}
	// } else if call.Begin.IsServerStream {
	// 		kind = "server"
	// }

	// header
	const group = "head."
	for h, vs := range call.InHeader.Header {
		switch h {
		case ":authority":
			// args = append(args,
			// 	slog.String("host", vs[0]),
			// )
			// continue
			h = "host" // HTTP/1.* -like
		case "content-type",
			"accept":
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
			args = append(args,
				slog.String(group+h, vs[0]),
			)
		} else {
			args = append(args,
				slog.Any(group+h, vs),
			)
		}
	}

	// content-length
	args = append(args,
		slog.Int("size", event.CompressedLength),
	)

	// body
	args = append(args,	slog.String("data",
		dataLogString(event.Payload, h.opts.Codec),
	))

	h.opts.debug(
		ctx, "[ READ::DATA ]", args...,
	)
}

func (h *dumpHandler) onOutHeader(ctx context.Context, event *stats.OutHeader) {
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
	call.OutHeader = (*event)

	if event.Client {
		// request sending
		call.ConnTagInfo = stats.ConnTagInfo{
			RemoteAddr: event.RemoteAddr,
			LocalAddr:  event.LocalAddr,
		}
	}

	return // silent

	// [DEBUG-4] Enabled ?
	if !h.opts.Logger.Enabled(ctx, h.opts.Debug) {
		return
	}

	if event.Client {
		// start sending request headers
		call.attrs = append(call.attrs,
			// resolved client (target) address
			"peer", event.RemoteAddr.String(),
			// "seed", event.LocalAddr.String(),
		)
	}

	// // n := len(call.attrs)
	// // attrs := make([]any, n, (n + len(event.Header) + 3))
	// // copy(attrs, call.attrs)

	// call.attrs = append(call.attrs,
	// 	slog.String("out.size", event.Compression),
	// )

	side := "res." // server side
	if event.Client {
		side = "req." // client side
	}

	params := call.attrs
	for h, vs := range event.Header {
		if len(vs) == 1 {
			params = append(params,
				slog.String(side+h, vs[0]),
			)
		} else {
			params = append(params,
				slog.Any(side+h, vs),
			)
		}
	}

	// h.opts.Logger.Log(
	// 	ctx, h.opts.Debug,
	// 	"[ SEND::HEAD ]",
	// 	params...,
	// )

	call.attrs = params // & header
}

func (h *dumpHandler) onOutTrailer(ctx context.Context, event *stats.OutTrailer) {
	
	call, _ := ctx.Value(gRPCallKey{}).(*gRPCallTag)
	call.OutTrailer = (*event)
	return // silent
	
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

func (h *dumpHandler) onOutData(ctx context.Context, event *stats.OutPayload) {
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
	call.OutPayload = (*event)

	// [DEBUG-4] Enabled ?
	if !h.opts.can(ctx, h.opts.Debug) {
		return
	}

	// call.attrs = append(call.attrs,
	// 	slog.String("out.type", fmt.Sprintf("%T", event.Payload)),
	// 	slog.String("out.data", fmt.Sprintf("{%+v}", event.Payload)),
	// 	slog.Int("out.data.size", event.Length),
	// 	slog.Int("out.data.pack", event.CompressedLength),
	// )

	args := call.attrs

	// basic
	args = append(args,
		slog.String("part", ternary(event.Client, client, server)),
		slog.Uint64("call", call.seqId), // local identifier
		slog.String("peer", call.ConnTagInfo.RemoteAddr.String()),
		slog.String("path", call.RPCTagInfo.FullMethodName),
	)

	// header
	const group = "head."
	for h, vs := range call.OutHeader.Header {
		switch h {
		case ":authority":
			// args = append(args,
			// 	slog.String("host", vs[0]),
			// )
			// continue
			h = "host" // HTTP/1.* -like
		case "content-type",
			"accept":
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
			args = append(args,
				slog.String(group+h, vs[0]),
			)
		} else {
			args = append(args,
				slog.Any(group+h, vs),
			)
		}
	}

	args = append(args,
		slog.Int("size", event.CompressedLength),
	)

	// group := "req." // server side
	// if event.Client {
	// 	group = "res." // client side
	// }

	// body
	args = append(args,	slog.String("data",
		dataLogString(event.Payload, h.opts.Codec),
	))

	h.opts.debug(
		ctx, "[ SEND::DATA ]", args...,
	)
}



func (h *dumpHandler) onRpcEnd(ctx context.Context, event *stats.End) {
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
	call.End = (*event) // shallowcopy

	status, _ := status.FromError(event.Error)

	// [DEBUG-4] Enabled ?
	var level = h.opts.Debug
	if event.Error != nil {
		level = slog.LevelWarn // (slog.LevelError + 3)
	}
	if !h.opts.can(ctx, level) {
		return
	}

	args := call.attrs
	
	// basic
	args = append(args,
		slog.String("part", ternary(event.Client, client, server)),
		slog.Uint64("call", call.seqId), // local identifier
		slog.String("peer", call.ConnTagInfo.RemoteAddr.String()),
		slog.String("path", call.RPCTagInfo.FullMethodName),
	)
	
	code := status.Code()
	args = append(args,
		slog.Int("grpc.code", int(code)),
		slog.String("grpc.status", code.String()),
	)

	if event.Error != nil {
		args = append(args,
			slog.Any("grpc.message", status.Message()),
		)
	}

	// disclose status.message detail(s)
	for _, nested := range status.Details() {
		switch data := nested.(type) {
		case *rpcpb.Error:
			args = append(args,
				slog.String("error.message", data.Message),
			)
		}
	}

	var (
		beginTime = call.Begin.BeginTime
		recvTime = call.InPayload.RecvTime
		sentTime = call.OutPayload.SentTime
		endTime = call.End.EndTime
	)

	const skrew = time.Microsecond

	if event.Client {
		// client: send/recv
		args = append(args,
			slog.Duration("time.send", sentTime.Sub(beginTime).Round(skrew)),
			slog.Duration("time.recv", recvTime.Sub(sentTime).Round(skrew)),
			slog.Duration("time.took", endTime.Sub(beginTime).Round(skrew)),
		)
	} else {
		// server: recv/send
		args = append(args,
			slog.Duration("time.recv", recvTime.Sub(beginTime).Round(skrew)),
			slog.Duration("time.send", endTime.Sub(sentTime).Round(skrew)),
			slog.Duration("time.took", endTime.Sub(recvTime).Round(skrew)),
		)
	}
	


	

	// dispo := "READ" // server side
	// if event.Client {
	// 	dispo = "SEND" // client side
	// }

	h.opts.log(
		// ctx, level,
		// context.TODO(), level,
		ctx, level, "[ CALL::END ]", args...,
	)
}

func dataLogString(data any, codec ProtoCodec) string {

	if data == nil {
		return ""
	}
	
	if msg, is := data.(proto.Message); is {
		text := codec.Format(msg)
		text = strings.ReplaceAll(text, "\"", "'")
		if !strings.HasPrefix(text, "{") {
			text = "{" + text + "}" // prototext ; separate *type{data}
		}
		return "&" + string(
			msg.ProtoReflect().Descriptor().FullName(),
		) + text
	}

	return fmt.Sprintf("%T", data)
}