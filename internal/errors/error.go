package errors

import (
	"cmp"
	"fmt"
	"net/http"
	"strings"

	rpcpb "github.com/webitel/im-account-service/proto/gen/rpc"

	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Parse tries to parse a JSON string into an error. If that
// fails, it will set the given string as the error detail.
func Parse(message string) (err *Error, ok bool) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, true
	}
	src := new(rpcpb.Error)
	enc := codecPlain
	der := enc.Unmarshal(
		[]byte(message), src,
	)
	if der != nil {
		src.Message = message
	}
	return FromProto(src), (der == nil)
}

func FromProto(src *rpcpb.Error) *Error {
	return (*Error)(src)
}

func FromError(src error) (err *Error, ok bool) {
	if src == nil {
		return nil, true
	}
	switch src := src.(type) {
	case *Error:
		{
			return src, true
		}
	}
	type grpcstatus interface {
		GRPCStatus() *status.Status
	}
	if impl, ok := src.(grpcstatus); ok {
		return FromStatus(impl.GRPCStatus())
	}
	return Parse(src.Error())
}

func FromStatus(src *status.Status) (err *Error, ok bool) {
	if src == nil {
		return nil, true
	}
	for _, any := range src.Proto().GetDetails() {
		sub, err := any.UnmarshalNew()
		if err != nil {
			// details = append(details, err)
			continue
		}
		switch e := sub.(type) {
		case *rpcpb.Error:
			{
				return (*Error)(e), true
			}
		}
	}

	// [finally]: try to parse JSON string
	if err, ok = Parse(src.Message()); !ok {
		err.Code = int32(src.Code())
		err.Status = src.Code().String()
	}

	return // err, ok?
}

// An internal Error details
type Error rpcpb.Error

// func (err *Error) Code() int32 {}
// func (err *Error) Status() string {}
// func (err *Error) Message() string {}

// Proto returns [e] as an *rpcpb.Error proto message.
func (err *Error) proto() *rpcpb.Error {
	return (*rpcpb.Error)(err)
}

// Proto returns [e] as an rpcpb.Error proto message.
func (err *Error) Proto() *rpcpb.Error {
	if err == nil {
		return nil
	}
	return proto.CloneOf(err.proto())
}

func (err *Error) ProtoAny() (*anypb.Any, error) {
	// if e == nil {} // This is error !
	return anypb.New(err.proto())
}

var _ error = (*Error)(nil)

var codecPlain = struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}{
	MarshalOptions: protojson.MarshalOptions{
		Multiline:         false,
		Indent:            "",
		AllowPartial:      false,
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
		Resolver:          nil,
	},
	UnmarshalOptions: protojson.UnmarshalOptions{
		AllowPartial:   false,
		DiscardUnknown: false,
		RecursionLimit: 0,
		Resolver:       nil,
	},
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return codecPlain.Format(err.proto())
}

func (err *Error) String() string {

	if err == nil {
		return ""
	}

	var (
		indent string
		format strings.Builder
	)
	defer format.Reset()

	if err.Code > 0 {
		// format.WriteString(indent)
		fmt.Fprintf(&format, "(#%d)", err.Code)
		indent = " "
	}

	if err.Status != "" {
		format.WriteString(indent)
		format.WriteString(err.Status)
		indent = " ; "
	}

	if err.Message != "" {
		format.WriteString(indent)
		format.WriteString(err.Message)
	}

	return format.String()
}

// GRPCStatus returns the grpc.Status represented by [e].
// Compatibility for grpc/status.FromError() method.
func (err *Error) GRPCStatus() *status.Status {
	src := err.proto()
	top := &spb.Status{
		Code:    int32(http2grpcCode(src.GetCode())),
		Message: cmp.Or(src.GetStatus(), src.GetMessage()),
		// Details: []*anypb.Any{sub},
	}
	sub, re := err.ProtoAny()
	if re != nil {
		top.Message = err.Error() // JSON
		// sub, _ = anypb.New(&rpcpb.Error{
		// 	// Id:      "",
		// 	Code:    500,
		// 	Status:  "Server Internal Error",
		// 	Message: re.Error(),
		// })
	} else {
		top.Details = []*anypb.Any{sub}
	}
	// top := &spb.Status{
	// 	Code:    int32(http2grpcCode(src.GetCode())),
	// 	Message: cmp.Or(src.GetStatus(), src.GetMessage()),
	// 	Details: []*anypb.Any{sub},
	// }
	return status.FromProto(top)
}

type Option func(err *Error)

// Error.Code Option
func Code(code int32) Option {
	return func(err *Error) {
		if code > 0 {
			err.Code = code
		}
	}
}

// Error.Status Option
func Status(code string) Option {
	return func(err *Error) {
		if code != "" {
			err.Status = code
		}
	}
}

func Message(form string, args ...any) Option {
	return func(err *Error) {
		text := form
		if len(args) > 0 {
			if form == "" {
				text = fmt.Sprint(args...)
			} else {
				text = fmt.Sprintf(form, args...)
			}
		}
		// if text != "" {
		err.Message = text
		// }
	}
}

func New(opts ...Option) (err *Error) {
	err = &Error{}
	err.init(opts)
	return // err
}

func (err *Error) init(opts []Option) {
	for _, setup := range opts {
		setup(err)
	}
}

func Errorf(message string, args ...any) *Error {
	return New(Message(message, args...))
}

// (#401) UNAUTHORIZED
//
//	 New(
//		Status("UNAUTHORIZED"),
//		Code(http.StatusUnauthorized),
//		opts...,
//	)
func Unauthorized(opts ...Option) *Error {
	err := New(
		Status("UNAUTHORIZED"),
		Code(http.StatusUnauthorized),
	)
	err.init(opts)
	return err
}

// (#400) BAD_REQUEST
//
//	 New(
//		Status("BAD_REQUEST"),
//		Code(http.StatusBadRequest),
//		opts...,
//	)
func BadRequest(opts ...Option) *Error {
	err := New(
		Status("BAD_REQUEST"),
		Code(http.StatusBadRequest),
	)
	err.init(opts)
	return err
}

// (#404) NOT_FOUND
//
//	 New(
//		Status("NOT_FOUND"),
//		Code(http.StatusNotFound),
//		opts...,
//	)
func NotFound(opts ...Option) *Error {
	err := New(
		Status("NOT_FOUND"),
		Code(http.StatusNotFound),
	)
	err.init(opts)
	return err
}
