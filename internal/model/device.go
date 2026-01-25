package model

import (
	"context"
	"crypto/md5"
	"net"
	"strings"

	ua "github.com/mileusna/useragent"
	v1 "github.com/webitel/im-account-service/proto/gen/im/service/auth/v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// Device (Client) Endpoint
type Device struct {
	// Id   UUID                 // OPTIONAL. Internal Device ID. Empty means NOT registered ; othewise: generate & remember
	// Sub  string               // OPTIONAL. Subscriber ID; Client-side SELF idenitification UNIQUE Device ID; IDFA, GAID, etc
	// Name string               // Device (Session) name

	Id   string               // OPTIONAL. Subscriber ID; Client-side SELF idenitification UNIQUE Device ID; IDFA, GAID, etc
	App  ua.UserAgent         // User-Agent: details
	Addr net.Addr             // Remote (Client) IP address [FROM]
	From []net.IP             // historical: addresses ever seen [FROM]
	Push *v1.PUSHSubscription // OPTIONAL. PUSH subscription for async notifications
}

// List of Device(s). End-User session conformity
type DeviceList Dataset[Device]

// [FROM] Remote IP address
func (peer *Device) IP() (ip net.IP) {
	if peer != nil {
		ip = AddrIP(peer.Addr)
	}
	return // ip | nil
}

// Hash of sensitive data
func (peer *Device) Hash() []byte {
	// MD5(device|mobile|tablet|desktop|bot|os|name)
	var (
		ua = &peer.App
		fd byte // uint8
	)
	for x, ok := range []bool{
		ua.Mobile,
		ua.Tablet,
		ua.Desktop,
		ua.Bot,
		// MAX: 8
	} {
		if ok {
			fd |= (1 << x)
		}
	}
	hash := md5.New()
	hash.Write([]byte(ua.Device))
	hash.Write([]byte{fd})
	hash.Write([]byte{'|'})
	hash.Write([]byte(ua.OS))
	hash.Write([]byte{'|'})
	hash.Write([]byte(ua.Name))
	return hash.Sum(nil)
}

// Device kind, e.g.: web | mobile | tablet | desktop | bot
func (peer *Device) Type() string {
	var (
		typeOf = "web"
		uainfo = &peer.App
	)
	if uainfo.Mobile {
		typeOf = "mobile"
	} else if uainfo.Tablet {
		typeOf = "tablet"
	} else if uainfo.Desktop {
		typeOf = "desktop"
	} else if uainfo.Bot {
		typeOf = "bot"
	}
	return typeOf
}

// Push unique [peer.Addr] to [peer.From] history front.
// True indicates a completely new [peer.Addr] in history.
func (peer *Device) FromAddr() (isNew bool) {
	ip := peer.IP()
	if len(ip) == 0 {
		return false
	}
	var (
		// ever used []addr
		data = peer.From
		e, n = 0, len(data)
	)
	for ; e < n && !ip.Equal(data[e]); e++ {
		// lookup: net.IP address duplicate
	}
	if isNew = (e == n); isNew {
		// NEW ; NOT FOUND !
		if n > 0 {
			// PUSH TO FRONT !
			data = append(data, nil) // grow
			copy(data[1:], data[0:n])
			data[0] = ip // peer.Addr
		} else {
			// FIRST
			data = append(data, ip)
		}
	} else {
		// EXISTS
		if e > 0 {
			// NOT ON TOP
			// MOVE TO FRONT
			copy(data[1:e+1], data[0:e])
			data[0] = ip // peer.Addr
		}
	}
	peer.From = data
	return // isNew
}

// Remote (Client) address [FROM]
func RemoteAddr(ctx context.Context) (from net.Addr) {
	// HTTP/2.* Metadata
	h2, _ := metadata.FromIncomingContext(ctx)
	// if len(h2) == 0 {
	// 	return // noop, false
	// }

	// Remote Addr
	resolve := []func() net.Addr{
		// [X-Forwarded-For]
		func() net.Addr {
			return ParseForwardedFor(
				h2.Get(H2_X_Forwarded_For),
			)
		},
		// [X-Real-IP]
		func() net.Addr {
			return ParseRealIP(
				h2.Get(H2_X_Real_IP),
			)
		},
		// google.golang.org/grpc/peer.Addr
		func() net.Addr {
			peer, _ := peer.FromContext(ctx)
			if peer != nil {
				return peer.Addr
			}
			return nil
		},
	}

	next, max := 0, len(resolve)
	for next < max {
		// determine
		from = resolve[next]()
		if from != nil {
			break
		}
		// continue
		next++
	}

	return // from
}

// Grab [User-Agent] as a remote client connection [Device] info
func GetDeviceAuthorization(ctx context.Context) (peer Device, ok bool) {

	h2, _ := metadata.FromIncomingContext(ctx)
	if len(h2) == 0 {
		return // noop, false
	}

	// :authority:      dev.webitel.com
	// content-type:    application/grpc
	// user-agent:      grpc-go/1.57.0
	// x-forwarded-for: 188.230.65.211[, <proxy1>]...
	// x-real-ip:       188.230.65.211:41718
	// ---------------------------------------------------
	peer.Addr = RemoteAddr(ctx)
	// User-Agent
	peer.App = ua.UserAgent{
		String: GetHeaderH2(
			h2, H2_User_Agent,
		),
	}
	// User-Agent:
	if s := peer.App.String; len(s) > 0 {
		peer.App = ua.Parse(s)
		// peer.Name = DeviceName(&from)
		ok = true
	}

	// // historical
	// _ = peer.FromAddr()
	// peer.From = []net.Addr{
	// 	peer.Addr,
	// }

	// [X-Webitel-Device] ; OPTIONAL
	peer.Id = strings.TrimSpace(
		GetHeaderH2(h2, H2_X_Device_ID),
	)
	// if service, id, ok := Split from.Sub != "" {}

	ok = (ok || peer.Addr != nil)
	return // from, ok
}

// form session name of client endpoint device info
func SessionName(peer *Device) (name string) {
	var (
		info = &peer.App
		form strings.Builder
	)
	defer func() {
		name = form.String()
	}()
	if info.Device != "" {
		form.WriteString(info.Device) // Dev
		form.WriteString(" (")
		defer form.WriteString(")")
	}
	form.WriteString(info.Name) // App
	if info.Version != "" {
		form.WriteString("/" + info.Version) // Ver
	}
	if info.OS != "" {
		form.WriteString("; " + info.OS)
		if info.OSVersion != "" {
			form.WriteString(" " + info.OSVersion)
		}
	}
	return // name = form.String()
}
