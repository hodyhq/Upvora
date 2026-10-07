package mcpserver

import (
	"bytes"
	"net/http"
	"unicode/utf8"
)

// CappedRecorder is an http.ResponseWriter that keeps at most cap bytes of
// the body, so a large /api/v1 response is never held in full in memory.
type CappedRecorder struct {
	header    http.Header
	code      int
	buf       bytes.Buffer
	cap       int
	truncated bool
}

// NewCappedRecorder records up to cap bytes of a response.
func NewCappedRecorder(cap int) *CappedRecorder {
	return &CappedRecorder{header: http.Header{}, code: http.StatusOK, cap: cap}
}

func (r *CappedRecorder) Header() http.Header  { return r.header }
func (r *CappedRecorder) WriteHeader(code int) { r.code = code }

// Write keeps what fits and reports the full length, so handlers never fail.
func (r *CappedRecorder) Write(p []byte) (int, error) {
	if room := r.cap - r.buf.Len(); room > 0 {
		if len(p) > room {
			r.buf.Write(p[:room])
			r.truncated = true
		} else {
			r.buf.Write(p)
		}
	} else if len(p) > 0 {
		r.truncated = true
	}
	return len(p), nil
}

// Body is the retained body, cut back to a whole UTF-8 character if truncated.
func (r *CappedRecorder) Body() string {
	b := r.buf.Bytes()
	if r.truncated {
		for len(b) > 0 && !utf8.Valid(b) {
			b = b[:len(b)-1]
		}
	}
	return string(b)
}

// Truncated reports whether the response was longer than the cap.
func (r *CappedRecorder) Truncated() bool { return r.truncated }

// Code is the response status.
func (r *CappedRecorder) Code() int { return r.code }
