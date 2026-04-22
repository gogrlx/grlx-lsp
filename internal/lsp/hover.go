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

	lineNumber := int(params.Position.Line)
	line := lineAt(doc.content, lineNumber)
	word := wordAtPosition(line, int(params.Position.Character))

	if word == "" {
		return reply(ctx, nil, nil)
	}

	if prop := h.findPropertyHover(doc, lineNumber, word); prop != nil {
		return reply(ctx, &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: buildPropertyMarkdown(prop.Ingredient, prop.Method, prop.Property),
			},
		}, nil)
	}

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

type propertyHoverMatch struct {
	Ingredient string
	Method     string
	Property   schema.Property
}

func (h *Handler) findPropertyHover(doc *document, lineNumber int, word string) *propertyHoverMatch {
	if doc.recipe == nil {
		return nil
	}

	for _, step := range doc.recipe.Steps {
		if step.MethodNode == nil || step.MethodNode.Line-1 > lineNumber {
			continue
		}
		for _, prop := range step.Properties {
			if prop.Key != word || prop.KeyNode == nil || prop.KeyNode.Line-1 != lineNumber {
				continue
			}
			method := h.registry.FindMethod(step.Ingredient, step.Method)
			if method == nil {
				return nil
			}
			for _, schemaProp := range method.Properties {
				if schemaProp.Key == prop.Key {
					return &propertyHoverMatch{
						Ingredient: step.Ingredient,
						Method:     step.Method,
						Property:   schemaProp,
					}
				}
			}
		}
	}

	return nil
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

func buildPropertyMarkdown(ingredient, method string, property schema.Property) string {
	var sb strings.Builder
	sb.WriteString("### `" + property.Key + "`\n\n")
	sb.WriteString("Used by `" + ingredient + "." + method + "`\n\n")
	sb.WriteString("- Type: `" + property.Type + "`\n")
	if property.Required {
		sb.WriteString("- Required: yes\n")
	} else {
		sb.WriteString("- Required: no\n")
	}
	if property.Description != "" {
		sb.WriteString("\n" + property.Description)
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
