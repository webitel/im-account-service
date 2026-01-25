package contacts

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/simplelru"
	impb "github.com/webitel/im-account-service/proto/gen/im/service/contact/v1"

	// adv1 "github.com/webitel/im-account-service/proto/gen/im/shared/contact/v1"
	"google.golang.org/grpc"
)

type Contact = impb.Contact

type ContactsClient struct {
	impb.ContactsClient
	cache simplelru.LRUCache[string, *Contact] // TODO
}

var _ impb.ContactsClient = (*ContactsClient)(nil)

func (c *ContactsClient) SearchContact(ctx context.Context, in *impb.SearchContactRequest, opts ...grpc.CallOption) (*impb.ContactList, error) {
	return c.ContactsClient.SearchContact(ctx, in, opts...)
}

func (c *ContactsClient) CreateContact(ctx context.Context, in *impb.CreateContactRequest, opts ...grpc.CallOption) (*impb.Contact, error) {
	return c.ContactsClient.CreateContact(ctx, in, opts...)
}

func (c *ContactsClient) UpdateContact(ctx context.Context, in *impb.UpdateContactRequest, opts ...grpc.CallOption) (*impb.Contact, error) {
	return c.ContactsClient.UpdateContact(ctx, in, opts...)
}

func (c *ContactsClient) DeleteContact(ctx context.Context, in *impb.DeleteContactRequest, opts ...grpc.CallOption) (*impb.Contact, error) {
	return c.ContactsClient.DeleteContact(ctx, in, opts...)
}

func (c *ContactsClient) CanSend(ctx context.Context, in *impb.CanSendRequest, opts ...grpc.CallOption) (*impb.CanSendResponse, error) {
	return c.ContactsClient.CanSend(ctx, in, opts...)
}
