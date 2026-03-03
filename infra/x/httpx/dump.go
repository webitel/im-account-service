package httpx

import (
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/google/uuid"
)

type TransportDump struct {
  Transport http.RoundTripper
  WithBody bool
}

func (x TransportDump) RoundTrip(req *http.Request) (res *http.Response, err error) {

  // region: DUMP Request
	reqId, _ := uuid.NewRandom() // fmt.Sprintf("%p", h.Context())
	dump, err := httputil.DumpRequestOut(req, x.WithBody && req.ContentLength > 0)
	
	tracef := log.Printf // stdlog.Tracef
	if err != nil {
		tracef = log.Printf // stdlog.Errorf
		dump = []byte("httputil.DumpRequestOut: "+ err.Error())
	}
	tracef("\t>>>>> OUTBOUND (%s) >>>>>\n\n%s\n\n", reqId, dump)
	// endregion
	
	// PERFORM !
	resp, err := x.Transport.RoundTrip(req)
	
	if err != nil {
		tracef = log.Printf // stdlog.Errorf
		dump = []byte("error: "+ err.Error())
		tracef("\t<<<<< RESPONSE (%s) <<<<<\n\n%s\n\n", reqId, dump)
		// Failure(!)
		return resp, err
	}

	// region: DUMP Response
	dump, err = httputil.DumpResponse(resp, x.WithBody && resp.ContentLength > 0)
	
	tracef = log.Printf // stdlog.Tracef
	if err != nil {
		tracef = log.Printf // stdlog.Errorf
		dump = []byte("httputil.DumpResponse: "+ err.Error())
	}
	tracef("\t<<<<< RESPONSE (%s) <<<<<\n\n%s\n\n", reqId, dump)
	// endregion

	// Success(!)
	return resp, err
}

