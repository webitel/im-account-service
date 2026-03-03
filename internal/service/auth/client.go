package auth

import (
	"strings"

	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/service"
)

// ClientAuthentication Options
type ClientAuthentication struct {
	Hint []string // OPTIONAL: alternative input value(s)
	Require bool // [X-Webitel-Client] header
}

func (x ClientAuthentication) Do(rpc *service.Context) error {
	
	// Once ..
	if rpc.App != nil {
		// [ OK ]
		return nil
	}

	// [X-Webitel-Client] header value(s)
	vs, _ := rpc.Header[model.H2_X_Client_ID]
	// & alternative input(s) ..
	vs = append(vs, x.Hint...)

	// CHECK: all input value(s) are the same value 
	clientId, ok := clientId(vs)
	if !ok && clientId != "" {
		// different values ; ambiguous
		return ErrClientAmbiguous
	}

	if clientId == "" {
		// Not specified !
		if x.Require {
			// [ ERR ]
			return ErrClientRequired
		}
		// [ OK ]
		return nil
	}

	// Lookup specified [client_id] application ..
	app, err := rpc.Service.GetApplication(rpc.Context, clientId)
	if err != nil {
		// storage lookup error
		return err
	}

	if app == nil {
		// Not Found !
		// [X-Webitel-Client] specified but invalid !
		return ErrClientUnauthorized
	}

	// Found !
	rpc.App = app
	err = AuthorizeBusiness(
		rpc, app.GetDc(),
	)
	if err != nil {
		return err
	}

	// Authorize App client (device) constraints ..
	_ = DeviceAuthentication{}.Do(rpc)
	err = rpc.App.Clients().Authorize(rpc.Device)
	if err != nil {
		return err
	}

	// [ OK ]
	return nil
}

func clientId(vs []string) (id string, ok bool) {

	for _, s := range vs {
		// canonize
		s = strings.TrimSpace(s)
		if s == "" {
			// ignore
			continue
		}
		// specified ?
		if id == "" {
			id = s
			continue
		}
		// match exact ?
		if id != s {
			// different values !
			return id, false
		}
	}
	// [ OK ]
	return id, (id != "")
}
