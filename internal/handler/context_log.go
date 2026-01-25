package handler

import (
	"cmp"
	"context"
	"log/slog"
)

type ctxLogValue struct {
	ctx *Context
	val slog.Value
	// group string
	// attrs []slog.Attr
}

// ContextLog returns [log/slog.LogValuer] attributes helper
func ContextLog(rpc *Context) *ctxLogValue { // slog.LogValuer { //
	return &ctxLogValue{ctx: rpc}
}

// Attrs returns SET of attributes for the current (underlying) Context state
func (x *ctxLogValue) Attrs() (attrs []slog.Attr) {

	// current: Authorization
	var (
		app     = x.ctx.App
		device  = x.ctx.Device
		session = x.ctx.Session
		account = x.ctx.Contact
	)

	// [rpc.dc]
	if app != nil {
		if dc := app.GetDc(); dc > 0 {
			attrs = append(attrs, slog.Int64(
				"dc", dc,
			))
		}
	}

	// [rpc.client.ip]
	if device != nil {
		if ip := device.IP(); len(ip) > 0 {
			attrs = append(attrs, slog.String(
				// "device.ip", ip.String(),
				"client.ip", ip.String(),
			))
		}
	}

	// [rpc.client.id] ; [X-Webitel-Client]
	if app != nil {
		attrs = append(attrs, slog.String(
			// "app.id", app.ClientId(),
			"client.id", app.ClientId(),
		))
	}

	// [rpc.client.sub]  ; [X-Webitel-Device]
	// [rpc.client.name] ; [User-Agent].(name/ver)
	if device != nil {
		if device.Id != "" {
			attrs = append(attrs, slog.String(
				// "device.id", device.Id, // [X-Webitel-Device]
				"client.sub", device.Id, // [X-Webitel-Device]
			))
		}
		app := &device.App
		if name := app.Name; name != "" {
			if app.Version != "" {
				name += "/" + app.Version
			}
			attrs = append(attrs, slog.String(
				// "device.app", name,
				"client.name", name,
			))
		}
	}

	// [rpc.session.id] ; internal session authorization
	if session != nil {
		if session.Id != "" {
			attrs = append(attrs, slog.String(
				"session.id", session.Id,
			))
		}
	}

	// [rpc.contact.id]   ; internal (account) identifier
	// [rpc.contact.iss]  ; external (contact) issuer identifier
	// [rpc.contact.sub]  ; external (contact) subject identifier
	// [rpc.contact.type] ; internal (account) protocol registered for issuer
	// [rpc.contact.name] ; internal (account) common name
	if account != nil {
		// internal: <type:id>
		if account.Id != "" {
			attrs = append(attrs, slog.String(
				// "account.id", account.Id,
				"contact.id", account.Id,
			))
		}
		// external: <sub@iss>
		if account.Iss != "" {
			attrs = append(attrs, slog.String(
				"contact.iss", account.Iss,
			))
		}
		if account.Sub != "" {
			attrs = append(attrs, slog.String(
				"contact.sub", account.Sub,
			))
		}
		// account info
		if account.Type != "" && account.Type != account.Iss {
			attrs = append(attrs, slog.String(
				"contact.type", account.Type,
			))
		}
		if account.Name != "" {
			attrs = append(attrs, slog.String(
				"contact.name", account.Name,
			))
		}
	}

	return // attrs
}

// Group return defered slog.Value attributes resolver.
// Implements [log/slog.LogValuer] interface.
func (x *ctxLogValue) Group(name string) slog.Attr {
	// return slog.GroupAttrs(name, x.Attrs()...)
	return slog.Any(name, slog.LogValuer(x))
}

// LogValue returns defered slog.GroupValue(attrs).
// Implements [log/slog.LogValuer] interface.
func (x *ctxLogValue) LogValue() slog.Value {
	// once: prepared ?
	if x.val.Kind() != slog.KindGroup {
		x.val = slog.GroupValue(x.Attrs()...)
	}
	// cache: prepared !
	return x.val
}

// LogEnabled reports whether given [level] is enabled for logging
func (rpc *Context) LogEnabled(ctx context.Context, level slog.Level) bool {
	if rpc.Logger != nil {
		ctx = cmp.Or(ctx, rpc.Context)
		return rpc.Logger.Enabled(ctx, level)
	}
	return false
}

// Log record message ..
func (rpc *Context) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	ctx = cmp.Or(ctx, rpc.Context)
	if !rpc.LogEnabled(ctx, level) {
		return
	}
	// expose Context.(Authentication) attributes
	args = append([]any{ContextLog(rpc).Group("rpc")}, args...)
	rpc.Logger.Log(ctx, level, msg, args...)
}

func (rpc *Context) Warn(msg string, args ...any) {
	rpc.Log(nil, slog.LevelWarn, msg, args...)
}

func (rpc *Context) Debug(msg string, args ...any) {
	rpc.Log(nil, slog.LevelDebug, msg, args...)
}

// TODO: implement more level(s) below ..
