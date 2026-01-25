package model

import (
	"net"
	"net/http"
	"net/netip"
	"net/textproto"
	"strings"

	"google.golang.org/grpc/metadata"
)

// HTTP/1.* well-known headers
// net/textproto.CanonicalMIMEHeaderKey()
const (
	H1_Origin          = "Origin"
	H1_User_Agent      = "User-Agent"
	H1_X_Forwarded_For = "X-Forwarded-For"
	H1_X_Real_IP       = "X-Real-IP"
	// Webitel [Device]=[subscriber_id] client-self identification token ; header
	H1_X_Device_ID = "X-Webitel-Device"
	// Webitel [Application]=[client_id] authorization ; header
	H1_X_Client_ID = "X-Webitel-Client"
	// Webitel [User] authorization token header
	H1_X_Access_Token = "X-Webitel-Access"
	// Native [Service] authorization
	H1_From_Service    = "From-Service"
	H1_From_Service_ID = "From-Service-Id"
)

// HTTP/2.* well-known headers
//
// Only the following ASCII characters are allowed in keys:
//
//	digits: 0-9
//	uppercase letters: A-Z (normalized to lower)
//	lowercase letters: a-z
//	special characters: -_.
//
// Uppercase letters are automatically converted to lowercase.
//
// Keys beginning with "grpc-" are reserved for grpc-internal use only and may result in errors if set in metadata.
//
// See https://pkg.go.dev/google.golang.org/grpc/metadata#New for syntax
const (
	H2_Origin          = "origin"
	H2_User_Agent      = "user-agent"
	H2_X_Forwarded_For = "x-forwarded-for"
	H2_X_Real_IP       = "x-real-ip"

	H2_From_Service    = "from-service" // Native [Service] authorization
	H2_From_Service_ID = "from-service-id"

	H2_X_Device_ID    = "x-webitel-device" // Webitel [Device] [subscriber_id] client-self identification token header
	H2_X_Client_ID    = "x-webitel-client" // Webitel [Application] [client_id] authorization token header
	H2_X_Access_Token = "x-webitel-access" // Webitel [Messaging] session authorization token header
)

func Coalesce[T comparable](vs ...T) T {
	var zero T
	for _, v := range vs {
		if v != zero {
			return v
		}
	}
	return zero
}

func CoalesceLast[T comparable](vs ...T) T {
	var zero T
	for n := len(vs) - 1; n >= 0; n-- {
		if vs[n] != zero {
			return vs[n]
		}
	}
	return zero
}

func GetHeaderH1(h1 http.Header, key string) string {
	if h1 != nil {
		key = textproto.CanonicalMIMEHeaderKey(key)
		return CoalesceLast(h1[key]...)
	}
	return ""
}

func GetHeaderH2(h2 metadata.MD, key string) string {
	if h2 != nil {
		return CoalesceLast(h2.Get(key)...)
	}
	return ""
}

func ParseRealIP(vs []string) net.Addr {
	// X-Real-IP: 188.230.65.211:41718
	input := CoalesceLast(vs...)
	if len(input) == 0 {
		return nil // NONE
	}
	// MUST: addr:port
	iport, err := netip.ParseAddrPort(input)
	if err == nil && iport.IsValid() {
		return net.TCPAddrFromAddrPort(iport)
	}
	// TRY: addr
	raddr, err := netip.ParseAddr(input)
	if err == nil {
		return &net.IPAddr{
			IP:   raddr.AsSlice(),
			Zone: raddr.Zone(),
			// Port: 0,
		}
	}
	// INVALID spec
	return nil
}

func ParseForwardedFor(vs []string) net.Addr {
	// X-Forwarded-For: <client>, <proxy1>, <proxy2>
	// -----------------------------------------------------------
	// X-Forwarded-For: 2001:db8:85a3:8d3:1319:8a2e:370:7348
	// X-Forwarded-For: 203.0.113.195
	// X-Forwarded-For: 203.0.113.195, 70.41.3.18, 150.172.238.178
	input := CoalesceLast(vs...)
	if len(input) == 0 {
		return nil // NONE
	}
	vs = strings.SplitN(input, ",", 2)
	input = strings.TrimSpace(vs[0])
	// MUST: addr
	raddr, err := netip.ParseAddr(input)
	if err == nil && raddr.IsValid() {
		return &net.IPAddr{
			IP:   raddr.AsSlice(),
			Zone: raddr.Zone(),
		}
	}
	// TRY: addr[:port]
	iport, err := netip.ParseAddrPort(input)
	if err == nil && iport.IsValid() {
		return net.TCPAddrFromAddrPort(iport)
	}
	// INVALID spec
	return nil
}
