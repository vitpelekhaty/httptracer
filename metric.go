package httptracer

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"
)

// Metric request duration metrics (time in nanoseconds)
type Metric struct {
	DNSLookup        time.Duration `json:"dns-lookup"`
	TCPConnection    time.Duration `json:"tcp-connection"`
	TLSHandshake     time.Duration `json:"tls-handshake"`
	ServerProcessing time.Duration `json:"server-processing"`
	ContentTransfer  time.Duration `json:"content-transfer"`
	NameLookup       time.Duration `json:"name-lookup"`
	Connect          time.Duration `json:"connect"`
	PreTransfer      time.Duration `json:"pre-transfer"`
	StartTransfer    time.Duration `json:"start-transfer"`
	Total            time.Duration `json:"total"`
}

func (t *Tracer) GotConn(_ httptrace.GotConnInfo) {
	t.time[3] = time.Now()
}

func (t *Tracer) GotFirstResponseByte() {
	t.time[4] = time.Now()
}

func (t *Tracer) DNSStart(_ httptrace.DNSStartInfo) {
	t.time[0] = time.Now()
}

func (t *Tracer) DNSDone(_ httptrace.DNSDoneInfo) {
	t.time[1] = time.Now()
}

func (t *Tracer) ConnectStart(network, addr string) {
	if t.time[1].IsZero() {
		t.time[1] = time.Now()
	}
}

func (t *Tracer) ConnectDone(network, addr string, err error) {
	if err == nil {
		t.time[2] = time.Now()
		return
	}

	t.err = err
}

func (t *Tracer) TLSHandshakeStart() {
	t.time[5] = time.Now()
}

func (t *Tracer) TLSHandshakeDone(_ tls.ConnectionState, err error) {
	t.time[6] = time.Now()
	t.err = err
}

func (t *Tracer) metric(req *http.Request) Metric {
	switch req.URL.Scheme {
	case "https":
		return t.HTTPSMetric()
	default:
		return t.HTTPMetric()
	}
}

func (t *Tracer) HTTPMetric() Metric {
	return Metric{
		DNSLookup:        t.time[1].Sub(t.time[0]),
		TCPConnection:    t.time[3].Sub(t.time[1]),
		ServerProcessing: t.time[4].Sub(t.time[3]),
		ContentTransfer:  t.time[7].Sub(t.time[4]),
		NameLookup:       t.time[1].Sub(t.time[0]),
		Connect:          t.time[3].Sub(t.time[0]),
		StartTransfer:    t.time[4].Sub(t.time[0]),
		Total:            t.time[7].Sub(t.time[0]),
	}
}

func (t *Tracer) HTTPSMetric() Metric {
	return Metric{
		DNSLookup:        t.time[1].Sub(t.time[0]),
		TCPConnection:    t.time[2].Sub(t.time[1]),
		TLSHandshake:     t.time[6].Sub(t.time[5]),
		ServerProcessing: t.time[4].Sub(t.time[3]),
		ContentTransfer:  t.time[7].Sub(t.time[4]),
		NameLookup:       t.time[1].Sub(t.time[0]),
		Connect:          t.time[2].Sub(t.time[0]),
		PreTransfer:      t.time[3].Sub(t.time[0]),
		StartTransfer:    t.time[4].Sub(t.time[0]),
		Total:            t.time[7].Sub(t.time[0]),
	}
}
