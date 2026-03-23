package amqp

import (
	"cmp"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/webitel/im-account-service/infra/pubsub/factory"
)

const (
	TopicExchangeType  = "topic"
	FanoutExchangeType = "fanout"
	DirectExchangeType = "direct"
)

// RabbitMQ AMQP 0.9.1 Broker Factory Options
type Factory struct {
	ServiceId string   // app_id ; service.(node).id
	ServiceName string // app ; service.(name) .well-known
	DataSourceName string // connectionString
	Logger watermill.LoggerAdapter
}

func NewFactory(dsn string, logger watermill.LoggerAdapter) (*Factory, error) {
	return &Factory{
		DataSourceName: dsn,
		Logger: logger,
	}, nil
}

func (f *Factory) BuildSubscriber(name string, subConfig *factory.SubscriberConfig) (message.Subscriber, error) {
	if subConfig == nil {
		return nil, fmt.Errorf("no subscriber configured")
	}
	conf := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: f.DataSourceName,
		},
		// Marshaler: amqp.DefaultMarshaler{},
		Marshaler: Marshaler{},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(s string) string {
				return subConfig.Exchange.Name
			},
			Type:    subConfig.Exchange.Type,
			Durable: subConfig.Exchange.Durable,
		},
		Queue: amqp.QueueConfig{
			GenerateName: func(topic string) string {
				_ = topic
				return cmp.Or(subConfig.Queue, f.ServiceId)
			},
			Durable: subConfig.QueueDurable,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(subscribeTopic string) string {
				_ = subscribeTopic
				return subConfig.BindingKey
			},
		},
		Consume: amqp.ConsumeConfig{
			Consumer:  name,
			Exclusive: subConfig.Exclusive,
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
	return amqp.NewSubscriber(conf, f.Logger)
}

func (f *Factory) BuildPublisher(pubConfig *factory.PublisherConfig) (message.Publisher, error) {
	conf := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: f.DataSourceName,
		},
		// Marshaler: amqp.DefaultMarshaler{},
		Marshaler: amqp.CorrelatingMarshaler{
			PostprocessPublishing: func(post amqp091.Publishing) amqp091.Publishing {

				if post.Timestamp.IsZero() {
					post.Timestamp = time.Now().UTC()
				}

				post.Headers["from-service"] = f.ServiceName
				post.Headers["from-service-id"] = f.ServiceId

				// post.UserId = f.app // service.(node).id
				// post.AppId = "im-account-service" // service.(name)

				post.Type, _ = post.Headers["x-proto-type"].(string)
				post.ContentType, _ = post.Headers["content-type"].(string)

				delete(post.Headers, "x-proto-type")
				delete(post.Headers, "content-type")

				return post
			},
			NotPersistentDeliveryMode: true,
		},
		Exchange: amqp.ExchangeConfig{
			GenerateName: func(_ string) string { // (topic string) string
				return pubConfig.Exchange.Name // static
			},
			Type:    pubConfig.Exchange.Type,
			Durable: pubConfig.Exchange.Durable,
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return topic
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
	return amqp.NewPublisher(conf, f.Logger)
}
