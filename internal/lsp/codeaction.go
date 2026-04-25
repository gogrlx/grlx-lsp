package lsp

import (
	"context"
	"encoding/json"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

func (h *Handler) handleCodeAction(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	ctx := context.Background()

	var params protocol.CodeActionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil || doc.recipe == nil {
		return reply(ctx, []protocol.CodeAction{}, nil)
	}

	var actions []protocol.CodeAction

	for _, diag := range params.Context.Diagnostics {
		if diag.Source != "grlx-lsp" {
			continue
		}

		for _, s := range doc.recipe.Steps {
			if s.MethodNode == nil {
				continue
			}

			m := h.registry.FindMethod(s.Ingredient, s.Method)
			if m == nil {
				continue
			}

			methodLine := uint32(s.MethodNode.Line - 1)
			if diag.Range.Start.Line != methodLine {
				continue
			}

			for _, prop := range m.Properties {
				if !prop.Required {
					continue
				}
				if diag.Message != "missing required property: "+prop.Key {
					continue
				}

				indent := s.MethodNode.Column - 1 + 2
				spaces := ""
				for range indent {
					spaces += " "
				}
				insertLine := methodLine + 1
				newText := spaces + "- " + prop.Key + ": \n"

				actions = append(actions, protocol.CodeAction{
					Title:       "Add required property: " + prop.Key,
					Kind:        protocol.QuickFix,
					Diagnostics: []protocol.Diagnostic{diag},
					Edit: &protocol.WorkspaceEdit{
						Changes: map[protocol.DocumentURI][]protocol.TextEdit{
							params.TextDocument.URI: {
								{
									Range: protocol.Range{
										Start: protocol.Position{Line: insertLine, Character: 0},
										End:   protocol.Position{Line: insertLine, Character: 0},
									},
									NewText: newText,
								},
							},
						},
					},
				})
			}
		}
	}

	return reply(ctx, actions, nil)
}
