package model

import (
	"net"
	"time"
)

// Session. Authorization
type Authorization struct {
	// Business (Domain) Account ID
	Dc int64
	// Session (internal) identifier
	Id string
	// Last known IP address ; FROM
	IP net.IP
	// When was the session created
	Date time.Time
	// Session (device) display name.
	Name string
	// [X-Webitel-Client] ; Client Application [client_id] ; VIA
	AppId string
	// [X-Webitel-Device] ; Client Device (self) identification
	Device Device
	// Authorized end-user Contact info
	Contact *ContactId // *Identity
	// Extra metadata claims
	Metadata map[string]any
	// Whether this is the current session
	Current bool
	// Grant an [access_token] for this session Authorization
	Grant *AccessToken
}

type SessionList = Dataset[Authorization]

// // Session. Authorization
// type Session struct {
// 	Dc   int64     // domain id
// 	Id   UUID      // session id
// 	Date time.Time // created date

// 	Name string       // session / device name
// 	Auth *AccessToken // access (token) granted ?

// 	ClientId UUID // App
// 	DeviceId UUID // Sub
// 	UserId   *ContactId

// 	// extra metadata claims
// 	Metadata map[string]any
// }

// type SessionList = Dataset[Session]
