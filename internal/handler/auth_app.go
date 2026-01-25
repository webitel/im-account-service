package handler

import (
	"github.com/webitel/im-account-service/internal/model"
)

func AppAuthorization(require bool) ContextFunc {
	return func(rpc *Context) error {

		app, err := GetApplication(rpc)
		if err != nil {
			return err
		}

		if app == nil {
			if !require {
				// NOT Specified | Found
				return nil
			}
			return ErrClientRequired
		}

		rpc.Dc = app.GetDc()
		rpc.App = app

		return nil
	}
}

// [X-Webitel-Client] ; Get Application authorization credentials
func GetApplication(rpc *Context) (*model.Application, error) {

	if rpc.App != nil {
		// once ; substitute
		return rpc.App, nil
	}

	if vs, ok := rpc.Header[model.H2_X_Client_ID]; ok {
		if clientId := model.CoalesceLast(vs...); clientId != "" {
			app, err := rpc.Service.GetApplication(rpc.Context, clientId)
			if err != nil {
				// storage internal error
				return nil, err
			}
			if app == nil {
				// Not Found !
				// [X-Webitel-Client] provided but invalid !
				return nil, ErrClientUnauthorized
			}
			// [ OK ]
			return app, nil
		}
		// Empty header value provided !
	}

	// Not specified !
	return nil, nil
}

// [X-Webitel-Device] ; Get [User-Agent] info
func DeviceAuthorization(require bool) ContextFunc {
	return func(rpc *Context) error {

		if rpc.Device == nil {
			// once
			device, _ := model.GetDeviceAuthorization(rpc.Context)
			rpc.Device = &device
		}

		if require && rpc.Device.Id == "" {
			return ErrDeviceRequired
		}

		return authorizeClient(rpc.App, rpc.Device)
	}
}

// Authorize client (device) within Application config
func authorizeClient(app *model.Application, client *model.Device) error {

	if app == nil {
		// [ OK ] No Configuration !
		return nil
	}
	// TODO: below ...
	return nil

	clients := app.Proto().GetClient()

	ok := (len(clients.GetUa()) == 0)
	for _, pattern := range clients.GetUa() {
		_ = pattern
		// ok, err := regexp.MatchString(pattern, client.App.String)
	}
	if !ok {
		return ErrDeviceUnauthorized
	}

	ip := client.IP()
	ok = (len(clients.GetNet().GetCidr()) == 0)
	for _, network := range clients.GetNet().GetCidr() {
		_, _ = ip, network
		// mask.Accept(client.IP())
	}
	if !ok {
		return ErrDeviceUnauthorized
	}

	// header := metadata.FromIncomingContext()
	origin := "" // model.GetHeaderH2(rpc.Header, model.H2_Origin)
	ok = (len(clients.GetWeb().GetOrigin()) == 0 || origin == "")
	for _, allowed := range clients.GetWeb().GetOrigin() {
		_ = allowed
		// mask.Accept(client.IP())
	}
	if !ok {
		return ErrDeviceUnauthorized
	}

	// [ OK ]
	return nil
}
