package middlewares

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/getfider/fider/app/pkg/web"
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

// Compress returns a middleware which compresses HTTP response using gzip compression
func Compress() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			if strings.Contains(c.Request.GetHeader("Accept-Encoding"), "gzip") {
				res := c.Response
				res.Header().Set("Content-Encoding", "gzip")
				res.Header().Del("Accept-Encoding")
				res.Header().Add("Vary", "Accept-Encoding")

				gw, _ := gzip.NewWriterLevel(res.Writer, gzip.DefaultCompression)

				c.Response.Writer = &gzipResponseWriter{
					Writer:         gw,
					ResponseWriter: c.Response.Writer,
				}

				err := next(c)
				_ = gw.Close()
				return err
			}
			return next(c)
		}
	}
}

// Unwrap exposes the underlying writer so http.NewResponseController can
// reach the real connection (per-route deadline extension on AI routes).
func (w *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(code)

}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	header := w.ResponseWriter.Header()
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", http.DetectContentType(b))
	}
	header.Del("Content-Length")

	return w.Writer.Write(b)
}

// Flush pushes the compressed bytes written so far to the client. Streaming
// responses (the MCP SDK's event stream) depend on it. Writer is the
// *gzip.Writer, which is not an http.Flusher: asserting it was one panicked on
// an SDK goroutine, beyond CatchPanic, and took the whole process down.
func (w *gzipResponseWriter) Flush() {
	if gw, ok := w.Writer.(*gzip.Writer); ok {
		_ = gw.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack hands over the underlying connection when it supports that.
func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
