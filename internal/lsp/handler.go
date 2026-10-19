// Package lsp implements the Language Server Protocol handler for grlx recipes.
package lsp

import (
	"context"
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
func (h *Handler) Handle(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	switch req.Method() {
	case "initialize":
		return h.handleInitialize(ctx, req)
	case "initialized":
		return nil, nil
	case "shutdown":
		return nil, nil
	case "exit":
		return nil, nil
	case "textDocument/didOpen":
		return h.handleDidOpen(ctx, req)
	case "textDocument/didChange":
		return h.handleDidChange(ctx, req)
	case "textDocument/didClose":
		return h.handleDidClose(ctx, req)
	case "textDocument/didSave":
		return nil, nil
	case "textDocument/completion":
		return h.handleCompletion(ctx, req)
	case "textDocument/hover":
		return h.handleHover(ctx, req)
	case "textDocument/definition":
		return h.handleDefinition(ctx, req)
	case "textDocument/diagnostic":
		return nil, nil
	default:
		if !req.IsCall() {
			return nil, nil
		}
		return nil, jsonrpc2.NewError(jsonrpc2.MethodNotFound, "method not supported: "+req.Method())
	}
}

func (h *Handler) handleInitialize(_ context.Context, _ *jsonrpc2.Request) (any, error) {
	openClose := true
	changeKind := protocol.TextDocumentSyncKindFull
	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: &openClose,
				Change:    &changeKind,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{".", ":", "-", " "},
			},
			HoverProvider:      protocol.Boolean(true),
			DefinitionProvider: protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{
			Name:    "grlx-lsp",
			Version: protocol.NewOptional("0.1.0"),
		},
	}, nil
}

func (h *Handler) handleDidOpen(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.DidOpenTextDocumentParams
	if err := protocol.Unmarshal(req.Params(), &params); err != nil {
		// Notifications must not fail the connection on a bad payload.
		return nil, nil
	}
	h.updateDocument(string(params.TextDocument.URI), params.TextDocument.Text)
	h.publishDiagnostics(ctx, string(params.TextDocument.URI))
	return nil, nil
}

func (h *Handler) handleDidChange(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.DidChangeTextDocumentParams
	if err := protocol.Unmarshal(req.Params(), &params); err != nil {
		// Notifications must not fail the connection on a bad payload.
		return nil, nil
	}
	if len(params.ContentChanges) > 0 {
		h.updateDocument(string(params.TextDocument.URI), contentChangeText(params.ContentChanges[len(params.ContentChanges)-1]))
		h.publishDiagnostics(ctx, string(params.TextDocument.URI))
	}
	return nil, nil
}

func contentChangeText(change protocol.TextDocumentContentChangeEvent) string {
	switch change := change.(type) {
	case *protocol.TextDocumentContentChangeWholeDocument:
		return change.Text
	case *protocol.TextDocumentContentChangePartial:
		return change.Text
	default:
		return ""
	}
}

func (h *Handler) handleDidClose(ctx context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.DidCloseTextDocumentParams
	if err := protocol.Unmarshal(req.Params(), &params); err != nil {
		// Notifications must not fail the connection on a bad payload.
		return nil, nil
	}
	h.mu.Lock()
	delete(h.docs, string(params.TextDocument.URI))
	h.mu.Unlock()
	// Diagnostics belong to the server; closing a document must clear them.
	if h.conn != nil {
		_ = h.conn.Notify(ctx, "textDocument/publishDiagnostics", protocol.PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []protocol.Diagnostic{},
		})
	}
	return nil, nil
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
