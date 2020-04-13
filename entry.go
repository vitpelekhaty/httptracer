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
	Response []string `json:"response,omitempty"`
	// Metric request statistic
	Metric Metric `json:"metric"`
	// Error connection error
	Error error `json:"error,omitempty"`
}
