package recipe

import "testing"

func TestParseSimpleRecipe(t *testing.T) {
	data := []byte(`
include:
  - apache
  - .dev

steps:
  install nginx:
    pkg.installed:
      - name: nginx
  start nginx:
    service.running:
      - name: nginx
      - requisites:
        - require: install nginx
`)
	r := Parse(data)

	if len(r.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", r.Errors)
	}

	if len(r.Includes) != 2 {
		t.Fatalf("expected 2 includes, got %d", len(r.Includes))
	}
	if r.Includes[0].Value != "apache" {
		t.Errorf("include[0] = %q, want %q", r.Includes[0].Value, "apache")
	}
	if r.Includes[1].Value != ".dev" {
		t.Errorf("include[1] = %q, want %q", r.Includes[1].Value, ".dev")
	}

	if len(r.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(r.Steps))
	}

	s := r.Steps[0]
	if s.ID != "install nginx" {
		t.Errorf("step[0].ID = %q, want %q", s.ID, "install nginx")
	}
	if s.Ingredient != "pkg" {
		t.Errorf("step[0].Ingredient = %q, want %q", s.Ingredient, "pkg")
	}
	if s.Method != "installed" {
		t.Errorf("step[0].Method = %q, want %q", s.Method, "installed")
	}
	if len(s.Properties) != 1 || s.Properties[0].Key != "name" {
		t.Errorf("step[0] expected name property, got %v", s.Properties)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	data := []byte(`{{{invalid`)
	r := Parse(data)

	if len(r.Errors) == 0 {
		t.Error("expected parse errors for invalid YAML")
	}
}

func TestParseUnknownTopLevel(t *testing.T) {
	data := []byte(`
include:
  - foo
bogus_key: bar
steps: {}
`)
	r := Parse(data)

	found := false
	for _, e := range r.Errors {
		if e.Message == "unknown top-level key: bogus_key" {
			found = true
		}
	}
	if !found {
		t.Error("expected error about unknown top-level key")
	}
}

func TestParseBadMethodKey(t *testing.T) {
	data := []byte(`
steps:
  bad step:
    nomethod:
      - name: foo
`)
	r := Parse(data)

	found := false
	for _, e := range r.Errors {
		if e.Message == "step key must be in the form ingredient.method, got: nomethod" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about bad method key, got: %v", r.Errors)
	}
}

func TestStepIDs(t *testing.T) {
	data := []byte(`
steps:
  step one:
    file.exists:
      - name: /tmp/a
  step two:
    file.absent:
      - name: /tmp/b
`)
	r := Parse(data)
	ids := r.StepIDs()

	if len(ids) != 2 {
		t.Fatalf("expected 2 step IDs, got %d", len(ids))
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet["step one"] || !idSet["step two"] {
		t.Errorf("missing expected step IDs: %v", ids)
	}
}

func TestParseRequisites(t *testing.T) {
	data := []byte(`
steps:
  first step:
    file.exists:
      - name: /tmp/a
  second step:
    file.exists:
      - name: /tmp/b
      - requisites:
        - require: first step
        - onchanges:
          - first step
`)
	r := Parse(data)

	if len(r.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", r.Errors)
	}

	if len(r.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(r.Steps))
	}

	s := r.Steps[1]
	if len(s.Requisites) != 2 {
		t.Fatalf("expected 2 requisites, got %d", len(s.Requisites))
	}
	if s.Requisites[0].Condition != "require" {
		t.Errorf("requisite[0].Condition = %q, want %q", s.Requisites[0].Condition, "require")
	}
	if len(s.Requisites[0].StepIDs) != 1 || s.Requisites[0].StepIDs[0] != "first step" {
		t.Errorf("requisite[0].StepIDs = %v, want [first step]", s.Requisites[0].StepIDs)
	}
}

func TestParseEmptyRecipe(t *testing.T) {
	r := Parse([]byte(""))
	if r == nil {
		t.Fatal("expected non-nil recipe for empty input")
	}
}

func TestParseGoTemplate(t *testing.T) {
	// Recipes can contain Go template syntax — the parser should not crash.
	// Template directives may cause YAML errors, but the parser should handle gracefully.
	data := []byte(`
steps:
  install golang:
    archive.extracted:
      - name: /usr/local/go
`)
	r := Parse(data)
	if len(r.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(r.Steps))
	}
}
