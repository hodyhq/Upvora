package middlewares_test

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/getfider/fider/app/middlewares"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
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

// The production crash, end to end: an MCP client on the 2026-07-28 protocol
// opens a subscriptions/listen stream on connect, and the server flushes its
// acknowledgement through Compress. Options as in mcpserver.Handler.
func TestCompress_MCPSubscriptionsListen(t *testing.T) {
	RegisterT(t)

	engine := mock.NewServer().Engine()
	server := sdk.NewServer(&sdk.Implementation{Name: "upvora", Version: "1"}, nil)
	sdk.AddTool(server, &sdk.Tool{Name: "t1"}, func(context.Context, *sdk.CallToolRequest, struct{}) (*sdk.CallToolResult, any, error) {
		return &sdk.CallToolResult{}, nil, nil
	})
	mcp := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server },
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	compressed := middlewares.Compress()(func(c *web.Context) error {
		raw := c.Request.Raw().Clone(c)
		raw.Body = io.NopCloser(strings.NewReader(c.Request.Body))
		mcp.ServeHTTP(c.Response.Writer, raw)
		return nil
	})
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = compressed(web.NewContext(engine, r, w, nil))
	}))
	defer httpServer.Close()

	acked := make(chan struct{}, 1)
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "1"}, &sdk.ClientOptions{
		ToolListChangedHandler: func(context.Context, *sdk.ToolListChangedRequest) {},
	})
	client.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if method == "notifications/subscriptions/acknowledged" {
				select {
				case acked <- struct{}{}:
				default:
				}
			}
			return next(ctx, method, req)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	Expect(err).IsNil()
	if session != nil {
		defer func() { _ = session.Close() }()
	}

	select {
	case <-acked:
	case <-ctx.Done():
		t.Fatal("no notifications/subscriptions/acknowledged on the listen stream")
	}
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
