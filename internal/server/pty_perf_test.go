package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

func BenchmarkPTYInputHTTPRoundTrip(b *testing.B) {
	handler, url, ptyID := benchmarkPTYServer(b)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn := benchmarkAttachPTY(b, ctx, url, ptyID)
	defer conn.Close(websocket.StatusNormalClosure, "")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		benchmarkPostNoContent(b, handler, "/v1/ptys/"+ptyID+"/write", protocol.WritePTYRequest{Data: "x"})
		benchmarkReadPTYStreamFrame(b, ctx, conn)
	}
}

func BenchmarkPTYInputWebSocketRoundTrip(b *testing.B) {
	_, url, ptyID := benchmarkPTYServer(b)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn := benchmarkAttachPTY(b, ctx, url, ptyID)
	defer conn.Close(websocket.StatusNormalClosure, "")
	frame := []byte(`{"type":"input","ptyId":"` + ptyID + `","data":"x"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i += 1 {
		if err := conn.Write(ctx, websocket.MessageText, frame); err != nil {
			b.Fatalf("write websocket input: %v", err)
		}
		benchmarkReadPTYStreamFrame(b, ctx, conn)
	}
}

func benchmarkPTYServer(b *testing.B) (http.Handler, string, string) {
	b.Helper()
	backend := newFakePTYBackend()
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: backend, EventSink: newFakeEventBus()})
	handler := server.NewHTTP(runtime)
	httpServer := httptest.NewServer(handler)
	b.Cleanup(httpServer.Close)
	created := benchmarkPostJSON[protocol.CreatedSession](b, handler, "/v1/sessions", protocol.CreateSessionRequest{
		Name:       "Whisk",
		RootDir:    b.TempDir(),
		InitialPTY: &protocol.StartPTYOptions{Cols: 80, Rows: 24},
	})
	return handler, httpServer.URL, created.MainPtyID
}

func benchmarkAttachPTY(b *testing.B, ctx context.Context, url string, ptyID string) *websocket.Conn {
	b.Helper()
	conn, _, err := websocket.Dial(ctx, strings.Replace(url, "http", "ws", 1)+"/v1/ptys/"+ptyID+"/attach?from=0", nil)
	if err != nil {
		b.Fatalf("dial attach: %v", err)
	}
	return conn
}

func benchmarkPostNoContent(tb testing.TB, handler http.Handler, path string, body any) {
	tb.Helper()
	rec := httptest.NewRecorder()
	data, err := json.Marshal(body)
	if err != nil {
		tb.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("content-type", "application/json")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		tb.Fatalf("POST %s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
}

func benchmarkPostJSON[T any](tb testing.TB, handler http.Handler, path string, body any) T {
	tb.Helper()
	rec := httptest.NewRecorder()
	data, err := json.Marshal(body)
	if err != nil {
		tb.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("content-type", "application/json")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		tb.Fatalf("POST %s status = %d body = %s", path, rec.Code, rec.Body.String())
	}
	raw, err := io.ReadAll(rec.Body)
	if err != nil {
		tb.Fatalf("read body: %v", err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		tb.Fatalf("decode body: %v", err)
	}
	return out
}

func benchmarkReadPTYStreamFrame(tb testing.TB, ctx context.Context, conn *websocket.Conn) protocol.PTYStreamFrame {
	tb.Helper()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		tb.Fatalf("read websocket frame: %v", err)
	}
	if typ != websocket.MessageText {
		tb.Fatalf("websocket message type = %v", typ)
	}
	var frame protocol.PTYStreamFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		tb.Fatalf("decode websocket frame: %v", err)
	}
	return frame
}
