package sinks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExplodeJSONStr(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		separator string
		expectErr bool
	}{
		{
			name:      "simple nested JSON",
			input:     `{"person":{"name":"Joe", "address":{"street":"123 Main St."}}}`,
			expected:  `{"person.address.street":"123 Main St.","person.name":"Joe"}`,
			separator: ".",
			expectErr: false,
		},
		{
			name:      "single level JSON",
			input:     `{"name":"Joe", "age":30}`,
			expected:  `{"age":30,"name":"Joe"}`,
			separator: ".",
			expectErr: false,
		},
		{
			name:      "empty JSON",
			input:     `{}`,
			expected:  `{}`,
			separator: ".",
			expectErr: false,
		},
		{
			name:      "invalid JSON",
			input:     `{"name": "Joe", "age":}`,
			expected:  ``,
			separator: ".",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := explodeJSONStr(tt.input, tt.separator)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.JSONEq(t, tt.expected, output)
			}
		})
	}
}
