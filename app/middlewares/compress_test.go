package middlewares_test

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getfider/fider/app/middlewares"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

func TestCompress(t *testing.T) {
	RegisterT(t)

	data := "Hello World\n"
	for i := 0; i <= 500; i++ {
		data += "Hello World\n"
	}

	server := mock.NewServer()
	server.Use(middlewares.Compress())
	handler := func(c *web.Context) error {
		return c.String(http.StatusOK, data)
	}

	status, response := server.
		AddHeader("Accept-Encoding", "gzip").
		Execute(handler)

	reader, _ := gzip.NewReader(response.Body)
	bytes, _ := io.ReadAll(reader)
	Expect(bytes).Equals([]byte(data))
	Expect(status).Equals(http.StatusOK)
	Expect(response.Header().Get("Vary")).Equals("Accept-Encoding")
	Expect(response.Header().Get("Content-Type")).Equals("text/plain; charset=utf-8")
	Expect(response.Header().Get("Content-Encoding")).Equals("gzip")
}

// Streaming handlers (the MCP SDK's event stream) flush through
// http.ResponseController. Flushing must push the compressed bytes to the
// client, never panic: a panic there runs on an SDK goroutine, outside
// CatchPanic, and takes the whole process down.
func TestCompress_Flush(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Compress())
	var flushErr error
	var flushed int
	handler := func(c *web.Context) error {
		_, _ = c.Response.Writer.Write([]byte("event: message\n\n"))
		flushErr = http.NewResponseController(c.Response.Writer).Flush()
		client := c.Response.Writer.(interface{ Unwrap() http.ResponseWriter }).Unwrap()
		flushed = client.(*httptest.ResponseRecorder).Body.Len()
		return nil
	}

	status, response := server.
		AddHeader("Accept-Encoding", "gzip").
		Execute(handler)

	Expect(flushErr).IsNil()
	Expect(flushed > 0).IsTrue()
	reader, _ := gzip.NewReader(response.Body)
	bytes, _ := io.ReadAll(reader)
	Expect(string(bytes)).Equals("event: message\n\n")
	Expect(status).Equals(http.StatusOK)
}

// A compressed response cannot be hijacked; say so instead of panicking.
func TestCompress_HijackNotSupported(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.Compress())
	var hijackErr error
	handler := func(c *web.Context) error {
		_, _, hijackErr = http.NewResponseController(c.Response.Writer).Hijack()
		return c.String(http.StatusOK, "ok")
	}

	server.AddHeader("Accept-Encoding", "gzip").Execute(handler)

	Expect(errors.Is(hijackErr, http.ErrNotSupported)).IsTrue()
}

func TestCompress_AfterPanic(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	server.Use(middlewares.CatchPanic())
	server.Use(middlewares.Compress())
	handler := func(c *web.Context) error {
		panic("Boom!")
	}

	status, response := server.
		AddHeader("Accept-Encoding", "gzip").
		Execute(handler)

	Expect(status).Equals(http.StatusInternalServerError)
	Expect(response.Header().Get("Vary")).Equals("Accept-Encoding")
	Expect(response.Header().Get("Content-Type")).Equals("text/html; charset=utf-8")
	Expect(response.Header().Get("Content-Encoding")).Equals("gzip")
}
