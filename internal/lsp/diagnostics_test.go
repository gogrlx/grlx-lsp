package lsp

import (
	"testing"

	"go.lsp.dev/protocol"
	"gopkg.in/yaml.v3"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestDiagnoseNilRecipe(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	doc := &document{content: "", recipe: nil}
	diags := h.diagnose(doc)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for nil recipe, got %d", len(diags))
	}
}

func TestDiagnoseParseErrors(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  bad step:
    not.a.valid.format:
      - name: foo`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}
	diags := h.diagnose(doc)
	// Should at least report the unknown ingredient
	if len(diags) == 0 {
		t.Error("expected diagnostics for invalid recipe")
	}
}

func TestDiagnoseEmptyIngredient(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	// Step with empty ingredient should be skipped
	r := &recipe.Recipe{
		Steps: []recipe.Step{
			{ID: "test", Ingredient: "", Method: "run"},
		},
	}
	doc := &document{content: "", recipe: r}
	diags := h.diagnose(doc)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for empty ingredient, got %d", len(diags))
	}
}

func TestDiagnoseRequisiteUnknownRef(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  first:
    file.exists:
      - name: /tmp/a
  second:
    file.exists:
      - name: /tmp/b
      - requisites:
        - require: nonexistent_step`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}
	diags := h.diagnose(doc)
	found := false
	for _, d := range diags {
		if d.Severity == protocol.DiagnosticSeverityWarning && contains(d.Message, "reference to unknown step") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for reference to unknown step")
	}
}

func TestIsValidRequisiteType(t *testing.T) {
	validTypes := []string{"require", "require_any", "onchanges", "onchanges_any", "onfail", "onfail_any"}
	for _, rt := range validTypes {
		if !isValidRequisiteType(rt) {
			t.Errorf("expected %q to be valid requisite type", rt)
		}
	}
	if isValidRequisiteType("bogus") {
		t.Error("expected 'bogus' to be invalid requisite type")
	}
}

func TestPointRange(t *testing.T) {
	r := pointRange(5, 10)
	if r.Start.Line != 5 || r.Start.Character != 10 {
		t.Errorf("unexpected start: %v", r.Start)
	}
	if r.End.Line != 5 || r.End.Character != 11 {
		t.Errorf("unexpected end: %v", r.End)
	}

	// Negative values should clamp to 0
	r = pointRange(-1, -5)
	if r.Start.Line != 0 || r.Start.Character != 0 {
		t.Errorf("expected clamped to 0, got: %v", r.Start)
	}
}

func TestYamlNodeRange(t *testing.T) {
	// Nil node
	r := yamlNodeRange(nil)
	if r.Start.Line != 0 || r.Start.Character != 0 {
		t.Errorf("expected (0,0) for nil node, got: %v", r.Start)
	}

	// Normal node
	node := &yaml.Node{Line: 3, Column: 5, Value: "hello"}
	r = yamlNodeRange(node)
	if r.Start.Line != 2 || r.Start.Character != 4 {
		t.Errorf("expected (2,4), got: (%d,%d)", r.Start.Line, r.Start.Character)
	}
	if r.End.Character != 9 { // 4 + len("hello")
		t.Errorf("expected end char 9, got: %d", r.End.Character)
	}
}

func TestPublishDiagnosticsNilConn(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  test:
    pkg.installed:
      - name: nginx`
	h.updateDocument("file:///test.grlx", src)

	// Should not panic with nil conn
	h.publishDiagnostics(t.Context(), "file:///test.grlx")
}

func TestPublishDiagnosticsNoDoc(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	// Should not panic when document doesn't exist
	h.publishDiagnostics(t.Context(), "file:///nonexistent.grlx")
}

func TestCheckRequiredAllPresent(t *testing.T) {
	m := &schema.Method{
		Name: "managed",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true},
			{Key: "source", Type: "string", Required: true},
		},
	}
	s := recipe.Step{
		Properties: []recipe.PropertyEntry{
			{Key: "name"},
			{Key: "source"},
		},
	}
	diags := checkRequired(s, m)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics when all required present, got %d", len(diags))
	}
}

func TestCheckUnknownNoUnknowns(t *testing.T) {
	m := &schema.Method{
		Name: "installed",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true},
			{Key: "version", Type: "string"},
		},
	}
	s := recipe.Step{
		Ingredient: "pkg",
		Method:     "installed",
		Properties: []recipe.PropertyEntry{
			{Key: "name", KeyNode: &yaml.Node{Line: 1, Column: 1, Value: "name"}},
		},
	}
	diags := checkUnknown(s, m)
	if len(diags) != 0 {
		t.Errorf("expected no unknown property diagnostics, got %d", len(diags))
	}
}

func TestCheckUnknownRequisitesAllowed(t *testing.T) {
	m := &schema.Method{
		Name: "installed",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true},
		},
	}
	s := recipe.Step{
		Ingredient: "pkg",
		Method:     "installed",
		Properties: []recipe.PropertyEntry{
			{Key: "name", KeyNode: &yaml.Node{Line: 1, Column: 1, Value: "name"}},
			{Key: "requisites", KeyNode: &yaml.Node{Line: 2, Column: 1, Value: "requisites"}},
		},
	}
	diags := checkUnknown(s, m)
	if len(diags) != 0 {
		t.Errorf("'requisites' should be allowed, got diagnostics: %v", diags)
	}
}
