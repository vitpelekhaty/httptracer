package httptracer

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"time"
)

// Tracer http tracer
type Tracer struct {
	writer    io.Writer
	trace     *httptrace.ClientTrace
	transport http.RoundTripper
	bodies    bool
}

// Option tracer option
type Option func(t *Tracer)

func WithWriter(writer io.Writer) Option {
	return func(t *Tracer) {
		t.writer = writer
	}
}

func WithBodies(value bool) Option {
	return func(t *Tracer) {
		t.bodies = value
	}
}

var times [8]time.Time

// New constructor of Tracer
func New(transport http.RoundTripper, options ...Option) *Tracer {
	ct := &httptrace.ClientTrace{
		GotConn: func(_ httptrace.GotConnInfo) {
			times[3] = time.Now()
		},
		GotFirstResponseByte: func() {
			times[4] = time.Now()
		},
		DNSStart: func(_ httptrace.DNSStartInfo) {
			times[0] = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			times[1] = time.Now()
		},
		ConnectStart: func(network, addr string) {
			if times[1].IsZero() {
				times[1] = time.Now()
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if err == nil {
				times[2] = time.Now()
				return
			}
		},
		TLSHandshakeStart: func() {
			times[5] = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			times[6] = time.Now()
		},
	}

	t := &Tracer{
		transport: transport,
		trace:     ct,
	}

	for _, option := range options {
		option(t)
	}

	return t
}

func (t *Tracer) RoundTrip(req *http.Request) (*http.Response, error) {
	entry := &Entry{Time: time.Now()}

	defer func() {
		t.writeEntry(entry)
	}()

	entry.Request = string(t.requestDump(req))

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), t.trace))
	resp, err := t.transport.RoundTrip(req)

	times[7] = time.Now()

	if times[0].IsZero() {
		times[0] = times[1]
	}

	entry.Error = err

	if err == nil {
		entry.Response = string(t.responseDump(resp))
	}

	entry.Stat = t.stat(req)

	return resp, err
}

func (t *Tracer) writeEntry(entry *Entry) error {
	if entry == nil || t.writer == nil {
		return nil
	}

	data, err := json.Marshal(entry)

	if err != nil {
		return err
	}

	_, err = t.writer.Write(data)

	return err
}

func (t *Tracer) requestDump(req *http.Request) []byte {
	dump, err := httputil.DumpRequest(req, t.bodies)

	if err != nil {
		return make([]byte, 0)
	}

	return dump
}

func (t *Tracer) responseDump(resp *http.Response) []byte {
	dump, err := httputil.DumpResponse(resp, t.bodies)

	if err != nil {
		return make([]byte, 0)
	}

	return dump
}

func (t *Tracer) stat(req *http.Request) Stat {
	switch req.URL.Scheme {
	case "https":
		return t.statHTTPS()
	default:
		return t.statHTTP()
	}
}

func (t *Tracer) statHTTP() Stat {
	return Stat{
		DNSLookup:        times[1].Sub(times[0]),
		TCPConnection:    times[3].Sub(times[1]),
		ServerProcessing: times[4].Sub(times[3]),
		ContentTransfer:  times[7].Sub(times[4]),
		NameLookup:       times[1].Sub(times[0]),
		Connect:          times[3].Sub(times[0]),
		StartTransfer:    times[4].Sub(times[0]),
		Total:            times[7].Sub(times[0]),
	}
}

func (t *Tracer) statHTTPS() Stat {
	return Stat{
		DNSLookup:        times[1].Sub(times[0]),
		TCPConnection:    times[2].Sub(times[1]),
		TLSHandshake:     times[6].Sub(times[5]),
		ServerProcessing: times[4].Sub(times[3]),
		ContentTransfer:  times[7].Sub(times[4]),
		NameLookup:       times[1].Sub(times[0]),
		Connect:          times[2].Sub(times[0]),
		PreTransfer:      times[3].Sub(times[0]),
		StartTransfer:    times[4].Sub(times[0]),
		Total:            times[7].Sub(times[0]),
	}
}
