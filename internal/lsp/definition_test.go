package lsp

import (
	"fmt"
	"strings"
	"testing"
)

func TestStepRefAtPosition(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want string
	}{
		{
			name: "require ref",
			line: "        - require: install nginx",
			col:  25,
			want: "install nginx",
		},
		{
			name: "cursor before value",
			line: "        - require: install nginx",
			col:  10,
			want: "",
		},
		{
			name: "onchanges ref",
			line: "        - onchanges: build app",
			col:  25,
			want: "build app",
		},
		{
			name: "no match",
			line: "      - name: foo",
			col:  10,
			want: "",
		},
		{
			name: "empty value",
			line: "        - require: ",
			col:  19,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepRefAtPosition(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("stepRefAtPosition(%q, %d) = %q, want %q", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestRequisiteValueAtPositionSequenceItem(t *testing.T) {
	content := `steps:
  first:
    file.exists:
      - name: /tmp/a
  second:
    file.exists:
      - name: /tmp/b
      - requisites:
        - require:
          - first`

	got, ok := requisiteValueAtPosition(content, 9, 12)
	if !ok || got != "first" {
		t.Fatalf("requisiteValueAtPosition() = %q, %v, want %q, true", got, ok, "first")
	}
}

func TestRequisiteValueAtPositionLongSequence(t *testing.T) {
	content := `steps:
  target:
    file.exists:
      - name: /tmp/target
  consumer:
    file.exists:
      - requisites:
        - require:
` + strings.Repeat("          - target\n", 12)

	for _, line := range []int{8, 16, 19} {
		t.Run(fmt.Sprintf("line_%d", line), func(t *testing.T) {
			ref, ok := requisiteValueAtPosition(content, line, 14)
			if !ok || ref != "target" {
				t.Errorf("reference = %q, %v; want target, true", ref, ok)
			}
			if !isRequisiteValuePosition(content, line, 14) {
				t.Fatal("step-ID completion must remain available throughout the list")
			}
			if _, ok := requisiteValueAtPosition(content, line, 10); ok {
				t.Fatal("list marker must not be treated as a reference")
			}
		})
	}
}

func TestRequisiteLookupStopsAtFollowingStep(t *testing.T) {
	content := `steps:
  consumer:
    file.exists:
      - requisites:
        - require:
` + strings.Repeat("          - target\n", 12) + `  following:
    file.exists:
      - name: /tmp/following
      - other:
          - target`
	line := 21
	if ref, ok := requisiteValueAtPosition(content, line, 14); ok {
		t.Fatalf("unrelated property resolved as requisite %q", ref)
	}
	if isRequisiteValuePosition(content, line, 14) {
		t.Fatal("unrelated property must not receive step-ID completions")
	}
}
