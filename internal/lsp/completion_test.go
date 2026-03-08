package lsp

import (
	"testing"

	"go.lsp.dev/protocol"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestCompleteTopLevel(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := ``
	h.updateDocument("file:///test.grlx", src)

	items := h.completeTopLevel("")
	if len(items) != len(schema.TopLevelKeys) {
		t.Errorf("expected %d top-level items, got %d", len(schema.TopLevelKeys), len(items))
	}
	labelSet := make(map[string]bool)
	for _, item := range items {
		labelSet[item.Label] = true
	}
	for _, key := range schema.TopLevelKeys {
		if !labelSet[key] {
			t.Errorf("missing top-level completion: %s", key)
		}
	}
}

func TestCompleteIngredientMethod(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())

	// Without dot prefix — should return all ingredient.method combos
	items := h.completeIngredientMethod("")
	if len(items) == 0 {
		t.Fatal("expected completion items for all ingredient.method combos")
	}
	labelSet := make(map[string]bool)
	for _, item := range items {
		labelSet[item.Label] = true
	}
	for _, name := range []string{"file.managed", "cmd.run", "pkg.installed", "service.running"} {
		if !labelSet[name] {
			t.Errorf("missing completion: %s", name)
		}
	}

	// With dot prefix — should complete methods for that ingredient
	items = h.completeIngredientMethod("file.")
	if len(items) == 0 {
		t.Fatal("expected method completions for file ingredient")
	}
	for _, item := range items {
		if item.Kind != protocol.CompletionItemKindFunction {
			t.Errorf("expected Function kind, got %v", item.Kind)
		}
	}

	// Unknown ingredient with dot
	items = h.completeIngredientMethod("bogus.")
	if len(items) != 0 {
		t.Errorf("expected no completions for unknown ingredient, got %d", len(items))
	}
}

func TestCompleteProperties(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  install nginx:
    pkg.installed:
      - name: nginx
      - `
	h.updateDocument("file:///test.grlx", src)
	doc := h.getDocument("file:///test.grlx")

	// Line 4 is inside pkg.installed properties, "name" is already used
	items := h.completeProperties(doc, 4)

	// Should offer "version" but NOT "name" (already used)
	for _, item := range items {
		if item.Label == "- name: " {
			t.Error("should not offer already-used property 'name'")
		}
	}

	// Should offer requisites
	foundRequisites := false
	for _, item := range items {
		if item.Label == "- requisites:" {
			foundRequisites = true
		}
	}
	if !foundRequisites {
		t.Error("expected requisites in property completions")
	}
}

func TestCompletePropertiesNoStep(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:`
	h.updateDocument("file:///test.grlx", src)
	doc := h.getDocument("file:///test.grlx")

	items := h.completeProperties(doc, 0)
	if len(items) != 0 {
		t.Errorf("expected no completions when no step found, got %d", len(items))
	}
}

func TestCompleteRequisiteTypes(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())

	items := h.completeRequisiteTypes("")
	if len(items) != len(schema.AllRequisiteTypes) {
		t.Errorf("expected %d requisite types, got %d", len(schema.AllRequisiteTypes), len(items))
	}
	for _, item := range items {
		if item.Kind != protocol.CompletionItemKindEnum {
			t.Errorf("expected Enum kind for requisite, got %v", item.Kind)
		}
	}
}

func TestCompleteStepIDs(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  install nginx:
    pkg.installed:
      - name: nginx
  start nginx:
    service.running:
      - name: nginx`
	h.updateDocument("file:///test.grlx", src)
	doc := h.getDocument("file:///test.grlx")

	items := h.completeStepIDs(doc)
	if len(items) != 2 {
		t.Errorf("expected 2 step IDs, got %d", len(items))
	}
	labelSet := make(map[string]bool)
	for _, item := range items {
		labelSet[item.Label] = true
	}
	if !labelSet["install nginx"] || !labelSet["start nginx"] {
		t.Errorf("missing expected step IDs, got: %v", labelSet)
	}
}

func TestCompleteStepIDsNilRecipe(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	doc := &document{content: "", recipe: nil}

	items := h.completeStepIDs(doc)
	if len(items) != 0 {
		t.Errorf("expected no step IDs for nil recipe, got %d", len(items))
	}
}

func TestIsTopLevel(t *testing.T) {
	content := "steps:\n  install:\n    pkg.installed:"
	if !isTopLevel(content, 0) {
		t.Error("line 0 should be top-level")
	}
	if isTopLevel(content, 1) {
		t.Error("line 1 should not be top-level (indented)")
	}
	if isTopLevel(content, 99) {
		t.Error("out-of-bounds line should not be top-level")
	}
	if isTopLevel(content, -1) {
		t.Error("negative line should not be top-level")
	}
}

func TestIsInRequisites(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"      - require: step1", true},
		{"      - onchanges: step1", true},
		{"      - onfail: step1", true},
		{"      - name: foo", false},
		{"steps:", false},
	}
	for _, tt := range tests {
		got := isInRequisites(tt.line)
		if got != tt.want {
			t.Errorf("isInRequisites(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestIsInRequisiteValue(t *testing.T) {
	content := `steps:
  first:
    file.exists:
      - name: /tmp/a
      - requisites:
        - require:
          - first`

	if !isInRequisiteValue(content, 6) {
		t.Error("line 6 should be in requisite value context")
	}
	if isInRequisiteValue(content, 0) {
		t.Error("line 0 should not be in requisite value context")
	}
}

func TestIsPropertyPosition(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"    - name: foo", true},
		{"    - ", true},
		{"    ", true},
		{"steps:", false},
		{"  - name: foo", false}, // only 2 spaces indent
	}
	for _, tt := range tests {
		got := isPropertyPosition(tt.line)
		if got != tt.want {
			t.Errorf("isPropertyPosition(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestBuildMethodDoc(t *testing.T) {
	ing := &schema.Ingredient{Name: "file", Description: "Manage files"}
	m := &schema.Method{
		Name:        "managed",
		Description: "Download and manage a file",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true, Description: "The file path"},
			{Key: "source", Type: "string", Required: true, Description: "Source URL"},
			{Key: "mode", Type: "string", Required: false},
		},
	}

	doc := buildMethodDoc(ing, m)
	if doc == "" {
		t.Fatal("expected non-empty doc string")
	}
	// Required properties should be marked with *
	if !contains(doc, "* name") {
		t.Error("expected required marker for 'name'")
	}
	if !contains(doc, "* source") {
		t.Error("expected required marker for 'source'")
	}
}

func TestFindStepForLine(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  install nginx:
    pkg.installed:
      - name: nginx
  start nginx:
    service.running:
      - name: nginx`
	h.updateDocument("file:///test.grlx", src)
	doc := h.getDocument("file:///test.grlx")

	// Line 3 is inside "install nginx" step
	step := h.findStepForLine(doc, 3)
	if step == nil {
		t.Fatal("expected to find step for line 3")
	}
	if step.Ingredient != "pkg" || step.Method != "installed" {
		t.Errorf("expected pkg.installed, got %s.%s", step.Ingredient, step.Method)
	}

	// Line 6 is inside "start nginx" step
	step = h.findStepForLine(doc, 6)
	if step == nil {
		t.Fatal("expected to find step for line 6")
	}
	if step.Ingredient != "service" || step.Method != "running" {
		t.Errorf("expected service.running, got %s.%s", step.Ingredient, step.Method)
	}

	// Nil recipe
	step = h.findStepForLine(&document{content: "", recipe: nil}, 0)
	if step != nil {
		t.Error("expected nil for nil recipe")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
