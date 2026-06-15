package service

import (
	"fmt"
	"reflect"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/webitel-go-kit/pkg/semconv"
)

// shorthand for operation result caching ..
//
//	if err == nil && data != nil {
//	  _ = (*Manager).AddCache(data)
//	}
func (srv *Manager) Cache(data any, err error) {

	if err != nil {
		// operation error ; ignore
		return
	}

	_ = srv.AddCache(data)
}

// add given [data] record to memory cache
func (srv *Manager) AddCache(data any) (err error) {

	rval := reflect.ValueOf(data)
	if !rval.IsValid() || rval.IsZero() {
		return nil // no value
	}

	defer func() {
		if e := recover(); e != nil {
			// srv.cache.Options().Log()
			srv.opts.Logger.Warn(
				"Failed to cache data record",
				"typeof", reflect.TypeOf(data).Name(),
				semconv.ErrorKey, e,
			)

			err, _ = e.(error)
			if err == nil {
				err = fmt.Errorf("%v", e)
			}
		}
	}()

	err = srv.cache.Add(data)
	return // err ; deferred
}

// remove given [data] record from memory cache
func (srv *Manager) DelCache(data any) (err error) {

	rval := reflect.ValueOf(data)
	if !rval.IsValid() || rval.IsZero() {
		return // no value
	}

	defer func() {
		if e := recover(); e != nil {
			// srv.cache.Options().Log()
			srv.opts.Logger.Warn(
				"Failed to remove data cache",
				"typeof", reflect.TypeOf(data).Name(),
				semconv.ErrorKey, e,
			)

			err, _ = e.(error)
			if err == nil {
				err = fmt.Errorf("%v", e)
			}
		}
	}()

	_ = srv.cache.Del(data)
	return // err ; deferred
}

func indexData(data any) []any {
	switch row := data.(type) {
	case *model.Application:
		return indexApp(row)
	case *model.Authorization:
		return indexSession(row)
	}
	return nil
}

// comparable cache index key types
type (
	_ClientId string
)

func indexApp(data *model.Application) []any {
	return []any{_ClientId(data.ClientId())}
}

type (
	_DeviceId      string
	_ContactId     string
	_SessionId     string
	_DeviceContact struct {
		deviceId  _DeviceId
		contactId _ContactId
	}
)

func indexSession(data *model.Authorization) []any {
	keys := make([]any, 0, 2)
	if data.Id != "" {
		keys = append(keys, _SessionId(data.Id))
	}
	if data.Device.Id != "" {
		if data.Contact.Id != "" {
			keys = append(keys, _DeviceContact{
				deviceId:  _DeviceId(data.Device.Id),
				contactId: _ContactId(data.Contact.Id),
			})
		}
	}
	return keys
	// return []any{
	//   _SessionId(data.Id),
	//   _DeviceSession{
	//     deviceId: _DeviceId(data.Device.Id),
	//     contactId: _ContactId(data.Contact.Id),
	//   },
	// }
}

func indexSessionRequest(req *SessionListOptions) (keys []any) {
	if req.Id != "" {
		keys = append(keys, _SessionId(req.Id))
	}
	if req.DeviceId != "" {
		if req.ContactId != nil {
			keys = append(keys, _DeviceContact{
				deviceId:  _DeviceId(req.DeviceId),
				contactId: _ContactId(req.ContactId.Id),
			})
		}
	}
	return
}
