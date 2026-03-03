package auth

import (
	"github.com/webitel/im-account-service/internal/model"
	"github.com/webitel/im-account-service/internal/service"
)

// DeviceAuthentication Options
type DeviceAuthentication struct {
	Require bool // [X-Webitel-Device] header
}

func (x DeviceAuthentication) Do(rpc *service.Context) error {
	
	// Once ..
	rpc.Device = GetDevice(rpc)
	// if rpc.Device == nil {
	// 	// Gather remote [User-Agent] info
	// 	client, _ := model.GetDeviceAuthorization(rpc.Context)
	// 	rpc.Device = &client
	// }

	if rpc.Device.Id == "" && x.Require {
		return ErrDeviceRequired
	}

	// [ OK ]
	return nil
}

func GetDevice(rpc *service.Context) *model.Device {
	if rpc.Device != nil {
		return rpc.Device
	}
	device, _ := model.GetDeviceAuthorization(rpc.Context)
	return &device
}