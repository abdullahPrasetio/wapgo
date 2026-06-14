package commands

import (
	"testing"
)

func TestSemverGreater(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.4.1", "1.4.0", true},
		{"1.4.0", "1.4.0", false},
		{"1.4.0", "1.4.1", false},
		{"2.0.0", "1.9.9", true},
		{"1.10.0", "1.9.0", true},
		{"1.0.0", "2.0.0", false},
		{"1.4.1-rc1", "1.4.0", true},  // pre-release suffix stripped
		{"0.9.0", "1.0.0", false},
	}

	for _, tt := range tests {
		got := semverGreater(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("semverGreater(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSemverParts(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.4.1", [3]int{1, 4, 1}},
		{"2.0.0", [3]int{2, 0, 0}},
		{"1.4.1-rc1", [3]int{1, 4, 1}},
		{"1.4.1+meta", [3]int{1, 4, 1}},
		{"0.9", [3]int{0, 9, 0}},
	}

	for _, tt := range tests {
		got := semverParts(tt.input)
		if got != tt.want {
			t.Errorf("semverParts(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
