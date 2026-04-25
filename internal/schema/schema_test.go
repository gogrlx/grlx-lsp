package schema

import "testing"

func TestDefaultRegistry(t *testing.T) {
	r := DefaultRegistry()

	if len(r.Ingredients) == 0 {
		t.Fatal("expected at least one ingredient")
	}

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

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	want := []string{
		"file.managed", "cmd.run", "pkg.installed", "pkg.upgraded",
		"pkg.uptodate", "pkg.unheld", "service.running", "service.reloaded",
		"user.present", "group.present",
	}
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

func TestPkgMethods(t *testing.T) {
	r := DefaultRegistry()
	ing := r.FindIngredient("pkg")
	if ing == nil {
		t.Fatal("missing pkg ingredient")
	}

	expectedMethods := []string{
		"cleaned", "group_installed", "held", "installed",
		"key_managed", "latest", "purged", "removed",
		"repo_managed", "unheld", "upgraded", "uptodate",
	}
	methodSet := make(map[string]bool)
	for _, m := range ing.Methods {
		methodSet[m.Name] = true
	}
	for _, name := range expectedMethods {
		if !methodSet[name] {
			t.Errorf("pkg ingredient missing method: %s", name)
		}
	}
}

func TestServiceMethods(t *testing.T) {
	r := DefaultRegistry()
	ing := r.FindIngredient("service")
	if ing == nil {
		t.Fatal("missing service ingredient")
	}

	expectedMethods := []string{
		"disabled", "enabled", "masked", "reloaded",
		"restarted", "running", "stopped", "unmasked",
	}
	methodSet := make(map[string]bool)
	for _, m := range ing.Methods {
		methodSet[m.Name] = true
	}
	for _, name := range expectedMethods {
		if !methodSet[name] {
			t.Errorf("service ingredient missing method: %s", name)
		}
	}
}

func TestRequiredProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("file", "managed")
	if m == nil {
		t.Fatal("missing file.managed")
	}

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

func TestCmdRunProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("cmd", "run")
	if m == nil {
		t.Fatal("missing cmd.run")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	for _, key := range []string{"name", "args", "runas", "cwd", "env", "path", "timeout", "shell", "creates", "unless", "onlyif"} {
		if _, ok := propMap[key]; !ok {
			t.Errorf("cmd.run missing property: %s", key)
		}
	}
	if !propMap["name"].Required {
		t.Error("cmd.run: name should be required")
	}
}

func TestPkgInstalledProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("pkg", "installed")
	if m == nil {
		t.Fatal("missing pkg.installed")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	for _, key := range []string{"name", "version", "fromrepo", "pkgs", "refresh", "reinstall"} {
		if _, ok := propMap[key]; !ok {
			t.Errorf("pkg.installed missing property: %s", key)
		}
	}
}

func TestFileTouchProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("file", "touch")
	if m == nil {
		t.Fatal("missing file.touch")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	for _, key := range []string{"name", "atime", "mtime", "makedirs"} {
		if _, ok := propMap[key]; !ok {
			t.Errorf("file.touch missing property: %s", key)
		}
	}
}

func TestUserPresentProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("user", "present")
	if m == nil {
		t.Fatal("missing user.present")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	for _, key := range []string{"name", "uid", "gid", "home", "shell", "groups", "comment", "createhome", "system", "password_hash"} {
		if _, ok := propMap[key]; !ok {
			t.Errorf("user.present missing property: %s", key)
		}
	}
}

func TestGroupPresentProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("group", "present")
	if m == nil {
		t.Fatal("missing group.present")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	for _, key := range []string{"name", "gid", "system", "members"} {
		if _, ok := propMap[key]; !ok {
			t.Errorf("group.present missing property: %s", key)
		}
	}
}

func TestUserAbsentProperties(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("user", "absent")
	if m == nil {
		t.Fatal("missing user.absent")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	if _, ok := propMap["purge"]; !ok {
		t.Error("user.absent missing property: purge")
	}
}

func TestFileContainsSourceNotRequired(t *testing.T) {
	r := DefaultRegistry()
	m := r.FindMethod("file", "contains")
	if m == nil {
		t.Fatal("missing file.contains")
	}

	propMap := make(map[string]Property)
	for _, p := range m.Properties {
		propMap[p.Key] = p
	}

	if p, ok := propMap["source"]; ok && p.Required {
		t.Error("file.contains: source should be optional")
	}
}
