package lsp

import (
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
