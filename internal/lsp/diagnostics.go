package lsp

import (
	"context"

	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func (h *Handler) publishDiagnostics(ctx context.Context, uri string) {
	doc := h.getDocument(uri)
	if doc == nil || h.conn == nil {
		return
	}

	diags := h.diagnose(doc)

	_ = h.conn.Notify(ctx, "textDocument/publishDiagnostics", protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentURI(uri),
		Diagnostics: diags,
	})
}

func (h *Handler) diagnose(doc *document) []protocol.Diagnostic {
	var diags []protocol.Diagnostic

	if doc.recipe == nil {
		return diags
	}

	// Report parse errors
	for _, e := range doc.recipe.Errors {
		diags = append(diags, protocol.Diagnostic{
			Range:    pointRange(e.Line-1, e.Col-1),
			Severity: protocol.DiagnosticSeverityError,
			Source:   "grlx-lsp",
			Message:  e.Message,
		})
	}

	stepIDs := make(map[string]bool)
	for _, s := range doc.recipe.Steps {
		stepIDs[s.ID] = true
	}

	for _, s := range doc.recipe.Steps {
		if s.Ingredient == "" {
			continue
		}

		ing := h.registry.FindIngredient(s.Ingredient)
		if ing == nil {
			diags = append(diags, protocol.Diagnostic{
				Range:    yamlNodeRange(s.MethodNode),
				Severity: protocol.DiagnosticSeverityError,
				Source:   "grlx-lsp",
				Message:  "unknown ingredient: " + s.Ingredient,
			})
			continue
		}

		m := h.registry.FindMethod(s.Ingredient, s.Method)
		if m == nil {
			diags = append(diags, protocol.Diagnostic{
				Range:    yamlNodeRange(s.MethodNode),
				Severity: protocol.DiagnosticSeverityError,
				Source:   "grlx-lsp",
				Message:  "unknown method: " + s.Ingredient + "." + s.Method,
			})
			continue
		}

		diags = append(diags, checkRequired(s, m)...)
		diags = append(diags, checkUnknown(s, m)...)

		// Validate requisite types and references
		for _, req := range s.Requisites {
			if !isValidRequisiteType(req.Condition) {
				diags = append(diags, protocol.Diagnostic{
					Range:    yamlNodeRange(req.Node),
					Severity: protocol.DiagnosticSeverityError,
					Source:   "grlx-lsp",
					Message:  "unknown requisite type: " + req.Condition,
				})
			}
			for _, ref := range req.StepIDs {
				if !stepIDs[ref] {
					diags = append(diags, protocol.Diagnostic{
						Range:    yamlNodeRange(req.Node),
						Severity: protocol.DiagnosticSeverityWarning,
						Source:   "grlx-lsp",
						Message:  "reference to unknown step: " + ref + " (may be defined in an included recipe)",
					})
				}
			}
		}
	}

	return diags
}

func checkRequired(s recipe.Step, m *schema.Method) []protocol.Diagnostic {
	var diags []protocol.Diagnostic
	propKeys := make(map[string]bool)
	for _, p := range s.Properties {
		propKeys[p.Key] = true
	}
	for _, prop := range m.Properties {
		if prop.Required && !propKeys[prop.Key] {
			diags = append(diags, protocol.Diagnostic{
				Range:    yamlNodeRange(s.MethodNode),
				Severity: protocol.DiagnosticSeverityWarning,
				Source:   "grlx-lsp",
				Message:  "missing required property: " + prop.Key,
			})
		}
	}
	return diags
}

func checkUnknown(s recipe.Step, m *schema.Method) []protocol.Diagnostic {
	var diags []protocol.Diagnostic
	validProps := make(map[string]bool)
	for _, prop := range m.Properties {
		validProps[prop.Key] = true
	}
	validProps["requisites"] = true
	for _, p := range s.Properties {
		if !validProps[p.Key] {
			diags = append(diags, protocol.Diagnostic{
				Range:    yamlNodeRange(p.KeyNode),
				Severity: protocol.DiagnosticSeverityWarning,
				Source:   "grlx-lsp",
				Message:  "unknown property: " + p.Key + " for " + s.Ingredient + "." + s.Method,
			})
		}
	}
	return diags
}

func isValidRequisiteType(name string) bool {
	for _, rt := range schema.AllRequisiteTypes {
		if rt.Name == name {
			return true
		}
	}
	return false
}

func pointRange(line, col int) protocol.Range {
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	return protocol.Range{
		Start: protocol.Position{Line: uint32(line), Character: uint32(col)},
		End:   protocol.Position{Line: uint32(line), Character: uint32(col + 1)},
	}
}

func yamlNodeRange(node *yaml.Node) protocol.Range {
	if node == nil {
		return pointRange(0, 0)
	}
	line := node.Line - 1
	col := node.Column - 1
	endCol := col + len(node.Value)
	if line < 0 {
		line = 0
	}
	if col < 0 {
		col = 0
	}
	return protocol.Range{
		Start: protocol.Position{Line: uint32(line), Character: uint32(col)},
		End:   protocol.Position{Line: uint32(line), Character: uint32(endCol)},
	}
}
