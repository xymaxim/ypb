package commands_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xymaxim/ypb/internal/commands"
)

func TestCommonFlagsValidate(t *testing.T) {
	t.Parallel()

	const portHint = "use --port"

	tests := []struct {
		name    string
		host    string
		wantErr string
	}{
		{"empty", "", ""},
		{"ipv4", "127.0.0.1", ""},
		{"ipv6", "::1", ""},
		{"host name", "localhost", ""},
		{"ipv4 with port", "127.0.0.1:9000", portHint},
		{"host name with port", "localhost:9000", portHint},
		{"ipv6 with port", "[::1]:9000", portHint},
		{"ipv6 in brackets", "[::1]", "leave out the brackets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&commands.CommonFlags{Host: tt.host}).Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
