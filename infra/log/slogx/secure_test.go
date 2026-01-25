package slogx_test

import (
	"testing"

	"github.com/webitel/im-account-service/infra/log/slogx"
)

func TestSecureString(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		raw  string
		opts []slogx.SecureStringOption
		want string
	}{
		// TODO: Add test cases.
		{
			name: "short",
			raw:  "short",
			opts: []slogx.SecureStringOption{},
			want: "########hort",
		},
		{
			name: "general",
			raw:  "my_secret_token_to_be_hidden",
			opts: []slogx.SecureStringOption{},
			want: "my_secre########dden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slogx.SecureString(tt.raw, tt.opts...)
			if got != tt.want {
				t.Errorf("SecureString() = %v, want %v", got, tt.want)
			}
		})
	}
}
