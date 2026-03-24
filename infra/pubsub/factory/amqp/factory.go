package amqp

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
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
		// Marshaler: Marshaler{},
		// Marshaler: amqp.DefaultMarshaler{},
		Marshaler: &Marshaler{
			Base: amqp.CorrelatingMarshaler{
				// PostprocessPublishing: func(amqp091.Publishing) amqp091.Publishing {
				// 	panic("TODO")
				// },
				NotPersistentDeliveryMode: true,
			},
			Service:   f.ServiceName,
			ServiceId: f.ServiceId,
		},
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
				// return cmp.Or(subConfig.Queue, f.ServiceId)
				queueName := subConfig.Queue
				const sep = "." 
				// internal queue-name MAY start with DOT(".")
				// make it UNIQUE & prefix with assigned service.(node).id
				switch queueName {
				case "", sep:
					queueName = (f.ServiceId + sep + generateRandomName(8, charset))
				}
				if strings.HasPrefix(queueName, sep) {
					queueName = (f.ServiceId + queueName)
				}
				return queueName // UNIQUE
			},
			Durable: subConfig.QueueDurable,
			AutoDelete: !subConfig.QueueDurable,
			Exclusive: !subConfig.QueueDurable,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(subscribeTopic string) string {
				_ = subscribeTopic
				return subConfig.BindingKey
			},
		},
		Consume: amqp.ConsumeConfig{
			NoRequeueOnNack: true, // FIXME
			Exclusive: subConfig.Exclusive,
			Consumer:  name,
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
		Marshaler: &Marshaler{
			Base: amqp.CorrelatingMarshaler{
				// PostprocessPublishing: func(amqp091.Publishing) amqp091.Publishing {
				// 	panic("TODO")
				// },
				NotPersistentDeliveryMode: true,
			},
			Service:   f.ServiceName,
			ServiceId: f.ServiceId,
		},
		// Marshaler: amqp.DefaultMarshaler{},
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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomName(length int, charset string) string {
	// Seed the generator (Go 1.20+ automatically seeds the default source, 
	// but explicit seeding is shown for clarity and compatibility)
	// For modern Go, you can often omit the rand.Seed() line if using the
	// default, package-level rand functions.
	// var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano())) // Pre-Go 1.20 method

	sb := strings.Builder{}
	sb.Grow(length)
	for range length {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()
}
