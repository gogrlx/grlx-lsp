package schema

import "testing"

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()

	if len(r.Ingredients) == 0 {
		t.Fatal("expected at least one ingredient")
	}

	// Verify all expected ingredients are present
	expected := []string{"cmd", "file", "group", "pkg", "service", "user"}
	for _, name := range expected {
		if r.FindIngredient(name) == nil {
			t.Errorf("missing expected ingredient: %s", name)
		}
	}
}

func TestFindIngredient(t *testing.T) {
	r := DefaultRegistry()

	ing := r.FindIngredient("file")
	if ing == nil {
		t.Fatal("expected to find file ingredient")
	}
	if ing.Name != "file" {
		t.Errorf("got name %q, want %q", ing.Name, "file")
	}

	if r.FindIngredient("nonexistent") != nil {
		t.Error("expected nil for nonexistent ingredient")
	}
}

func TestFindMethod(t *testing.T) {
	r := DefaultRegistry()

	m := r.FindMethod("file", "managed")
	if m == nil {
		t.Fatal("expected to find file.managed")
	}
	if m.Name != "managed" {
		t.Errorf("got method %q, want %q", m.Name, "managed")
	}

	if r.FindMethod("file", "nonexistent") != nil {
		t.Error("expected nil for nonexistent method")
	}
	if r.FindMethod("nonexistent", "managed") != nil {
		t.Error("expected nil for nonexistent ingredient")
	}
}

func TestAllDottedNames(t *testing.T) {
	r := DefaultRegistry()
	names := r.AllDottedNames()

	if len(names) == 0 {
		t.Fatal("expected at least one dotted name")
	}

	// Check that some known names are present
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	want := []string{"file.managed", "cmd.run", "pkg.installed", "service.running", "user.present", "group.present"}
	for _, w := range want {
		if !nameSet[w] {
			t.Errorf("missing expected dotted name: %s", w)
		}
	}
}

func TestFileMethods(t *testing.T) {
	r := DefaultRegistry()
	ing := r.FindIngredient("file")
	if ing == nil {
		t.Fatal("missing file ingredient")
	}

	expectedMethods := []string{
		"absent", "append", "cached", "contains", "content",
		"directory", "exists", "managed", "missing", "prepend",
		"symlink", "touch",
	}
	methodSet := make(map[string]bool)
	for _, m := range ing.Methods {
		methodSet[m.Name] = true
	}
	for _, name := range expectedMethods {
		if !methodSet[name] {
			t.Errorf("file ingredient missing method: %s", name)
		}
	}
}

func TestRequiredProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("file", "managed")
	if m == nil {
		t.Fatal("missing file.managed")
	}

	// name and source should be required
	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	if p, ok := propMap["name"]; !ok || !p.Required {
		t.Error("file.managed: name should be required")
	}
	if p, ok := propMap["source"]; !ok || !p.Required {
		t.Error("file.managed: source should be required")
	}
	if p, ok := propMap["user"]; !ok || p.Required {
		t.Error("file.managed: user should be optional")
	}
}
