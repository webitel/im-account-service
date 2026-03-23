package service

import (
	"github.com/webitel/im-account-service/internal/service/updates"
	// v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
)

// .well-known service generated Update types
type (
  UpdateDeviceRegister = updates.UpdateDeviceRegister
  UpdateDeviceUnregister = updates.UpdateDeviceUnregister
  UpdateDeviceLogout = updates.UpdateDeviceLogout
)