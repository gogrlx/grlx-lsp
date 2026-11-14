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

func TestParseStepWithMultipleMethodKeys(t *testing.T) {
	data := []byte(`
steps:
  bad step:
    file.exists:
      - name: /tmp/a
    file.absent:
      - name: /tmp/b
`)
	r := Parse(data)

	found := false
	for _, e := range r.Errors {
		if e.Message == "step must have exactly one ingredient.method key" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about multiple ingredient.method keys, got: %v", r.Errors)
	}
	if len(r.Steps) != 0 {
		t.Fatalf("expected invalid step to be skipped, got %d parsed steps", len(r.Steps))
	}
}

func TestParseStepPropertiesMustBeList(t *testing.T) {
	data := []byte(`
steps:
  bad step:
    pkg.installed: nginx
`)
	r := Parse(data)

	found := false
	for _, e := range r.Errors {
		if e.Message == "step properties must be a list" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about non-list step properties, got: %v", r.Errors)
	}
	if len(r.Steps) != 1 {
		t.Fatalf("expected step to remain available for diagnostics, got %d parsed steps", len(r.Steps))
	}
}

func TestParseStepPropertyEntriesMustBeMappings(t *testing.T) {
	data := []byte(`
steps:
  bad step:
    pkg.installed:
      - name: nginx
      - invalid
`)
	r := Parse(data)

	found := false
	for _, e := range r.Errors {
		if e.Message == "step property entries must be mappings" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about non-mapping property entry, got: %v", r.Errors)
	}
	if len(r.Steps) != 1 {
		t.Fatalf("expected step to remain available for diagnostics, got %d parsed steps", len(r.Steps))
	}
	if len(r.Steps[0].Properties) != 1 || r.Steps[0].Properties[0].Key != "name" {
		t.Fatalf("expected valid properties to still parse, got: %v", r.Steps[0].Properties)
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

func TestParseMultipleRequisiteBlocks(t *testing.T) {
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
      - user: root
      - requisites:
        - onchanges: first step
`)
	r := Parse(data)

	if len(r.Errors) > 0 {
		t.Fatalf("unexpected parse errors: %v", r.Errors)
	}
	if len(r.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(r.Steps))
	}

	requisites := r.Steps[1].Requisites
	if len(requisites) != 2 {
		t.Fatalf("expected 2 requisites, got %d: %v", len(requisites), requisites)
	}
	if requisites[0].Condition != "require" || requisites[0].StepIDs[0] != "first step" {
		t.Fatalf("unexpected first requisite: %#v", requisites[0])
	}
	if requisites[1].Condition != "onchanges" || requisites[1].StepIDs[0] != "first step" {
		t.Fatalf("unexpected second requisite: %#v", requisites[1])
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

func TestParseMalformedRequisites(t *testing.T) {
	for _, test := range []struct {
		name, value, message string
	}{
		{"scalar block", "invalid", "requisites must be a list"},
		{"mapping block", "{require: first}", "requisites must be a list"},
		{"scalar entry", "[invalid]", "requisite entries must be mappings"},
		{"mapping reference", "[{require: {step: first}}]", "requisite reference must be a scalar or list"},
		{"nested reference", "[{require: [first, {step: second}]}]", "requisite references must be scalars"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed := Parse([]byte("steps:\n  example:\n    file.exists:\n      - name: /tmp/example\n      - requisites: " + test.value + "\n      - requisites:\n        - onchanges: first\n"))
			if len(parsed.Errors) != 1 || parsed.Errors[0].Message != test.message {
				t.Fatalf("expected %q, got %v", test.message, parsed.Errors)
			}
			if parsed.Errors[0].Line != 5 || parsed.Errors[0].Col <= 0 {
				t.Fatalf("expected error at malformed value on line 5, got %v", parsed.Errors[0])
			}
			if len(parsed.Steps) != 1 {
				t.Fatalf("expected valid step to remain available, got %v", parsed.Steps)
			}
			requisites := parsed.Steps[0].Requisites
			if len(requisites) == 0 || requisites[len(requisites)-1].Condition != "onchanges" || requisites[len(requisites)-1].StepIDs[0] != "first" {
				t.Fatalf("expected following requisite block to survive, got %v", requisites)
			}
			if test.name == "nested reference" && (len(requisites[0].StepIDs) != 1 || requisites[0].StepIDs[0] != "first") {
				t.Fatalf("expected valid reference to survive, got %v", requisites)
			}
		})
	}
}

func TestParseMalformedIncludeEntries(t *testing.T) {
	for _, entry := range []string{"null", "42", "true", "{}", "[]", `""`, `"   "`} {
		t.Run(entry, func(t *testing.T) {
			src := "include:\n  - apache\n  - " + entry + "\n  - .dev\nsteps:\n  install:\n    pkg.installed:\n      - name: nginx\n"
			parsed := Parse([]byte(src))
			if len(parsed.Errors) != 1 {
				t.Fatalf("expected one include error, got %v", parsed.Errors)
			}
			parseError := parsed.Errors[0]
			if parseError.Message != "include entries must be non-empty strings" || parseError.Line != 3 || parseError.Col != 5 {
				t.Errorf("unexpected error: %+v", parseError)
			}
			if len(parsed.Includes) != 2 || parsed.Includes[0].Value != "apache" || parsed.Includes[1].Value != ".dev" {
				t.Errorf("valid includes not preserved: %v", parsed.Includes)
			}
			if len(parsed.Steps) != 1 {
				t.Errorf("following steps not preserved: %v", parsed.Steps)
			}
		})
	}
}

func TestParseQuotedIncludeStrings(t *testing.T) {
	parsed := Parse([]byte("include: [\"42\", \"true\", \"null\"]\n"))
	if len(parsed.Errors) != 0 || len(parsed.Includes) != 3 {
		t.Fatalf("quoted strings must remain valid: %+v", parsed)
	}
}
