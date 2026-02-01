package auth

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/webitel/im-account-service/infra/pubsub"
	"github.com/webitel/im-account-service/infra/pubsub/factory"
)

func (c *Client) Subscribe(broker pubsub.Provider) error {

	if broker == nil {
		return nil
	}

	sub, err := broker.GetFactory().BuildSubscriber(
		"", // name ; autogen
		&factory.SubscriberConfig{
			Exchange: factory.ExchangeConfig{
				Name:    "webitel",
				Type:    "topic",
				Durable: true, // exchange durable(!)
			},
			Queue:             "todo_exclusive_queue_for_account_service_node_id",
			RoutingKey:        "invalidate.#",
			ExclusiveConsumer: false, // true, !!!
		},
	)

	if err != nil {
		return err
	}

	_ = broker.GetRouter().AddHandler(
		"webitel",
		// subscriber
		"#", sub,
		// publisher
		"", nil,
		// handler
		c.onInvalidate,
	)

	// messages, err := sub.Subscribe(context.TODO(), "invalidate.#")

	// if err != nil {
	// 	return err
	// }

	// go consumeUpdates(messages)
	return nil
}

func (c *Client) onInvalidate(update *message.Message) (_ []*message.Message, _ error) {

	// invalidate.session.d42f82ab-421a-49c6-98a2-5af30abc5b2a
	// [ Properties -> headers ]:
	// app_id:	go.webitel.api-a15aa91e-e3bb-4165-b969-ac49467cbef9
	// cause:	revoke
	// event:	invalidate
	// objclass:	session
	// session:	d42f82ab-421a-49c6-98a2-5af30abc5b2a
	// timestamp:	1769509517003

	// exchange := update.Metadata.Get(".exchange")
	topic := update.Metadata.Get(".topic")

	cause := update.Metadata.Get("cause")
	objclass := update.Metadata.Get("objclass")
	objectId := update.Metadata.Get(objclass)

	c.logger.Debug(
		("[ RECV::MSG ] " + topic),
		"invalidate", cause,
		"objclass", objclass,
		objclass, objectId,
	)

	// switch objclass {
	// case `customer`: // certification
	// case `session`: // authorization
	// case `domain`: // organization
	// case `user`:
	// 	{
	// 		uid, err := strconv.ParseInt(objectId, 10, 64)
	// 	}
	// case `role`:
	// case `obac`:
	// default:
	// }

	// TODO: handle cache[d] entries ...

	// ACK ; No publish !
	return nil, nil
}
