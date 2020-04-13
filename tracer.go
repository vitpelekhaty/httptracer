package httptracer

import (
	"bufio"
	"bytes"
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
	time      [8]time.Time
	err       error
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

// New constructor of Tracer
func New(transport http.RoundTripper, options ...Option) *Tracer {
	t := &Tracer{
		transport: transport,
	}

	for _, option := range options {
		option(t)
	}

	ct := &httptrace.ClientTrace{
		GotConn:              t.GotConn,
		GotFirstResponseByte: t.GotFirstResponseByte,
		DNSStart:             t.DNSStart,
		DNSDone:              t.DNSDone,
		ConnectStart:         t.ConnectStart,
		ConnectDone:          t.ConnectDone,
		TLSHandshakeStart:    t.TLSHandshakeStart,
		TLSHandshakeDone:     t.TLSHandshakeDone,
	}

	t.trace = ct

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

	t.time[7] = time.Now()

	if t.time[0].IsZero() {
		t.time[0] = t.time[1]
	}

	entry.Error = t.err

	if err == nil {
		entry.Response = t.responseDump(resp)
	}

	entry.Metric = t.metric(req)

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
