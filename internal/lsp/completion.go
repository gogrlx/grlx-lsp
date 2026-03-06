package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func (h *Handler) handleCompletion(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	ctx := context.Background()

	var params protocol.CompletionParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil {
		return reply(ctx, &protocol.CompletionList{}, nil)
	}

	line := lineAt(doc.content, int(params.Position.Line))
	col := int(params.Position.Character)
	if col > len(line) {
		col = len(line)
	}
	prefix := strings.TrimSpace(line[:col])

	var items []protocol.CompletionItem

	switch {
	case isTopLevel(doc.content, int(params.Position.Line)):
		items = h.completeTopLevel(prefix)
	case isInRequisites(line):
		items = h.completeRequisiteTypes(prefix)
	case isInRequisiteValue(doc.content, int(params.Position.Line)):
		items = h.completeStepIDs(doc)
	case isPropertyPosition(line):
		items = h.completeProperties(doc, int(params.Position.Line))
	default:
		items = h.completeIngredientMethod(prefix)
	}

	return reply(ctx, &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil)
}

func (h *Handler) completeTopLevel(_ string) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, key := range schema.TopLevelKeys {
		items = append(items, protocol.CompletionItem{
			Label: key,
			Kind:  protocol.CompletionItemKindKeyword,
		})
	}
	return items
}

func (h *Handler) completeIngredientMethod(prefix string) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// If prefix contains a dot, complete methods for that ingredient
	if dotIdx := strings.Index(prefix, "."); dotIdx >= 0 {
		ingName := prefix[:dotIdx]
		ing := h.registry.FindIngredient(ingName)
		if ing != nil {
			for _, m := range ing.Methods {
				items = append(items, protocol.CompletionItem{
					Label:         ingName + "." + m.Name,
					Kind:          protocol.CompletionItemKindFunction,
					Detail:        m.Description,
					Documentation: buildMethodDoc(ing, &m),
				})
			}
		}
		return items
	}

	// Otherwise, complete all ingredient.method combos
	for _, name := range h.registry.AllDottedNames() {
		parts := strings.SplitN(name, ".", 2)
		ing := h.registry.FindIngredient(parts[0])
		m := h.registry.FindMethod(parts[0], parts[1])
		detail := ""
		if m != nil {
			detail = m.Description
		}
		doc := ""
		if ing != nil && m != nil {
			doc = buildMethodDoc(ing, m)
		}
		items = append(items, protocol.CompletionItem{
			Label:         name,
			Kind:          protocol.CompletionItemKindFunction,
			Detail:        detail,
			Documentation: doc,
		})
	}
	return items
}

func (h *Handler) completeProperties(doc *document, line int) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Find which step this line belongs to
	step := h.findStepForLine(doc, line)
	if step == nil {
		return items
	}

	m := h.registry.FindMethod(step.Ingredient, step.Method)
	if m == nil {
		return items
	}

	// Collect already-used properties
	used := make(map[string]bool)
	for _, p := range step.Properties {
		used[p.Key] = true
	}

	for _, prop := range m.Properties {
		if used[prop.Key] {
			continue
		}
		detail := prop.Type
		if prop.Required {
			detail += " (required)"
		}
		items = append(items, protocol.CompletionItem{
			Label:      "- " + prop.Key + ": ",
			InsertText: "- " + prop.Key + ": ",
			Kind:       protocol.CompletionItemKindProperty,
			Detail:     detail,
		})
	}

	// Also offer requisites
	if !used["requisites"] {
		items = append(items, protocol.CompletionItem{
			Label:  "- requisites:",
			Kind:   protocol.CompletionItemKindKeyword,
			Detail: "Step dependencies",
		})
	}

	return items
}

func (h *Handler) completeRequisiteTypes(_ string) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	for _, rt := range schema.AllRequisiteTypes {
		items = append(items, protocol.CompletionItem{
			Label:  "- " + rt.Name + ": ",
			Kind:   protocol.CompletionItemKindEnum,
			Detail: rt.Description,
		})
	}
	return items
}

func (h *Handler) completeStepIDs(doc *document) []protocol.CompletionItem {
	var items []protocol.CompletionItem
	if doc.recipe == nil {
		return items
	}
	for _, id := range doc.recipe.StepIDs() {
		items = append(items, protocol.CompletionItem{
			Label: id,
			Kind:  protocol.CompletionItemKindReference,
		})
	}
	return items
}

func buildMethodDoc(ing *schema.Ingredient, m *schema.Method) string {
	var sb strings.Builder
	sb.WriteString(ing.Name + "." + m.Name)
	if m.Description != "" {
		sb.WriteString(" — " + m.Description)
	}
	if len(m.Properties) > 0 {
		sb.WriteString("\n\nProperties:\n")
		for _, p := range m.Properties {
			marker := "  "
			if p.Required {
				marker = "* "
			}
			sb.WriteString(marker + p.Key + " (" + p.Type + ")")
			if p.Description != "" {
				sb.WriteString(" — " + p.Description)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Heuristics for context detection

func isTopLevel(content string, line int) bool {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return false
	}
	l := lines[line]
	// Top-level if no leading whitespace or empty
	return len(l) == 0 || (len(strings.TrimLeft(l, " \t")) == len(l))
}

func isInRequisites(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Inside a requisites block: indented under "- requisites:"
	return strings.HasPrefix(trimmed, "- require") ||
		strings.HasPrefix(trimmed, "- onchanges") ||
		strings.HasPrefix(trimmed, "- onfail")
}

func isInRequisiteValue(content string, line int) bool {
	lines := strings.Split(content, "\n")
	// Look backwards for a requisite condition key
	for i := line; i >= 0 && i >= line-5; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "- require:") ||
			strings.HasPrefix(trimmed, "- require_any:") ||
			strings.HasPrefix(trimmed, "- onchanges:") ||
			strings.HasPrefix(trimmed, "- onchanges_any:") ||
			strings.HasPrefix(trimmed, "- onfail:") ||
			strings.HasPrefix(trimmed, "- onfail_any:") {
			return true
		}
	}
	return false
}

func isPropertyPosition(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Property lines start with "- " and are indented
	indent := len(line) - len(strings.TrimLeft(line, " "))
	return indent >= 4 && (strings.HasPrefix(trimmed, "- ") || trimmed == "-" || trimmed == "")
}

func (h *Handler) findStepForLine(doc *document, line int) *struct {
	Ingredient string
	Method     string
	Properties []struct{ Key string }
} {
	if doc.recipe == nil {
		return nil
	}
	// Find the step whose method node is on or before this line
	for i := len(doc.recipe.Steps) - 1; i >= 0; i-- {
		s := &doc.recipe.Steps[i]
		if s.MethodNode != nil && s.MethodNode.Line-1 <= line {
			result := &struct {
				Ingredient string
				Method     string
				Properties []struct{ Key string }
			}{
				Ingredient: s.Ingredient,
				Method:     s.Method,
			}
			for _, p := range s.Properties {
				result.Properties = append(result.Properties, struct{ Key string }{Key: p.Key})
			}
			return result
		}
	}
	return nil
}
