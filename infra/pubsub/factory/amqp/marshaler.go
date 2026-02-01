package amqp

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"

	amqp091 "github.com/rabbitmq/amqp091-go"
)

type Marshaler struct{}

var _ amqp.Marshaler = Marshaler{}

func (Marshaler) Marshal(send *message.Message) (amqp091.Publishing, error) {
	return amqp.DefaultMarshaler{}.Marshal(send)
}

func (Marshaler) Unmarshal(recv amqp091.Delivery) (*message.Message, error) {
	// return amqp.DefaultMarshaler{}.Unmarshal(recv)
	// c := amqp.DefaultMarshaler{}
	// msgUUIDStr, err := c.unmarshalMessageUUID(amqpMsg)
	// if err != nil {
	// 	return nil, err
	// }

	msg := message.NewMessage(recv.CorrelationId, recv.Body)
	msg.Metadata = make(message.Metadata, len(recv.Headers)+2)

	msg.Metadata[".exchange"] = recv.Exchange
	msg.Metadata[".topic"] = recv.RoutingKey

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

	return msg, nil
}
