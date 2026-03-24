package amqp

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

const (

	ContentTypeHeader = "content-type" // encoding
	MessageTypeHeader = "x-proto-type" // .well-known structure

	FromServiceHeader = "from-service" // .well-known service name
	FromServiceIdHeader = "from-service-id" // publisher service node-id

	// DOT(".") Header(s) are for internal use only
	// Informative, from Marshaler.Unmarshal() method ..

	TopicHeader = ".topic"
	ExchangeHeader = ".exchange"

	// MAY be populated while Unmarshal Delivery into Message for subscription
	// Indicates that Delivery.(*Message) is your own [FROM]:[service] publication
	SelfPublishedHeader = ".self"
)

// Indicates that recv.(*Message) is your own [FROM]:[service] publication
func IsSelfPublication(recv *message.Message) (ok bool) {
	_, ok = recv.Metadata[SelfPublishedHeader]
	return // ok
}

// var (

// 	ErrSelfPublication = fmt.Errorf("SELF publication ; skip")
// )

type Marshaler struct{
	Base amqp.Marshaler
	// SELF Publisher identification values
	Service, ServiceId string
}

var _ amqp.Marshaler = (*Marshaler)(nil)

func (c *Marshaler) Marshal(send *message.Message) (amqp091.Publishing, error) {
	// return amqp.DefaultMarshaler{}.Marshal(send)
	enc := c.Base
	if enc == nil {
		enc = amqp.DefaultMarshaler{
			// PostprocessPublishing: func(amqp091.Publishing) amqp091.Publishing {
			// 	panic("TODO")
			// },
			// MessageUUIDHeaderKey: "correlation",
			NotPersistentDeliveryMode: true,
		}
	}
	post, err := enc.Marshal(send)
	if err != nil {
		return post, err
	}

	if post.Timestamp.IsZero() {
		post.Timestamp = time.Now().UTC()
	}

	var ok bool
	// Headers["content-type"]
	if post.ContentType, ok = post.Headers[ContentTypeHeader].(string); ok {
		delete(post.Headers, ContentTypeHeader)
	}
	// Headers["x-proto-type"]
	if post.Type, ok = post.Headers[MessageTypeHeader].(string); ok {
		delete(post.Headers, MessageTypeHeader)
	}
	// Headers["from-service"]
	if c.Service != "" {
		post.Headers[FromServiceHeader] = c.Service
	}
	// Headers["from-service-id"]
	if c.ServiceId != "" {
		post.Headers[FromServiceIdHeader] = c.ServiceId
	}

	return post, nil
}

func (c *Marshaler) Unmarshal(recv amqp091.Delivery) (*message.Message, error) {

	// fromId, _ := recv.Headers[FromServiceIdHeader]
	// if c.ServiceId != "" && c.ServiceId == fromId {
	// 	from, _ := recv.Headers[FromServiceHeader]
	// 	if c.Service == "" || c.Service == from {
	// 		// SELF publication ; skip ..
	// 		// _ = recv.Nack(false, false)
	// 		return nil, ErrSelfPublication
	// 	}
	// }

	msg := message.NewMessage(recv.CorrelationId, recv.Body)
	msg.Metadata = make(message.Metadata, len(recv.Headers)+2)

	msg.Metadata[ExchangeHeader] = recv.Exchange
	msg.Metadata[TopicHeader] = recv.RoutingKey

	for key, value := range recv.Headers {
		// if key == d.computeMessageUUIDHeaderKey() {
		// 	continue
		// }

		var ok bool
		msg.Metadata[key], ok = value.(string)
		if !ok {
			// return nil, errors.Errorf("metadata %s is not a string, but %#v", key, value)
			msg.Metadata[key] = fmt.Sprintf("%v", value)
		}
	}

	// Header["content-type"]
	if recv.ContentType != "" && msg.Metadata[ContentTypeHeader] == "" {
		msg.Metadata[ContentTypeHeader] = recv.ContentType
	}
	 
	// Header["x-proto-type"]
	if recv.Type != "" && msg.Metadata[MessageTypeHeader] == "" {
		msg.Metadata[MessageTypeHeader] = recv.Type
	}

	fromId, _ := recv.Headers[FromServiceIdHeader]
	if c.ServiceId != "" && c.ServiceId == fromId {
		from, _ := recv.Headers[FromServiceHeader]
		if c.Service == "" || c.Service == from {
			// mark as SELF publication !
			msg.Metadata[SelfPublishedHeader] = "1"
		}
	}

	return msg, nil
}
