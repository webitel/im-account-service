package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/webitel/im-account-service/internal/service/cache"
)

// Service Domain Manager
type Manager struct {
	opts Options
	cache *cache.LRU
}

// Domain Options
func (c *Manager) Options() Options {
	return c.opts
}

// New Service Domain
func New(opts Options) (*Manager, error) {
	return &Manager{
		opts: opts,
		cache: cache.New(func(cache *cache.Options) {
			cache.Logger = opts.Logger
			cache.TTL = (24 * time.Hour)
			cache.IndexKeys = indexData
		}),
	}, nil
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