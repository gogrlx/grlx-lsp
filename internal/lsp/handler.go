// Package lsp implements the Language Server Protocol handler for grlx recipes.
package lsp

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

// Handler implements the LSP server.
type Handler struct {
	conn     jsonrpc2.Conn
	registry *schema.Registry
	mu       sync.RWMutex
	docs     map[string]*document // URI -> document
}

type document struct {
	content string
	recipe  *recipe.Recipe
}

// NewHandler creates a new LSP handler.
func NewHandler(registry *schema.Registry) *Handler {
	return &Handler{
		registry: registry,
		docs:     make(map[string]*document),
	}
}

// SetConn sets the jsonrpc2 connection for sending notifications.
func (h *Handler) SetConn(conn jsonrpc2.Conn) {
	h.conn = conn
}

// Handle dispatches LSP requests.
func (h *Handler) Handle(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	switch req.Method() {
	case "initialize":
		return h.handleInitialize(ctx, reply, req)
	case "initialized":
		return reply(ctx, nil, nil)
	case "shutdown":
		return reply(ctx, nil, nil)
	case "exit":
		return reply(ctx, nil, nil)
	case "textDocument/didOpen":
		return h.handleDidOpen(ctx, reply, req)
	case "textDocument/didChange":
		return h.handleDidChange(ctx, reply, req)
	case "textDocument/didClose":
		return h.handleDidClose(ctx, reply, req)
	case "textDocument/didSave":
		return reply(ctx, nil, nil)
	case "textDocument/completion":
		return h.handleCompletion(ctx, reply, req)
	case "textDocument/hover":
		return h.handleHover(ctx, reply, req)
	case "textDocument/diagnostic":
		return reply(ctx, nil, nil)
	default:
		return reply(ctx, nil, jsonrpc2.NewError(jsonrpc2.MethodNotFound, "method not supported: "+req.Method()))
	}
}

func (h *Handler) handleInitialize(_ context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
	return reply(context.Background(), protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{".", ":", "-", " "},
			},
			HoverProvider: true,
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "grlx-lsp",
			Version: "0.1.0",
		},
	}, nil)
}

func (h *Handler) handleDidOpen(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidOpenTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}
	h.updateDocument(string(params.TextDocument.URI), params.TextDocument.Text)
	h.publishDiagnostics(ctx, string(params.TextDocument.URI))
	return reply(ctx, nil, nil)
}

func (h *Handler) handleDidChange(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidChangeTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}
	if len(params.ContentChanges) > 0 {
		h.updateDocument(string(params.TextDocument.URI), params.ContentChanges[len(params.ContentChanges)-1].Text)
		h.publishDiagnostics(ctx, string(params.TextDocument.URI))
	}
	return reply(ctx, nil, nil)
}

func (h *Handler) handleDidClose(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.DidCloseTextDocumentParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(context.Background(), nil, err)
	}
	h.mu.Lock()
	delete(h.docs, string(params.TextDocument.URI))
	h.mu.Unlock()
	return reply(context.Background(), nil, nil)
}

func (h *Handler) updateDocument(uri, content string) {
	r := recipe.Parse([]byte(content))
	h.mu.Lock()
	h.docs[uri] = &document{content: content, recipe: r}
	h.mu.Unlock()
}

func (h *Handler) getDocument(uri string) *document {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.docs[uri]
}

// lineAt returns the text of the given line (0-indexed).
func lineAt(content string, line int) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}
