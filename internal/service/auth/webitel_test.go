package auth

import (
	"slices"
	"testing"

	adpb "github.com/webitel/im-account-service/internal/client/webitel/proto/gen/auth"
)

func TestWebitelPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []*adpb.Permission
		want []string
	}{
		{name: "nil", in: nil, want: nil},
		{name: "empty", in: []*adpb.Permission{}, want: nil},
		{
			name: "ids only, empty skipped",
			in: []*adpb.Permission{
				{Id: "read", Name: "Select"},
				{Id: ""},
				nil,
				{Id: "im_read_all"},
			},
			want: []string{"read", "im_read_all"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := webitelPermissions(tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("webitelPermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}
