package lsp

import (
	"context"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestCloseClearsDiagnostics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	serverStream, clientStream := jsonrpc2.NewChannelStreamPair(8)
	server := jsonrpc2.NewConn(serverStream)
	client := jsonrpc2.NewConn(clientStream)
	defer server.Close()
	defer client.Close()
	handler := NewHandler(schema.DefaultRegistry())
	handler.SetConn(server)
	server.Go(ctx, handler.Handle)
	notifications := make(chan protocol.PublishDiagnosticsParams, 8)
	client.Go(ctx, func(_ context.Context, request *jsonrpc2.Request) (any, error) {
		if request.Method() == "textDocument/publishDiagnostics" {
			var params protocol.PublishDiagnosticsParams
			if err := protocol.Unmarshal(request.Params(), &params); err != nil {
				return nil, err
			}
			notifications <- params
		}
		return nil, nil
	})
	const docURI = "file:///lifecycle.grlx"
	if err := client.Notify(ctx, "textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{"uri": docURI, "languageId": "grlx", "version": 1, "text": "steps: [invalid]"},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case params := <-notifications:
		if len(params.Diagnostics) == 0 {
			t.Fatal("expected initial parse diagnostic")
		}
	case <-ctx.Done():
		t.Fatal("no initial diagnostics received")
	}
	if err := client.Notify(ctx, "textDocument/didClose", map[string]any{
		"textDocument": map[string]any{"uri": docURI},
	}); err != nil {
		t.Fatal(err)
	}
	select {
	case params := <-notifications:
		if string(params.URI) != docURI || len(params.Diagnostics) != 0 {
			t.Fatalf("expected empty diagnostics for closed URI, got %+v", params)
		}
		if handler.getDocument(docURI) != nil {
			t.Fatal("closed document retained")
		}
	case <-ctx.Done():
		t.Fatal("closing document did not clear diagnostics")
	}
}
