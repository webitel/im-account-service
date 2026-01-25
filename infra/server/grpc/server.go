package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	"github.com/webitel/im-account-service/config"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

var Module = fx.Module(
	"grpc_server", fx.Provide(
		func(conf *config.Config, logger *slog.Logger, lc fx.Lifecycle) (*Server, error) {

			srv, err := New(conf.Service.Address, logger)
			if err != nil {
				return nil, err
			}

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						logger.Info(fmt.Sprintf("Server [grpc] Listening on %s:%d", srv.Host(), srv.Port()))
						if err := srv.Listen(); err != nil {
							logger.Error("grpc server error", "error", err)
						}
					}()

					return nil
				},
				OnStop: func(ctx context.Context) error {
					if err := srv.Shutdown(); err != nil {
						logger.Error("error stopping grpc server", "error", err.Error())

						return err
					}

					return nil
				},
			})

			return srv, nil
		}),
)

type Server struct {
	Addr string
	host string
	port int
	log  *slog.Logger
	*grpc.Server
	listener net.Listener
}

// New provides a new gRPC server.
func New(addr string, log *slog.Logger) (*Server, error) {

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(),
		grpc.StatsHandler(newServiceHandler(log)),
	)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	h, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(p)

	if h == "::" {
		h = publicAddr()
	}

	return &Server{
		Addr:     addr,
		Server:   s,
		log:      log,
		host:     h,
		port:     port,
		listener: l,
	}, nil
}

func (s *Server) Listen() error {
	return s.Serve(s.listener)
}

func (s *Server) Shutdown() error {
	s.log.Debug("receive shutdown grpc")
	s.Server.GracefulStop()
	// return s.listener.Close() // FIXME: already closed ??? close tcp 127.0.0.1:26010: use of closed network connection
	// err := s.listener.Close()
	// return err
	return nil
}

func (s *Server) Host() string {
	if e, ok := os.LookupEnv("PROXY_GRPC_HOST"); ok {
		return e
	}
	return s.host
}

func (s *Server) Port() int {
	return s.port
}

func publicAddr() string {
	interfaces, err := net.Interfaces()

	if err != nil {
		return ""
	}
	for _, i := range interfaces {
		addresses, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			if isPublicIP(ip) {
				return ip.String()
			}
			// process IP address
		}
	}
	return ""
}

func isPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	return !IP.IsPrivate() // true
}
