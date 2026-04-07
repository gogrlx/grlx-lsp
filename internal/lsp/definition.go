package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (h *Handler) handleDefinition(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	ctx := context.Background()

	var params protocol.DefinitionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil || doc.recipe == nil {
		return reply(ctx, nil, nil)
	}

	line := lineAt(doc.content, int(params.Position.Line))
	col := int(params.Position.Character)

	// Only provide definition inside requisite value positions
	if !isInRequisiteValue(doc.content, int(params.Position.Line)) {
		return reply(ctx, nil, nil)
	}

	// Extract the step ID reference under the cursor
	ref := stepRefAtPosition(line, col)
	if ref == "" {
		return reply(ctx, nil, nil)
	}

	// Find the step definition
	for _, s := range doc.recipe.Steps {
		if s.ID == ref && s.IDNode != nil {
			loc := protocol.Location{
				URI: params.TextDocument.URI,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(s.IDNode.Line - 1),
						Character: uint32(s.IDNode.Column - 1),
					},
					End: protocol.Position{
						Line:      uint32(s.IDNode.Line - 1),
						Character: uint32(s.IDNode.Column - 1 + len(s.ID)),
					},
				},
			}
			return reply(ctx, loc, nil)
		}
	}

	return reply(ctx, nil, nil)
}

// stepRefAtPosition extracts a step ID reference from a requisite value line.
// Requisite values appear after "- <condition>: " as plain text.
func stepRefAtPosition(line string, col int) string {
	trimmed := strings.TrimSpace(line)

	// Look for pattern "- <condition>: <step ref>"
	for _, prefix := range []string{
		"- require: ", "- require_any: ",
		"- onchanges: ", "- onchanges_any: ",
		"- onfail: ", "- onfail_any: ",
	} {
		if strings.HasPrefix(trimmed, prefix) {
			// The rest of the trimmed line is the step ID
			ref := strings.TrimSpace(trimmed[len(prefix):])
			if ref == "" {
				return ""
			}
			// Check the cursor is actually in the value portion
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			valueStart := indent + len(prefix)
			if col >= valueStart {
				return ref
			}
		}
	}

	return ""
}
