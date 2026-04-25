// Package recipe parses .grlx recipe files and provides structured access
// to their contents for diagnostics, completion, and hover.
package recipe

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Recipe represents a parsed .grlx file.
type Recipe struct {
	Root     *yaml.Node
	Includes []Include
	Steps    []Step
	Errors   []ParseError
}

// Include represents a single include directive.
type Include struct {
	Value string
	Node  *yaml.Node
}

// Step represents a single step in the recipe.
type Step struct {
	ID         string
	IDNode     *yaml.Node
	Ingredient string
	Method     string
	MethodNode *yaml.Node
	Properties []PropertyEntry
	Requisites []RequisiteEntry
}

// PropertyEntry is a key-value pair in a step's property list.
type PropertyEntry struct {
	Key       string
	Value     interface{}
	KeyNode   *yaml.Node
	ValueNode *yaml.Node
}

// RequisiteEntry is a parsed requisite condition.
type RequisiteEntry struct {
	Condition string
	StepIDs   []string
	Node      *yaml.Node
}

// ParseError represents a recipe parse error with location.
type ParseError struct {
	Message string
	Line    int
	Col     int
}

// templateBlockRe matches entire lines that are only Go template control flow
// (if, else, end, range, block, define, with, and comments).
var templateBlockRe = regexp.MustCompile(`^\s*\{\{-?\s*(?:/\*.*?\*/|(?:if|else|end|range|block|define|with|template)\b).*?-?\}\}\s*$`)

// templateExprRe matches inline Go template expressions like {{ props "key" }}.
var templateExprRe = regexp.MustCompile(`\{\{-?\s*.*?\s*-?\}\}`)

// StripTemplates pre-processes recipe text so that Go template directives do
// not break the YAML parser.  Control-flow lines are blanked (preserving line
// count) and inline expressions are replaced with a placeholder string.
func StripTemplates(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if templateBlockRe.MatchString(line) {
			lines[i] = ""
			continue
		}
		if strings.Contains(line, "{{") {
			lines[i] = templateExprRe.ReplaceAllString(line, "TMPL")
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

// Parse parses raw YAML bytes into a Recipe.  Go template directives are
// stripped before YAML parsing so that {{ }} expressions do not produce
// spurious errors.
func Parse(data []byte) *Recipe {
	r := &Recipe{}

	clean := StripTemplates(data)

	var doc yaml.Node
	if err := yaml.Unmarshal(clean, &doc); err != nil {
		r.Errors = append(r.Errors, ParseError{Message: "invalid YAML: " + err.Error(), Line: 0, Col: 0})
		return r
	}

	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return r
	}

	r.Root = doc.Content[0]

	if r.Root.Kind != yaml.MappingNode {
		r.Errors = append(r.Errors, ParseError{Message: "recipe must be a YAML mapping", Line: r.Root.Line, Col: r.Root.Column})
		return r
	}

	for i := 0; i+1 < len(r.Root.Content); i += 2 {
		keyNode := r.Root.Content[i]
		valNode := r.Root.Content[i+1]

		switch keyNode.Value {
		case "include":
			r.parseIncludes(valNode)
		case "steps":
			r.parseSteps(valNode)
		default:
			r.Errors = append(r.Errors, ParseError{
				Message: "unknown top-level key: " + keyNode.Value,
				Line:    keyNode.Line,
				Col:     keyNode.Column,
			})
		}
	}

	return r
}

func (r *Recipe) parseIncludes(node *yaml.Node) {
	if node.Kind != yaml.SequenceNode {
		r.Errors = append(r.Errors, ParseError{
			Message: "include must be a list",
			Line:    node.Line,
			Col:     node.Column,
		})
		return
	}
	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			r.Includes = append(r.Includes, Include{Value: item.Value, Node: item})
		}
	}
}

func (r *Recipe) parseSteps(node *yaml.Node) {
	if node.Kind != yaml.MappingNode {
		r.Errors = append(r.Errors, ParseError{
			Message: "steps must be a mapping",
			Line:    node.Line,
			Col:     node.Column,
		})
		return
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		stepIDNode := node.Content[i]
		stepBodyNode := node.Content[i+1]
		r.parseStep(stepIDNode, stepBodyNode)
	}
}

func (r *Recipe) parseStep(idNode, bodyNode *yaml.Node) {
	if bodyNode.Kind != yaml.MappingNode {
		r.Errors = append(r.Errors, ParseError{
			Message: "step body must be a mapping",
			Line:    bodyNode.Line,
			Col:     bodyNode.Column,
		})
		return
	}

	if len(bodyNode.Content) != 2 {
		r.Errors = append(r.Errors, ParseError{
			Message: "step must have exactly one ingredient.method key",
			Line:    bodyNode.Line,
			Col:     bodyNode.Column,
		})
		return
	}

	methodKeyNode := bodyNode.Content[0]
	methodValNode := bodyNode.Content[1]

	parts := strings.SplitN(methodKeyNode.Value, ".", 2)
	ingredient := ""
	method := ""
	if len(parts) == 2 {
		ingredient = parts[0]
		method = parts[1]
	} else {
		r.Errors = append(r.Errors, ParseError{
			Message: "step key must be in the form ingredient.method, got: " + methodKeyNode.Value,
			Line:    methodKeyNode.Line,
			Col:     methodKeyNode.Column,
		})
	}

	step := Step{
		ID:         idNode.Value,
		IDNode:     idNode,
		Ingredient: ingredient,
		Method:     method,
		MethodNode: methodKeyNode,
	}

	// The value should be a sequence of mappings (property list)
	if methodValNode.Kind == yaml.SequenceNode {
		for _, item := range methodValNode.Content {
			if item.Kind == yaml.MappingNode {
				for j := 0; j+1 < len(item.Content); j += 2 {
					k := item.Content[j]
					v := item.Content[j+1]
					if k.Value == "requisites" {
						step.Requisites = parseRequisites(v)
					} else {
						step.Properties = append(step.Properties, PropertyEntry{
							Key:       k.Value,
							KeyNode:   k,
							ValueNode: v,
						})
					}
				}
			}
		}
	}

	r.Steps = append(r.Steps, step)
}

func parseRequisites(node *yaml.Node) []RequisiteEntry {
	var reqs []RequisiteEntry
	if node.Kind != yaml.SequenceNode {
		return reqs
	}
	for _, item := range node.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			condition := item.Content[i].Value
			valNode := item.Content[i+1]
			var stepIDs []string
			switch valNode.Kind {
			case yaml.ScalarNode:
				stepIDs = append(stepIDs, valNode.Value)
			case yaml.SequenceNode:
				for _, s := range valNode.Content {
					if s.Kind == yaml.ScalarNode {
						stepIDs = append(stepIDs, s.Value)
					}
				}
			}
			reqs = append(reqs, RequisiteEntry{
				Condition: condition,
				StepIDs:   stepIDs,
				Node:      item.Content[i],
			})
		}
	}
	return reqs
}

// StepIDs returns all step IDs defined in this recipe.
func (r *Recipe) StepIDs() []string {
	var ids []string
	for _, s := range r.Steps {
		ids = append(ids, s.ID)
	}
	return ids
}
