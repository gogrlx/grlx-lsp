package lsp

import (
	"testing"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestDiagnoseUnknownIngredient(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	doc := &document{
		content: `steps:
  bad step:
    bogus.method:
      - name: foo`,
		recipe: recipe.Parse([]byte(`steps:
  bad step:
    bogus.method:
      - name: foo`)),
	}

	diags := h.diagnose(doc)
	found := false
	for _, d := range diags {
		if d.Message == "unknown ingredient: bogus" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown ingredient diagnostic, got: %v", diags)
	}
}

func TestDiagnoseUnknownMethod(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  bad step:
    file.nonexistent:
      - name: foo`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	diags := h.diagnose(doc)
	found := false
	for _, d := range diags {
		if d.Message == "unknown method: file.nonexistent" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown method diagnostic, got: %v", diags)
	}
}

func TestDiagnoseMissingRequired(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	// file.managed requires both name and source
	src := `steps:
  manage file:
    file.managed:
      - user: root`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	diags := h.diagnose(doc)
	foundName := false
	foundSource := false
	for _, d := range diags {
		if d.Message == "missing required property: name" {
			foundName = true
		}
		if d.Message == "missing required property: source" {
			foundSource = true
		}
	}
	if !foundName {
		t.Error("expected diagnostic for missing required property: name")
	}
	if !foundSource {
		t.Error("expected diagnostic for missing required property: source")
	}
}

func TestDiagnoseUnknownProperty(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  my step:
    file.absent:
      - name: /tmp/foo
      - bogusprop: bar`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	diags := h.diagnose(doc)
	found := false
	for _, d := range diags {
		if d.Message == "unknown property: bogusprop for file.absent" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown property diagnostic, got: %v", diags)
	}
}

func TestDiagnoseValidRecipe(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  install nginx:
    pkg.installed:
      - name: nginx`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	diags := h.diagnose(doc)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for valid recipe, got: %v", diags)
	}
}

func TestDiagnoseUnknownRequisiteType(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  first:
    file.exists:
      - name: /tmp/a
  second:
    file.exists:
      - name: /tmp/b
      - requisites:
        - bogus_req: first`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	diags := h.diagnose(doc)
	found := false
	for _, d := range diags {
		if d.Message == "unknown requisite type: bogus_req" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown requisite type diagnostic, got: %v", diags)
	}
}

func TestLineAt(t *testing.T) {
	content := "line0\nline1\nline2"
	if got := lineAt(content, 0); got != "line0" {
		t.Errorf("lineAt(0) = %q, want %q", got, "line0")
	}
	if got := lineAt(content, 2); got != "line2" {
		t.Errorf("lineAt(2) = %q, want %q", got, "line2")
	}
	if got := lineAt(content, 99); got != "" {
		t.Errorf("lineAt(99) = %q, want empty", got)
	}
}

func TestWordAtPosition(t *testing.T) {
	tests := []struct {
		line string
		col  int
		want string
	}{
		{"    file.managed:", 8, "file.managed"},
		{"    - name: foo", 6, "name"},
		{"    - require: step one", 10, "require"},
		{"", 0, ""},
		{"  pkg.installed:", 5, "pkg.installed"},
	}
	for _, tt := range tests {
		got := wordAtPosition(tt.line, tt.col)
		if got != tt.want {
			t.Errorf("wordAtPosition(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
		}
	}
}
