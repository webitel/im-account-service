package model

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/webitel/im-account-service/proto/gen/im/service/admin/v1"
)

const (
	contactA = "0c9a9b1e-7d5f-4c1a-9b8e-3f2a1d4c5e6a"
	contactB = "6f1e2d3c-4b5a-4987-8a7b-6c5d4e3f2a1b"
	contactC = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
)

func TestValidateContactAllowList(t *testing.T) {
	tests := []struct {
		name    string
		list    *v1.ContactAllowList
		wantErr bool
	}{
		{
			name:    "nil list is valid (no restriction)",
			list:    nil,
			wantErr: false,
		},
		{
			name:    "empty list is valid (nobody allowed)",
			list:    &v1.ContactAllowList{},
			wantErr: false,
		},
		{
			name: "one default and several distinct rules",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: contactA, Default: true},
				{ContactId: contactB},
				{ContactId: contactC},
			}},
			wantErr: false,
		},
		{
			name: "upper-case valid UUID is accepted",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: "0C9A9B1E-7D5F-4C1A-9B8E-3F2A1D4C5E6A"},
			}},
			wantErr: false,
		},
		{
			name: "two default rules",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: contactA, Default: true},
				{ContactId: contactB, Default: true},
			}},
			wantErr: true,
		},
		{
			name: "duplicate contact_id",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: contactA},
				{ContactId: contactA},
			}},
			wantErr: true,
		},
		{
			name: "initialized structure with empty id",
			list: &v1.ContactAllowList{
				Rules: []*v1.ContactAllowRule{
					{},
					{ContactId: contactB},
					{ContactId: contactC},
				},
			},
			wantErr: true,
		},
		{
			name: "no default non empty list",

			list: &v1.ContactAllowList{
				Rules: []*v1.ContactAllowRule{
					{ContactId: contactA},
					{ContactId: contactB},
					{ContactId: contactC},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate contact_id differing only by case",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: contactA},
				{ContactId: "0C9A9B1E-7D5F-4C1A-9B8E-3F2A1D4C5E6A"},
			}},
			wantErr: true,
		},
		{
			name: "not a UUID",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: "not-a-uuid"},
			}},
			wantErr: true,
		},
		{
			name: "empty contact_id",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: ""},
			}},
			wantErr: true,
		},
		{
			name: "32-char hex without hyphens is rejected",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: "0c9a9b1e7d5f4c1a9b8e3f2a1d4c5e6a"},
			}},
			wantErr: true,
		},
		{
			name: "urn:uuid prefix is rejected",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: "urn:uuid:" + contactA},
			}},
			wantErr: true,
		},
		{
			name: "braced UUID is rejected",
			list: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
				{ContactId: "{" + contactA + "}"},
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContactAllowList(tt.list)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateContactAllowList() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if code := status.Code(err); code != codes.InvalidArgument {
					t.Errorf("status code = %v, want %v", code, codes.InvalidArgument)
				}
			}
		})
	}
}

func TestValidateInputApp(t *testing.T) {
	if err := ValidateInputApp(nil); err != nil {
		t.Errorf("ValidateInputApp(nil) error = %v, want nil", err)
	}

	if err := ValidateInputApp(&v1.InputApp{}); err != nil {
		t.Errorf("ValidateInputApp(empty) error = %v, want nil", err)
	}

	bad := &v1.InputApp{AllowContacts: &v1.ContactAllowList{Rules: []*v1.ContactAllowRule{
		{ContactId: contactA, Default: true},
		{ContactId: contactB, Default: true},
	}}}
	if err := ValidateInputApp(bad); status.Code(err) != codes.InvalidArgument {
		t.Errorf("ValidateInputApp(two defaults) code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}
