package httptracer

import (
	"time"
)

// Entry trace entry
type Entry struct {
	// Time of request
	Time time.Time `json:"time"`
	// Request dump
	Request []string `json:"request"`
	// Response dump
	Response []string `json:"response"`
	// Stat request statistic
	Stat Stat `json:"stat"`
	// Error connection error
	Error error `json:"error"`
}

// Stat request statistic (nanoseconds)
type Stat struct {
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
