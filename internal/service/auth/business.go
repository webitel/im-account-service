package auth

import (
	"github.com/webitel/im-account-service/internal/errors"
	"github.com/webitel/im-account-service/internal/service"
)

func AuthorizeBusiness(rpc *service.Context, pdc int64) error {

	isValid := func(oid int64) bool {
		return oid > 0
	}

	if !isValid(pdc) {
		return errors.BadRequest(
			errors.Message("messaging: invalid business account"),
		)
	}
	if !isValid(rpc.Dc) {
		// authorize
		rpc.Dc = pdc
	}
	// [RE]CHECK
	if rpc.Dc != pdc {
		// cross domain operation ! NOT ALLOWED !
		return errors.BadRequest(
			errors.Message("messaging: invalid business account"),
		)
	}
	// [ OK ]
	return nil
}