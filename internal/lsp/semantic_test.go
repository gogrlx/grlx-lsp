package lsp

import (
	"testing"

	"github.com/gogrlx/grlx-lsp/internal/recipe"
	"github.com/gogrlx/grlx-lsp/internal/schema"
)

func TestComputeSemanticTokensBasic(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  install nginx:
    pkg.installed:
      - name: nginx`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	tokens := h.computeSemanticTokens(doc)
	if len(tokens) == 0 {
		t.Fatal("expected semantic tokens, got none")
	}
	if len(tokens)%5 != 0 {
		t.Fatalf("token data length %d is not a multiple of 5", len(tokens))
	}
}

func TestComputeSemanticTokensTemplate(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `steps:
  deploy config:
    file.managed:
      - name: /etc/app/config.yaml
      - user: {{ props "app_user" }}
      - mode: "644"`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	tokens := h.computeSemanticTokens(doc)
	if len(tokens) == 0 {
		t.Fatal("expected semantic tokens")
	}

	numTokens := len(tokens) / 5
	foundMacro := false
	for i := range numTokens {
		tokenType := tokens[i*5+3]
		if tokenType == 6 {
			foundMacro = true
		}
	}
	if !foundMacro {
		t.Error("expected at least one macro token for template expression")
	}
}

func TestComputeSemanticTokensComment(t *testing.T) {
	h := NewHandler(schema.DefaultRegistry())
	src := `{{/* comment */}}
steps:
  my step:
    file.exists:
      - name: /tmp/a`
	doc := &document{
		content: src,
		recipe:  recipe.Parse([]byte(src)),
	}

	tokens := h.computeSemanticTokens(doc)
	numTokens := len(tokens) / 5
	foundComment := false
	for i := range numTokens {
		tokenType := tokens[i*5+3]
		if tokenType == 7 {
			foundComment = true
		}
	}
	if !foundComment {
		t.Error("expected at least one comment token for template comment")
	}
}

func TestSortTokens(t *testing.T) {
	tokens := []semanticToken{
		{line: 2, startChar: 0, length: 3, tokenType: 0},
		{line: 0, startChar: 5, length: 2, tokenType: 1},
		{line: 0, startChar: 0, length: 4, tokenType: 2},
	}
	sortTokens(tokens)

	if tokens[0].line != 0 || tokens[0].startChar != 0 {
		t.Errorf("first token should be (0,0), got (%d,%d)", tokens[0].line, tokens[0].startChar)
	}
	if tokens[1].line != 0 || tokens[1].startChar != 5 {
		t.Errorf("second token should be (0,5), got (%d,%d)", tokens[1].line, tokens[1].startChar)
	}
	if tokens[2].line != 2 {
		t.Errorf("third token should be on line 2, got %d", tokens[2].line)
	}
}

func TestEncodeSemanticTokens(t *testing.T) {
	tokens := []semanticToken{
		{line: 0, startChar: 0, length: 5, tokenType: 4},
		{line: 1, startChar: 2, length: 3, tokenType: 5},
	}
	data := encodeSemanticTokens(tokens)
	if len(data) != 10 {
		t.Fatalf("expected 10 values, got %d", len(data))
	}
	// First: deltaLine=0, deltaChar=0, length=5, type=4, mod=0
	if data[0] != 0 || data[1] != 0 || data[2] != 5 || data[3] != 4 {
		t.Errorf("first token encoding wrong: %v", data[:5])
	}
	// Second: deltaLine=1, deltaChar=2, length=3, type=5, mod=0
	if data[5] != 1 || data[6] != 2 || data[7] != 3 || data[8] != 5 {
		t.Errorf("second token encoding wrong: %v", data[5:10])
	}
}
