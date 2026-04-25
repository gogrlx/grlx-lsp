package lsp

import (
	"context"
	"encoding/json"
	"strings"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
)

var semanticTokenTypes = []string{
	"namespace", // 0: ingredient name
	"function",  // 1: method name
	"property",  // 2: property key
	"string",    // 3: property value
	"keyword",   // 4: top-level keys (include, steps), requisite types
	"variable",  // 5: step ID
	"macro",     // 6: template expressions {{ }}
	"comment",   // 7: template comments {{/* */}}
}

func (h *Handler) handleSemanticTokensFull(_ context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	ctx := context.Background()

	var params protocol.SemanticTokensParams
	if err := json.Unmarshal(req.Params(), &params); err != nil {
		return reply(ctx, nil, err)
	}

	doc := h.getDocument(string(params.TextDocument.URI))
	if doc == nil || doc.recipe == nil {
		return reply(ctx, &protocol.SemanticTokens{}, nil)
	}

	tokens := h.computeSemanticTokens(doc)
	return reply(ctx, &protocol.SemanticTokens{Data: tokens}, nil)
}

func (h *Handler) computeSemanticTokens(doc *document) []uint32 {
	var raw []semanticToken
	r := doc.recipe

	if r.Root != nil {
		for i := 0; i+1 < len(r.Root.Content); i += 2 {
			keyNode := r.Root.Content[i]
			raw = append(raw, semanticToken{
				line:      keyNode.Line - 1,
				startChar: keyNode.Column - 1,
				length:    len(keyNode.Value),
				tokenType: 4, // keyword
			})
		}
	}

	for _, inc := range r.Includes {
		if inc.Node != nil {
			raw = append(raw, semanticToken{
				line:      inc.Node.Line - 1,
				startChar: inc.Node.Column - 1,
				length:    len(inc.Value),
				tokenType: 3, // string
			})
		}
	}

	for _, s := range r.Steps {
		if s.IDNode != nil {
			raw = append(raw, semanticToken{
				line:      s.IDNode.Line - 1,
				startChar: s.IDNode.Column - 1,
				length:    len(s.ID),
				tokenType: 5, // variable (step ID)
			})
		}

		if s.MethodNode != nil && strings.Contains(s.MethodNode.Value, ".") {
			dotIdx := strings.Index(s.MethodNode.Value, ".")
			col := s.MethodNode.Column - 1
			raw = append(raw, semanticToken{
				line:      s.MethodNode.Line - 1,
				startChar: col,
				length:    dotIdx,
				tokenType: 0, // namespace (ingredient)
			})
			raw = append(raw, semanticToken{
				line:      s.MethodNode.Line - 1,
				startChar: col + dotIdx + 1,
				length:    len(s.MethodNode.Value) - dotIdx - 1,
				tokenType: 1, // function (method)
			})
		}

		for _, p := range s.Properties {
			if p.KeyNode != nil {
				raw = append(raw, semanticToken{
					line:      p.KeyNode.Line - 1,
					startChar: p.KeyNode.Column - 1,
					length:    len(p.Key),
					tokenType: 2, // property
				})
			}
		}

		for _, req := range s.Requisites {
			if req.Node != nil {
				raw = append(raw, semanticToken{
					line:      req.Node.Line - 1,
					startChar: req.Node.Column - 1,
					length:    len(req.Condition),
					tokenType: 4, // keyword
				})
			}
		}
	}

	lines := strings.Split(doc.content, "\n")
	for i, line := range lines {
		for j := 0; j < len(line)-1; j++ {
			if line[j] == '{' && line[j+1] == '{' {
				end := strings.Index(line[j:], "}}")
				if end < 0 {
					continue
				}
				end += j + 2
				inner := line[j+2 : end-2]
				tokenType := 6 // macro
				trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(inner), "-"))
				if strings.HasPrefix(trimmed, "/*") {
					tokenType = 7 // comment
				}
				raw = append(raw, semanticToken{
					line:      i,
					startChar: j,
					length:    end - j,
					tokenType: tokenType,
				})
				j = end - 1
			}
		}
	}

	return encodeSemanticTokens(raw)
}

type semanticToken struct {
	line      int
	startChar int
	length    int
	tokenType int
}

func encodeSemanticTokens(tokens []semanticToken) []uint32 {
	sortTokens(tokens)
	data := make([]uint32, 0, len(tokens)*5)
	prevLine := 0
	prevChar := 0
	for _, t := range tokens {
		deltaLine := t.line - prevLine
		deltaChar := t.startChar
		if deltaLine == 0 {
			deltaChar = t.startChar - prevChar
		}
		data = append(data, uint32(deltaLine), uint32(deltaChar), uint32(t.length), uint32(t.tokenType), 0)
		prevLine = t.line
		prevChar = t.startChar
	}
	return data
}

func sortTokens(tokens []semanticToken) {
	for i := 1; i < len(tokens); i++ {
		key := tokens[i]
		j := i - 1
		for j >= 0 && (tokens[j].line > key.line || (tokens[j].line == key.line && tokens[j].startChar > key.startChar)) {
			tokens[j+1] = tokens[j]
			j--
		}
		tokens[j+1] = key
	}
}
