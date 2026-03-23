package updates

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Codec for content encoding
type Codec interface {
  // Decode binary data into Update
	Decode([]byte, any) error
  // Encode Update to binary form
	Encode(any) ([]byte, error)
	// MIME type compatible string
	String() string
}

// [application/protobuf+json; charset=utf-8]  
// https://www.iana.org/assignments/media-types/media-types.xhtml  
// https://datatracker.ietf.org/doc/html/draft-ietf-dispatch-mime-protobuf-06
type ProtoJsonCodec struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}

var _ Codec = ProtoJsonCodec{}

// MIME type compatible string
func (ProtoJsonCodec) String() string {
	return "application/protobuf+json"
}

func (c ProtoJsonCodec) Decode(src []byte, dst any) error {
	
	msg, ok := dst.(proto.Message)
	if !ok || msg == nil {
		return fmt.Errorf("protobuf+json: expect proto.Message data")
	}

	return c.UnmarshalOptions.Unmarshal(src, msg)
}

func (c ProtoJsonCodec) Encode(src any) ([]byte, error) {

	msg, ok := src.(proto.Message)
	if !ok || msg == nil {
		return nil, fmt.Errorf("protobuf+json: expect proto.Message data")
	}

	return c.MarshalOptions.Marshal(msg)
}



type ProtobufCodec struct {
	proto.MarshalOptions
	proto.UnmarshalOptions
}

var _ Codec = ProtobufCodec{}

// MIME type compatible string
func (ProtobufCodec) String() string {
	return "application/protobuf"
}

func (c ProtobufCodec) Decode(src []byte, dst any) error {
	
	msg, ok := dst.(proto.Message)
	if !ok || msg == nil {
		return fmt.Errorf("protobuf: expect proto.Message data")
	}

	return c.UnmarshalOptions.Unmarshal(src, msg)
}

func (c ProtobufCodec) Encode(src any) ([]byte, error) {

	msg, ok := src.(proto.Message)
	if !ok || msg == nil {
		return nil, fmt.Errorf("protobuf: expect proto.Message data")
	}

	return c.MarshalOptions.Marshal(msg)
}



var (
	CodecProtoBuf = ProtobufCodec{
		MarshalOptions:   proto.MarshalOptions{
			AllowPartial:      true,
			Deterministic:     true,
			UseCachedSize:     false,
		},
		UnmarshalOptions: proto.UnmarshalOptions{
			Merge:             false,
			AllowPartial:      true,
			DiscardUnknown:    false,
			RecursionLimit:    0,
			NoLazyDecoding:    false,
			Resolver:          nil,
		},
	}
	CodecProtoJson = ProtoJsonCodec{
		MarshalOptions:   protojson.MarshalOptions{
			Multiline:         true,
			Indent:            "  ",
			AllowPartial:      true,
			UseProtoNames:     true,
			UseEnumNumbers:    false,
			EmitUnpopulated:   false,
			EmitDefaultValues: false,
			Resolver:          nil,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			AllowPartial:      true,
			DiscardUnknown:    false,
			RecursionLimit:    0,
			Resolver:          nil,
		},
	}
	// DefaultCodec Codec = &CodecProtoBuf // production
	DefaultCodec Codec = &CodecProtoJson   // development
)