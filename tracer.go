package httptracer

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/http/httputil"
	"time"
)

// Trace wraps a client for tracing
func Trace(client *http.Client, options ...Option) *http.Client {
	if client.Transport != nil {
		tracer := New(client.Transport, options...)
		client.Transport = tracer

		return client
	}

	tracer := New(http.DefaultTransport, options...)
	client.Transport = tracer

	return client
}

// Tracer http tracer
type Tracer struct {
	writer    io.Writer
	trace     *httptrace.ClientTrace
	transport http.RoundTripper
	bodies    bool
	cfn       CallbackFunc
}

// Option tracer option
type Option func(t *Tracer)

// WithWriter sets a writer
func WithWriter(writer io.Writer) Option {
	return func(t *Tracer) {
		t.writer = writer
	}
}

// WithBodies indicates the need to read a request/response body
func WithBodies(value bool) Option {
	return func(t *Tracer) {
		t.bodies = value
	}
}

// CallbackFunc Tracer callback function.
// Function called when another entry was added
type CallbackFunc func(entry *Entry)

// WithCallback sets a callback function
func WithCallback(callback CallbackFunc) Option {
	return func(t *Tracer) {
		t.cfn = callback
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

// RoundTrip implementation of http.RoundTripper interface
func (t *Tracer) RoundTrip(req *http.Request) (*http.Response, error) {
	entry := &Entry{Time: time.Now()}

	defer func() {
		if err := t.writeEntry(entry); err == nil {
			if t.cfn != nil {
				t.cfn(entry)
			}
		}
	}()

	entry.Request = t.requestDump(req)

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), t.trace))
	resp, err := t.transport.RoundTrip(req)

	times[7] = time.Now()

	if times[0].IsZero() {
		times[0] = times[1]
	}

	entry.Error = err

	if err == nil {
		entry.Response = t.responseDump(resp)
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

func (t *Tracer) requestDump(req *http.Request) []string {
	dump, err := httputil.DumpRequest(req, t.bodies)

	if err != nil {
		return make([]string, 0)
	}

	return t.dumpLines(dump)
}

func (t *Tracer) responseDump(resp *http.Response) []string {
	dump, err := httputil.DumpResponse(resp, t.bodies)

	if err != nil {
		return make([]string, 0)
	}

	return t.dumpLines(dump)
}

func (t *Tracer) dumpLines(dump []byte) []string {
	lines := make([]string, 0)

	if len(dump) == 0 {
		return lines
	}

	scanner := bufio.NewScanner(bytes.NewReader(dump))
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
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
