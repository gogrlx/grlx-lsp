package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func (h *Handler) handleHover(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	ctx := context.Background()

	var params protocol.HoverParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil {
		return reply(ctx, nil, nil)
	}

	line := lineAt(doc.content, int(params.Position.Line))
	word := wordAtPosition(line, int(params.Position.Character))

	if word == "" {
		return reply(ctx, nil, nil)
	}

	// Check if it's an ingredient.method reference
	if strings.Contains(word, ".") {
		parts := strings.SplitN(word, ".", 2)
		if len(parts) == 2 {
			m := h.registry.FindMethod(parts[0], parts[1])
			ing := h.registry.FindIngredient(parts[0])
			if m != nil && ing != nil {
				return reply(ctx, &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.Markdown,
						Value: buildMethodMarkdown(ing.Name, m),
					},
				}, nil)
			}
		}
	}

	// Check if it's just an ingredient name
	ing := h.registry.FindIngredient(word)
	if ing != nil {
		var methods []string
		for _, m := range ing.Methods {
			methods = append(methods, ing.Name+"."+m.Name)
		}
		return reply(ctx, &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: "**" + ing.Name + "** — " + ing.Description + "\n\nMethods: `" + strings.Join(methods, "`, `") + "`",
			},
		}, nil)
	}

	// Check if it's a requisite type
	for _, rt := range h.registry.RequisiteTypes {
		if rt.Name == word {
			return reply(ctx, &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.Markdown,
					Value: "**" + rt.Name + "** — " + rt.Description,
				},
			}, nil)
		}
	}

	return reply(ctx, nil, nil)
}

func buildMethodMarkdown(ingredient string, m *schema.Method) string {
	var sb strings.Builder
	sb.WriteString("### " + ingredient + "." + m.Name + "\n\n")
	if m.Description != "" {
		sb.WriteString(m.Description + "\n\n")
	}
	if len(m.Properties) > 0 {
		sb.WriteString("| Property | Type | Required | Description |\n")
		sb.WriteString("|----------|------|----------|-------------|\n")
		for _, p := range m.Properties {
			req := ""
			if p.Required {
				req = "yes"
			}
			desc := p.Description
			if desc == "" {
				desc = "—"
			}
			sb.WriteString("| `" + p.Key + "` | " + p.Type + " | " + req + " | " + desc + " |\n")
		}
	}
	return sb.String()
}

func wordAtPosition(line string, col int) string {
	if col > len(line) {
		col = len(line)
	}

	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	end := col
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') ||
		b == '_' || b == '.' || b == '-'
}
