package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"

	"github.com/webitel/im-account-service/config"
	infra_tls "github.com/webitel/im-account-service/infra/tls"
	"github.com/webitel/im-account-service/infra/x/grpcx"
	"github.com/webitel/im-account-service/infra/x/logx"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var Module = fx.Module(
	"grpc_server", fx.Provide(
		func(config *config.Config, logger *slog.Logger, ssl *infra_tls.Config, runtime fx.Lifecycle) (*Server, error) {

			var creds *tls.Config
			if ssl != nil {
				creds = ssl.Server
			}

			srv, err := New(config.Service.Address, logger, creds)
			if err != nil {
				return nil, err
			}

			runtime.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						logger.Info(fmt.Sprintf("[ server ] Listening [grpc] %s", srv.Addr)) // %s:%d", srv.Host(), srv.Port()))
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
func New(addr string, log *slog.Logger, ssl *tls.Config) (*Server, error) {

	serverOpts := []grpc.ServerOption{
		// grpc.ChainUnaryInterceptor(),
		
	}

	// Configure TLS if provided
	if ssl != nil {
		serverOpts = append(serverOpts, grpc.Creds(
			credentials.NewTLS(ssl),
		))
	}

	if logx.Debug("grpc") {
		serverOpts = append(serverOpts,
			grpc.StatsHandler(grpcx.DumpHandler(func(opts *grpcx.DumpOptions) {
				opts.Debug = slog.LevelDebug
				opts.Logger = logx.ModuleLogger("im-account-server", log)
			})),
		)
	}

	s := grpc.NewServer(serverOpts...)

	ls, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	addr = ls.Addr().String()
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	port, _ := strconv.Atoi(p)

	// IPv6 ; listening on all available interfaces ?
	if h == "::" {
		h = publicAddr()
	}

	return &Server{
		Addr:     addr,
		Server:   s,
		log:      log,
		host:     h,
		port:     port,
		listener: ls,
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

// Advertise returns the address to be advertised to other services.
func (s *Server) Advertise() string {
	port := strconv.Itoa(s.port)
	return net.JoinHostPort(s.Host(), port)
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
	return "" // "127.0.0.1"
}

func isPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	return !IP.IsPrivate() // true
}
