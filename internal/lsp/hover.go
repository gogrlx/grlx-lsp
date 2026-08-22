package lsp

import (
	"context"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func (h *Handler) handleHover(_ context.Context, req *jsonrpc2.Request) (any, error) {
	var params protocol.HoverParams
	if err := protocol.Unmarshal(req.Params(), &params); err != nil {
		return nil, err
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil {
		return nil, nil
	}

	lineNumber := int(params.Position.Line)
	line := lineAt(doc.content, lineNumber)
	word := wordAtPosition(line, int(params.Position.Character))

	if word == "" {
		return nil, nil
	}

	if prop := h.findPropertyHover(doc, lineNumber, word); prop != nil {
		return &protocol.Hover{
			Contents: markdownContent(buildPropertyMarkdown(prop.Ingredient, prop.Method, prop.Property)),
		}, nil
	}

	if strings.Contains(word, ".") {
		parts := strings.SplitN(word, ".", 2)
		if len(parts) == 2 {
			m := h.registry.FindMethod(parts[0], parts[1])
			ing := h.registry.FindIngredient(parts[0])
			if m != nil && ing != nil {
				return &protocol.Hover{
					Contents: markdownContent(buildMethodMarkdown(ing.Name, m)),
				}, nil
			}
		}
	}

	ing := h.registry.FindIngredient(word)
	if ing != nil {
		var methods []string
		for _, m := range ing.Methods {
			methods = append(methods, ing.Name+"."+m.Name)
		}
		return &protocol.Hover{
			Contents: markdownContent("**" + ing.Name + "** — " + ing.Description + "\n\nMethods: `" + strings.Join(methods, "`, `") + "`"),
		}, nil
	}

	for _, rt := range h.registry.RequisiteTypes {
		if rt.Name == word {
			return &protocol.Hover{
				Contents: markdownContent("**" + rt.Name + "** — " + rt.Description),
			}, nil
		}
	}

	return nil, nil
}

func markdownContent(value string) *protocol.MarkupContent {
	return &protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: value,
	}
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
