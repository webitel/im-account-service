package updates

import (
	"fmt"

	v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type (
	
	UpdateDeviceRegister v1.UpdateDeviceRegister
	UpdateDeviceUnregister v1.UpdateDeviceUnregister
	UpdateDeviceLogout v1.UpdateDeviceLogout
)

var _ Update = (*UpdateDeviceRegister)(nil)

func (e *UpdateDeviceRegister) isUpdate() {}

func (e *UpdateDeviceRegister) Topic(_ string) string {
	return fmt.Sprintf("updates.device.register.%d", e.Proto().GetAuthorization().GetDc())
}

func (e *UpdateDeviceRegister) Proto() *v1.UpdateDeviceRegister {
	if e != nil {
		return (*v1.UpdateDeviceRegister)(e)
	}
	return nil
}

func (e *UpdateDeviceRegister) ProtoReflect() protoreflect.Message {
	return e.Proto().ProtoReflect()
}

// func (e *UpdateDeviceRegister) Publication(ctx context.Context) (Publication, error) {
// 	return Publication{
// 		Topic:   e.Topic(),
// 		Message: message.Message{},
// 		Context: ctx,
// 	}, nil
// }

var _ Update = (*UpdateDeviceRegister)(nil)

func (e *UpdateDeviceUnregister) isUpdate() {}

func (e *UpdateDeviceUnregister) Topic(_ string) string {
	return fmt.Sprintf("updates.device.unregister.%d", e.Proto().GetAuthorization().GetDc())
}

func (e *UpdateDeviceUnregister) Proto() *v1.UpdateDeviceUnregister {
	if e != nil {
		return (*v1.UpdateDeviceUnregister)(e)
	}
	return nil
}

func (e *UpdateDeviceUnregister) ProtoReflect() protoreflect.Message {
	return e.Proto().ProtoReflect()
}

var _ Update = (*UpdateDeviceLogout)(nil)

func (e *UpdateDeviceLogout) isUpdate() {}

func (e *UpdateDeviceLogout) Topic(_ string) string {
	return fmt.Sprintf("updates.device.logout.%d", e.Proto().GetAuthorization().GetDc())
}

func (e *UpdateDeviceLogout) Proto() *v1.UpdateDeviceLogout {
	if e != nil {
		return (*v1.UpdateDeviceLogout)(e)
	}
	return nil
}

func (e *UpdateDeviceLogout) ProtoReflect() protoreflect.Message {
	return e.Proto().ProtoReflect()
}