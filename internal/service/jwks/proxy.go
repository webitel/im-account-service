package jwks

import (
	"io"
	"net/http"

	"github.com/lestrrat-go/httprc/v3"
)

type httpProxy struct {
  client httprc.HTTPClient
}

// NOTE: with such impl jwk.Cache.Refresh(!) unable to get an error =((
func (c httpProxy) Do(req *http.Request) (res *http.Response, err error) {

  client := c.client
  if client == nil {
    client = http.DefaultClient
  }

  res, err = client.Do(req)

  // [ OK ] ?
  if err == nil {
    return res, nil
  }

  // [ ERR ]
  code := http.StatusOK
  res = &http.Response{
  	Status:           "", // http.StatusText(code),
  	StatusCode:       code,
  	Proto:            req.Proto,
  	ProtoMajor:       req.ProtoMajor,
  	ProtoMinor:       req.ProtoMinor,
  	Header:           http.Header{},
  	Body:             errProxy{err},
  	ContentLength:    0,
  	TransferEncoding: nil, // []string{},
  	Close:            true,
  	Uncompressed:     false,
  	Trailer:          nil, // http.Header{},
  	Request:          req,
  	TLS:              nil, // &tls.ConnectionState{},
  }

  return res, nil
}

type errProxy struct {
  err error
}

var _ io.ReadCloser = errProxy{}

func (e errProxy) Read(_ []byte) (_ int, err error) {
	return 0, e.err
}

func (e errProxy) Close() error {
	return nil
}

