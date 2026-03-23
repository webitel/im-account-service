package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/webitel/im-account-service/internal/service/cache"
)

// Service Domain Manager
type Manager struct {
	mx sync.Mutex
	opts Options
	cache *cache.LRU
	updates UpdatesManager
}

// Domain Options
func (c *Manager) Options() Options {
	return c.opts
}

// New Service Domain
func New(opts Options) (*Manager, error) {
	c := &Manager{
		opts: opts,
		cache: cache.New(func(cache *cache.Options) {
			cache.Logger = opts.Logger
			cache.TTL = (24 * time.Hour) // (30 * time.Second)
			cache.IndexKeys = indexData
		}),
	}
	// region: subscribe on cluster Updates ..
	err := c.subscribeOnClusterUpdates()
	if err != nil {
		return c, err
	}
	// broker := c.opts.Broker
	// sub, err := broker.GetFactory().BuildSubscriber(
	// 	"", // name ; autogen
	// 	&factory.SubscriberConfig{
	// 		Exchange: factory.ExchangeConfig{
	// 			Name:    "im_system.events",
	// 			Type:    "topic",
	// 			Durable: true, // exchange durable(!)
	// 		},
	// 		Queue:             "", // "todo_exclusive_queue_for_account_service_node_id",
	// 		RoutingKey:        "#",
	// 		ExclusiveConsumer: true,
	// 	},
	// )

	// if err != nil {
	// 	return nil, err
	// }

	// _ = broker.GetRouter().AddConsumerHandler(
	// 	"im.account.cluster.updates",
	// 	// subscriber
	// 	"updates.device.#", sub,
	// 	// handler
	// 	c.onClusterUpdate,
	// )
	// endregion: subscribe on cluster Updates ..
	return c, nil
}

// CanLog reports whether given [level] is enabled for logging
func (c *Manager) CanLog(ctx context.Context, level slog.Level) bool {
	if c.opts.Logger != nil {
		return c.opts.Logger.Enabled(ctx, level)
	}
	return false
}

// Log record message ..
func (c *Manager) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !c.CanLog(ctx, level) {
		return
	}
	c.opts.Logger.Log(ctx, level, msg, args...)
}

func (c *Manager) Info(ctx context.Context, msg string, args ...any) {
	c.Log(ctx, slog.LevelInfo, msg, args...)
}

func (c *Manager) Warn(ctx context.Context, msg string, args ...any) {
	c.Log(ctx, slog.LevelWarn, msg, args...)
}

func (c *Manager) Debug(ctx context.Context, msg string, args ...any) {
	c.Log(ctx, slog.LevelDebug, msg, args...)
}

// TODO: implement more level(s) below ..