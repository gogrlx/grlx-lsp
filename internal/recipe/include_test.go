package recipe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveIncludeRelativeFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "dev.grlx")
	os.WriteFile(target, []byte("steps: {}"), 0o644)
	current := filepath.Join(dir, "main.grlx")

	got, ok := ResolveInclude(dir, current, ".dev")
	if !ok {
		t.Fatal("expected to resolve .dev")
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveIncludeRelativeInitFile(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "apache")
	os.MkdirAll(sub, 0o755)
	target := filepath.Join(sub, "init.grlx")
	os.WriteFile(target, []byte("steps: {}"), 0o644)
	current := filepath.Join(dir, "main.grlx")

	got, ok := ResolveInclude(dir, current, ".apache")
	if !ok {
		t.Fatal("expected to resolve .apache")
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveIncludeAbsolute(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "webserver.grlx")
	os.WriteFile(target, []byte("steps: {}"), 0o644)
	current := filepath.Join(dir, "sub", "main.grlx")

	got, ok := ResolveInclude(dir, current, "webserver")
	if !ok {
		t.Fatal("expected to resolve webserver")
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}

func TestResolveIncludeAbsoluteInitPreferred(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "apache")
	os.MkdirAll(sub, 0o755)
	initTarget := filepath.Join(sub, "init.grlx")
	fileTarget := filepath.Join(dir, "apache.grlx")
	os.WriteFile(initTarget, []byte("steps: {}"), 0o644)
	os.WriteFile(fileTarget, []byte("steps: {}"), 0o644)
	current := filepath.Join(dir, "main.grlx")

	got, ok := ResolveInclude(dir, current, "apache")
	if !ok {
		t.Fatal("expected to resolve apache")
	}
	if got != initTarget {
		t.Errorf("got %q, want %q (init.grlx should be preferred)", got, initTarget)
	}
}

func TestResolveIncludeNotFound(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "main.grlx")

	_, ok := ResolveInclude(dir, current, ".nonexistent")
	if ok {
		t.Error("expected not found for .nonexistent")
	}
}

func TestResolveIncludeDottedAbsolute(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "nginx")
	os.MkdirAll(sub, 0o755)
	target := filepath.Join(sub, "vhost.grlx")
	os.WriteFile(target, []byte("steps: {}"), 0o644)
	current := filepath.Join(dir, "main.grlx")

	got, ok := ResolveInclude(dir, current, "nginx.vhost")
	if !ok {
		t.Fatal("expected to resolve nginx.vhost")
	}
	if got != target {
		t.Errorf("got %q, want %q", got, target)
	}
}
