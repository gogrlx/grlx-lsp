package lsp

import (
	"testing"

	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestBuildMethodMarkdown(t *testing.T) {
	m := &schema.Method{
		Name:        "managed",
		Description: "Download and manage a file from a source",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true, Description: "The file path"},
			{Key: "source", Type: "string", Required: true, Description: "Source URL"},
			{Key: "mode", Type: "string", Required: false, Description: "File permissions"},
		},
	}

	md := buildMethodMarkdown("file", m)

	// Should contain header
	if !contains(md, "### file.managed") {
		t.Error("expected markdown header with ingredient.method")
	}

	// Should contain description
	if !contains(md, "Download and manage a file") {
		t.Error("expected description in markdown")
	}

	// Should contain properties table
	if !contains(md, "| Property |") {
		t.Error("expected properties table")
	}
	if !contains(md, "| `name` |") {
		t.Error("expected name property in table")
	}
	if !contains(md, "| `source` |") {
		t.Error("expected source property in table")
	}

	// Required properties should show "yes"
	if !contains(md, "| yes |") {
		t.Error("expected 'yes' for required properties")
	}
}

func TestBuildMethodMarkdownNoProperties(t *testing.T) {
	m := &schema.Method{
		Name:        "cleaned",
		Description: "Clean package cache",
	}

	md := buildMethodMarkdown("pkg", m)
	if !contains(md, "### pkg.cleaned") {
		t.Error("expected header")
	}
	// Should not contain a table
	if contains(md, "| Property |") {
		t.Error("should not have property table for method with no properties")
	}
}

func TestBuildMethodMarkdownNoDescription(t *testing.T) {
	m := &schema.Method{
		Name: "test",
		Properties: []schema.Property{
			{Key: "name", Type: "string", Required: true},
		},
	}

	md := buildMethodMarkdown("foo", m)
	if !contains(md, "### foo.test") {
		t.Error("expected header")
	}
	if !contains(md, "| `name` |") {
		t.Error("expected name property")
	}
}

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		b    byte
		want bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'.', true},
		{'-', true},
		{' ', false},
		{':', false},
		{'\t', false},
		{'(', false},
	}
	for _, tt := range tests {
		got := isWordChar(tt.b)
		if got != tt.want {
			t.Errorf("isWordChar(%q) = %v, want %v", tt.b, got, tt.want)
		}
	}
}

func TestWordAtPositionEdgeCases(t *testing.T) {
	tests := []struct {
		line string
		col  int
		want string
	}{
		{"", 0, ""},
		{"hello", 100, "hello"},                // col beyond line length
		{"  file.managed:", 2, "file.managed"}, // col 2 is start of word
		{"file.managed:", 12, "file.managed"},  // just before colon
		{"a", 0, "a"},
		{"a", 1, "a"},
	}
	for _, tt := range tests {
		got := wordAtPosition(tt.line, tt.col)
		if got != tt.want {
			t.Errorf("wordAtPosition(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
		}
	}
}
