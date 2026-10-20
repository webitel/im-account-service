package service

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/webitel/im-account-service/infra/pubsub/factory"
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/service/updates"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// UpdatesManager used to deal with Update(s)
type UpdatesManager struct {
	srv *Manager
	pub message.Publisher
}

func (c *Manager) Updates() (*UpdatesManager, error) {

	c.mx.Lock()
	defer c.mx.Unlock()

	if c.updates.srv == c {
		// once: inited
		return &c.updates, nil
	}

	pub, err := c.opts.Broker.GetFactory().BuildPublisher(
		&factory.PublisherConfig{
			Exchange: factory.ExchangeConfig{
				Name:    "im_system.events",
				Type:    "topic",
				Durable: true,
			},
		},
	)

	if err != nil {
		c.Warn(context.TODO(),
			"Failed to declare Publisher",
			semconv.ErrorKey, err,
		)
		return nil, err
	}

	c.updates = UpdatesManager{
		srv: c,
		pub: pub,
	}

	return &c.updates, nil
}

func (c *Manager) PublishUpdate(args updates.Update) error {

	broker, err := c.Updates()
	if err != nil {
		// Failed to prepare Publisher
		return err
	}

	err = broker.PublishUpdate(args)
	if err != nil {
		c.Warn(context.TODO(),
			"Failed to publish Update",
			semconv.ErrorKey, err,
		)
		return err
	}

	// [ OK ]
	return nil
}

func (c *UpdatesManager) Service() *Manager {
	if c != nil {
		return c.srv
	}
	return nil
}

func (c *UpdatesManager) Publisher() message.Publisher {
	if c != nil {
		return c.pub
	}
	return nil
}

func (c *UpdatesManager) PublishUpdate(args updates.Update) error {

	codec := updates.DefaultCodec

	data, err := codec.Encode(args)
	if err != nil {
		// failed to encode Update
		return err
	}

	update := message.Message{
		UUID: watermill.NewULID(),
		Metadata: message.Metadata{
			updates.ContentTypeHeader: codec.String(), // charset-utf-8
			updates.MessageTypeHeader: string(args.ProtoReflect().Descriptor().FullName()),
		},
		Payload: message.Payload(data),
	}

	err = c.Publisher().Publish(
		args.Topic(""), &update,
	)

	if err != nil {
		// failed to publish Update
		return err
	}

	// [ OK ]
	return nil
}

// subscribe for im-account-service Updates to sync runtime cache state(s)
func (c *Manager) subscribeOnClusterUpdates() error {
	broker := c.opts.Broker
	sub, err := broker.GetFactory().BuildSubscriber(
		"", // name ; autogen
		&factory.SubscriberConfig{
			Exchange: factory.ExchangeConfig{
				Name:    "im_system.events",
				Type:    "topic",
				Durable: true, // exchange durable(!)
			},
			Queue:        ".im-account-cluster", // ".cluster.updates",
			QueueDurable: false,
			BindingKey:   "updates.#", // updates.device.#
			Exclusive:    false,       // consumes from cluster node(s) ..
		},
	)

	if err != nil {
		return err
	}

	_ = broker.GetRouter().AddConsumerHandler(
		// consumer-id
		"im-account-cluster-updates",
		// subscriber
		"updates.#", sub,
		// callback
		c.onClusterUpdate,
	)

	return nil
}

func (c *Manager) onClusterUpdate(recv *message.Message) (_ error) {
	// fromId, _ := recv.Metadata["from-service-id"]
	// if fromId == c.opts.Id {
	// 	// SELF Publication ; skip ..
	// }

	if updates.IsSelfUpdate(recv) {
		// Indicates that WE published this recv.(*Message) from code around !
		// Just SKIP the processing, because the fact that we got it says
		// that we generated it based on certain changes that have already occurred.
		c.Debug(recv.Context(),
			"[ im-account-service ] SELF published Update, skip ..",
			// "exchange", recv.Metadata[".exchange"],
			"topic", recv.Metadata[".topic"],
		)
		return // nil
	}

	typeAs := recv.Metadata[updates.MessageTypeHeader]
	typeOf, err := protoregistry.GlobalTypes.FindMessageByName(
		protoreflect.FullName(typeAs),
	)
	if err != nil {
		// protoregistry.NotFound
		return
	}
	// mimetype := recv.Metadata[updates.ContentTypeHeader]
	// codec := updates.GetCodec(mimetype)
	codec := updates.DefaultCodec
	data := typeOf.New().Interface()
	err = codec.Decode([]byte(recv.Payload), data)
	if err != nil {
		// failed to decode Update arguments
		return
	}
	switch args := data.(type) {
	case *v1.UpdateDeviceLogout:
		{
			session, _ := c.cache.Get(_SessionId(args.Authorization.Id)).(*model.Authorization)
			if session != nil {
				c.cache.Del(session)
			}
		}
	case *v1.UpdateDeviceRegister:
		// TODO:
	case *v1.UpdateDeviceUnregister:
	}
	return
}
