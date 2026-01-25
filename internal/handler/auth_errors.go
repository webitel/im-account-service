package handler

import "github.com/webitel/im-account-service/internal/errors"

var (
	//
	ErrDeviceRequired = errors.Unauthorized(
		errors.Status("UNAUTHORIZED_CLIENT"),
		errors.Message("messaging: device authorization required"),
	)

	// ErrDeviceAuthorization = errors.Unauthorized(
	// 	errors.Status("UNAUTHORIZED_CLIENT"),
	// 	errors.Message("messaging: invalid device authorization"),
	// )

	ErrDeviceUnauthorized = errors.Unauthorized(
		errors.Status("UNAUTHORIZED_CLIENT"),
		errors.Message("messaging: device not authorized"),
	)

	ErrClientRequired = errors.Unauthorized(
		errors.Status("UNAUTHORIZED_CLIENT"),
		errors.Message("messaging: client authorization required"),
	)

	ErrClientAmbiguous = errors.Unauthorized(
		errors.Status("UNAUTHORIZED_CLIENT"),
		errors.Message("messaging: ambiguous client authorization"),
	)

	ErrClientUnauthorized = errors.Unauthorized(
		errors.Status("UNAUTHORIZED_CLIENT"),
		errors.Message("messaging: invalid client authorization"),
	)

	ErrAccountUnauthorized = errors.Unauthorized(
		errors.Status("UNAUTHORIZED"),
		errors.Message("messaging: account not authorized"),
	)

	ErrTokenInvalid = errors.Unauthorized(
		errors.Status("UNAUTHORIZED"),
		errors.Message("messaging: invalid access token"),
	)
)
