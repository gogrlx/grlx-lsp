// Package lsp implements the Language Server Protocol handler for grlx recipes.
package lsp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

// Handler implements the LSP server.
type Handler struct {
	conn          jsonrpc2.Conn
	registry      *schema.Registry
	mu            sync.RWMutex
	docs          map[string]*document // URI -> document
	workspaceRoot string               // filesystem path of workspace root
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
	case "textDocument/definition":
		return h.handleDefinition(ctx, reply, req)
	case "textDocument/diagnostic":
		return reply(ctx, nil, nil)
	case "textDocument/semanticTokens/full":
		return h.handleSemanticTokensFull(ctx, reply, req)
	case "textDocument/codeAction":
		return h.handleCodeAction(ctx, reply, req)
	default:
		return reply(ctx, nil, jsonrpc2.NewError(jsonrpc2.MethodNotFound, "method not supported: "+req.Method()))
	}
}

func (h *Handler) handleInitialize(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	var params protocol.InitializeParams
	if err := json.Unmarshal(req.Params(), &params); err == nil {
		if len(params.WorkspaceFolders) > 0 {
			h.workspaceRoot = uriToPath(string(params.WorkspaceFolders[0].URI))
		} else {
			rootURI := string(params.RootURI) //lint:ignore SA1019 fallback for older clients
			if rootURI != "" {
				h.workspaceRoot = uriToPath(rootURI)
			}
		}
	}
	return reply(context.Background(), protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: protocol.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    protocol.TextDocumentSyncKindFull,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{".", ":", "-", " "},
			},
			HoverProvider:      true,
			DefinitionProvider: true,
			CodeActionProvider: &protocol.CodeActionOptions{
				CodeActionKinds: []protocol.CodeActionKind{protocol.QuickFix},
			},
			SemanticTokensProvider: map[string]any{
				"legend": map[string]any{
					"tokenTypes":     semanticTokenTypes,
					"tokenModifiers": []string{},
				},
				"full": true,
			},
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

// uriToPath converts a file:// URI to a filesystem path.
func uriToPath(uri string) string {
	const prefix = "file://"
	if strings.HasPrefix(uri, prefix) {
		return strings.TrimPrefix(uri, prefix)
	}
	return uri
}

// pathToURI converts a filesystem path to a file:// URI.
func pathToURI(path string) string {
	if strings.HasPrefix(path, "/") {
		return "file://" + path
	}
	return "file:///" + path
}

// collectIncludedStepIDs parses all included recipes and returns their step IDs.
func (h *Handler) collectIncludedStepIDs(uri string, doc *document) []string {
	if doc.recipe == nil || h.workspaceRoot == "" {
		return nil
	}
	currentFile := uriToPath(uri)
	seen := make(map[string]bool)
	for _, inc := range doc.recipe.Includes {
		resolved, ok := recipe.ResolveInclude(h.workspaceRoot, currentFile, inc.Value)
		if !ok {
			continue
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		incRecipe := recipe.Parse(data)
		for _, id := range incRecipe.StepIDs() {
			seen[id] = true
		}
	}
	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}
