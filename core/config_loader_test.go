package core

import (
	// "HelmyTask/core"
	
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeName_Table(t *testing.T) {
	// GIVEN: table-driven inputs/outputs
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{"empty", "", ""},
		{"spaces-only", "   ", ""},
		{"single", "a", "A"},
		{"caps-ok", "Ahmed", "Ahmed"},
		{"mixed+spaces", "  aHMED  ", "AHMED"},
	}

	// WHEN/THEN: loop & assert
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeName(tc.in)
			assert.Equal(t, tc.out, got)
		})
	}
}
